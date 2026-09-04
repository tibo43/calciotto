package services_test

import (
	"errors"
	"testing"
	"time"

	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// voteEnv is the fixture every test below starts from: a group with its two
// default teams, a match with a composed roster (so there is something to
// vote for), and a handful of extra players not on that roster. Like
// registrationEnv, players are not added to the group's membership on
// purpose — MatchVoteService knows nothing about membership, that check
// belongs to the route's middleware (see the service's own doc comment).
type voteEnv struct {
	tx      *gorm.DB
	votes   *services.MatchVoteService
	matches *services.MatchService
	groupID uuid.UUID
	matchID uuid.UUID
	// roster holds the players actually placed on a team for matchID —
	// the only valid candidates to vote for.
	roster []uuid.UUID
	// bench holds players that exist but were never added to the match's
	// roster — used to exercise ErrVotedForPlayerNotOnRoster.
	bench []uuid.UUID
}

// newVoteEnv creates a group, a match, rosterSize players placed on the
// match's black team, and benchSize further players who exist but are never
// placed on any team.
func newVoteEnv(t *testing.T, label string, rosterSize, benchSize int) *voteEnv {
	t.Helper()

	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	teamService := services.NewTeamService(tx)
	playerService := services.NewPlayerService(tx)
	matchService := services.NewMatchService(tx)

	group, err := groupService.CreateGroup("Zzz Votes "+label, services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	teams, err := teamService.GetTeamsByGroupID(group.ID)
	if err != nil {
		t.Fatalf("failed to load group's teams: %v", err)
	}
	if len(teams) != 2 {
		t.Fatalf("expected CreateGroup to create exactly 2 teams, got %d", len(teams))
	}
	black := teams[0]

	matchID, err := matchService.CreateMatch(services.MatchSpec{Date: models.Date(time.Now())}, group.ID)
	if err != nil {
		t.Fatalf("failed to create match: %v", err)
	}

	env := &voteEnv{
		tx:      tx,
		votes:   services.NewMatchVoteService(tx),
		matches: matchService,
		groupID: group.ID,
		matchID: matchID,
	}

	rosterPlayers := make([]models.PlayerCustom, 0, rosterSize)
	for i := 0; i < rosterSize; i++ {
		id, err := playerService.CreatePlayer("Zzz Vote Roster " + label + " " + uuid.NewString())
		if err != nil {
			t.Fatalf("failed to create roster player %d: %v", i, err)
		}
		env.roster = append(env.roster, id)
		rosterPlayers = append(rosterPlayers, models.PlayerCustom{ID: id, GoalsScored: 0})
	}
	if rosterSize > 0 {
		if err := matchService.UpdateMatch(models.MatchWithDetails{
			ID: matchID,
			Teams: []models.TeamWithPlayers{
				{ID: black.ID, Players: rosterPlayers},
			},
		}); err != nil {
			t.Fatalf("failed to compose the roster: %v", err)
		}
	}

	for i := 0; i < benchSize; i++ {
		id, err := playerService.CreatePlayer("Zzz Vote Bench " + label + " " + uuid.NewString())
		if err != nil {
			t.Fatalf("failed to create bench player %d: %v", i, err)
		}
		env.bench = append(env.bench, id)
	}

	return env
}

// TestDeleteMatch_Integration_RemovesVotes: MatchVote rows are not cascaded
// by the database (MatchVote declares no association on Match, same as
// MatchRegistration), so deleting a match has to take its votes with it too —
// otherwise they'd be orphan rows nothing can ever read or clean up again.
func TestDeleteMatch_Integration_RemovesVotes(t *testing.T) {
	env := newVoteEnv(t, "DeleteMatch", 2, 0)

	if err := env.votes.Vote(env.matchID, env.roster[0], env.roster[1]); err != nil {
		t.Fatalf("Vote returned error: %v", err)
	}

	if err := env.matches.DeleteMatch(env.matchID, env.groupID); err != nil {
		t.Fatalf("DeleteMatch returned error: %v", err)
	}

	var remaining int64
	if err := env.tx.Model(&models.MatchVote{}).Where("match_id = ?", env.matchID).
		Count(&remaining).Error; err != nil {
		t.Fatalf("failed to count votes: %v", err)
	}
	if remaining != 0 {
		t.Errorf("%d votes survived the match deletion, want 0", remaining)
	}
}

// TestVote_Integration_SelfVoteRejected: a roster player cannot vote for
// themselves, even though they are otherwise a perfectly valid candidate.
func TestVote_Integration_SelfVoteRejected(t *testing.T) {
	env := newVoteEnv(t, "SelfVote", 2, 0)

	if err := env.votes.Vote(env.matchID, env.roster[0], env.roster[0]); !errors.Is(err, services.ErrCannotVoteForSelf) {
		t.Errorf("Vote(self) = %v, want services.ErrCannotVoteForSelf", err)
	}

	summary, err := env.votes.ListVotes(env.matchID, env.roster[0])
	if err != nil {
		t.Fatalf("ListVotes returned error: %v", err)
	}
	if len(summary.Tally) != 0 {
		t.Errorf("tally = %+v after a rejected self-vote, want empty", summary.Tally)
	}
}

// TestVote_Integration_NotOnRosterRejected: a real player who simply never
// played in this match cannot be voted for.
func TestVote_Integration_NotOnRosterRejected(t *testing.T) {
	env := newVoteEnv(t, "NotOnRoster", 1, 1)

	if err := env.votes.Vote(env.matchID, env.roster[0], env.bench[0]); !errors.Is(err, services.ErrVotedForPlayerNotOnRoster) {
		t.Errorf("Vote(bench player) = %v, want services.ErrVotedForPlayerNotOnRoster", err)
	}
}

// TestVote_Integration_UpsertReplacesExistingVote is the core divergence from
// MatchRegistration: casting a second vote is not a conflict, it replaces the
// first one — the tally reflects only the caller's *current* choice.
func TestVote_Integration_UpsertReplacesExistingVote(t *testing.T) {
	env := newVoteEnv(t, "Upsert", 3, 0)
	voter := env.roster[0]

	if err := env.votes.Vote(env.matchID, voter, env.roster[1]); err != nil {
		t.Fatalf("first Vote returned error: %v", err)
	}

	summary, err := env.votes.ListVotes(env.matchID, voter)
	if err != nil {
		t.Fatalf("ListVotes returned error: %v", err)
	}
	if summary.MyVoteFor == nil || *summary.MyVoteFor != env.roster[1] {
		t.Fatalf("MyVoteFor = %v, want %s", summary.MyVoteFor, env.roster[1])
	}
	if len(summary.Tally) != 1 || summary.Tally[0].PlayerID != env.roster[1] || summary.Tally[0].Votes != 1 {
		t.Fatalf("tally after first vote = %+v, want a single vote for roster[1]", summary.Tally)
	}

	// Second vote from the same voter, for a different candidate: the tally
	// must show exactly one vote total, now for the new candidate — not two
	// rows and not a rejection.
	newCandidate := env.roster[2]
	if err := env.votes.Vote(env.matchID, voter, newCandidate); err != nil {
		t.Fatalf("second Vote (changing choice) returned error: %v", err)
	}

	summary, err = env.votes.ListVotes(env.matchID, voter)
	if err != nil {
		t.Fatalf("ListVotes returned error: %v", err)
	}
	if summary.MyVoteFor == nil || *summary.MyVoteFor != newCandidate {
		t.Fatalf("MyVoteFor after changing = %v, want %s", summary.MyVoteFor, newCandidate)
	}
	totalVotes := 0
	for _, c := range summary.Tally {
		totalVotes += c.Votes
	}
	if totalVotes != 1 {
		t.Errorf("total votes across the tally = %d, want 1 (the voter's single, current vote)", totalVotes)
	}
	for _, c := range summary.Tally {
		if c.PlayerID == env.roster[1] {
			t.Errorf("the old candidate (%s) still has a vote after the voter changed their mind: %+v", env.roster[1], summary.Tally)
		}
	}
}

// TestUnvote_Integration_NoOpWhenNotVoted mirrors ReopenRegistrations' own
// no-op-success test: removing a vote that was never cast must not error.
func TestUnvote_Integration_NoOpWhenNotVoted(t *testing.T) {
	env := newVoteEnv(t, "UnvoteNoOp", 1, 0)

	if err := env.votes.Unvote(env.matchID, env.roster[0]); err != nil {
		t.Errorf("Unvote with no existing vote returned error: %v, want nil (no-op success)", err)
	}
}

// TestUnvote_Integration_RemovesTheVote: casting then withdrawing a vote
// leaves the tally exactly as it started.
func TestUnvote_Integration_RemovesTheVote(t *testing.T) {
	env := newVoteEnv(t, "UnvoteRemoves", 2, 0)
	voter, candidatePlayer := env.roster[0], env.roster[1]

	if err := env.votes.Vote(env.matchID, voter, candidatePlayer); err != nil {
		t.Fatalf("Vote returned error: %v", err)
	}
	if err := env.votes.Unvote(env.matchID, voter); err != nil {
		t.Fatalf("Unvote returned error: %v", err)
	}

	summary, err := env.votes.ListVotes(env.matchID, voter)
	if err != nil {
		t.Fatalf("ListVotes returned error: %v", err)
	}
	if summary.MyVoteFor != nil {
		t.Errorf("MyVoteFor = %v after Unvote, want nil", summary.MyVoteFor)
	}
	if len(summary.Tally) != 0 {
		t.Errorf("tally = %+v after the only vote was withdrawn, want empty", summary.Tally)
	}

	// Unvoting again is still a no-op, not an error.
	if err := env.votes.Unvote(env.matchID, voter); err != nil {
		t.Errorf("Unvote twice returned error: %v, want nil", err)
	}
}

// TestListVotes_Integration_TallyOrderedByCountThenName checks the ordering
// contract: most votes first, alphabetical name as the tie-break.
func TestListVotes_Integration_TallyOrderedByCountThenName(t *testing.T) {
	env := newVoteEnv(t, "TallyOrder", 3, 0)
	alice, bob, carol := env.roster[0], env.roster[1], env.roster[2]

	// bob gets 2 votes, alice and carol get 1 each — but distinct voters are
	// needed since a voter can cast only one vote per match. Use bench
	// players purely as voters here (voter eligibility is not this service's
	// concern — see its own doc comment).
	voter1, voter2, voter3 := uuid.New(), uuid.New(), uuid.New()
	// voter ids must be real players only insofar as ListVotes' MyVoteFor
	// lookup does not require it, and Vote never checks the voter's own
	// identity against any table — it only rejects voter == votedFor.
	if err := env.votes.Vote(env.matchID, voter1, bob); err != nil {
		t.Fatalf("Vote 1 returned error: %v", err)
	}
	if err := env.votes.Vote(env.matchID, voter2, bob); err != nil {
		t.Fatalf("Vote 2 returned error: %v", err)
	}
	if err := env.votes.Vote(env.matchID, voter3, alice); err != nil {
		t.Fatalf("Vote 3 returned error: %v", err)
	}
	if err := env.votes.Vote(env.matchID, uuid.New(), carol); err != nil {
		t.Fatalf("Vote 4 returned error: %v", err)
	}

	summary, err := env.votes.ListVotes(env.matchID, uuid.New())
	if err != nil {
		t.Fatalf("ListVotes returned error: %v", err)
	}
	if len(summary.Tally) != 3 {
		t.Fatalf("tally = %+v, want 3 candidates", summary.Tally)
	}
	if summary.Tally[0].PlayerID != bob || summary.Tally[0].Votes != 2 {
		t.Errorf("tally[0] = %+v, want bob with 2 votes", summary.Tally[0])
	}
	// alice and carol are tied at 1 vote; alphabetical name breaks the tie,
	// but since these are randomly-suffixed names, just check both are
	// present with 1 vote each in the remaining slots.
	for _, c := range summary.Tally[1:] {
		if c.Votes != 1 {
			t.Errorf("tally entry %+v has %d votes, want 1", c, c.Votes)
		}
	}
	if summary.MyVoteFor != nil {
		t.Errorf("MyVoteFor = %v for a caller who never voted, want nil", summary.MyVoteFor)
	}
}

// TestVote_Integration_UnknownMatchNotFound mirrors
// TestCloseRegistrations_Integration_WrongGroupNotFound: an id naming no
// match at all is ErrMatchNotFound, not a roster/self-vote refusal.
func TestVote_Integration_UnknownMatchNotFound(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	votes := services.NewMatchVoteService(tx)

	unknownMatch := uuid.New()
	if err := votes.Vote(unknownMatch, uuid.New(), uuid.New()); !errors.Is(err, services.ErrMatchNotFound) {
		t.Errorf("Vote on an unknown match = %v, want services.ErrMatchNotFound", err)
	}
	if _, err := votes.ListVotes(unknownMatch, uuid.New()); !errors.Is(err, services.ErrMatchNotFound) {
		t.Errorf("ListVotes on an unknown match = %v, want services.ErrMatchNotFound", err)
	}
	// Unvote, on the other hand, is a no-op regardless — see its own comment.
	if err := votes.Unvote(unknownMatch, uuid.New()); err != nil {
		t.Errorf("Unvote on an unknown match returned error: %v, want nil (no-op)", err)
	}
}

// TestVote_Integration_WindowClosedForOldUnscheduledMatch checks the 24h
// window end to end for the non-scheduled anchor: a match recorded well over
// a day ago (CreatedAt, backdated directly since CreateMatch always stamps
// "now") refuses a new vote, a change of vote, and a withdrawal alike, but
// still answers ListVotes with whatever tally already existed.
func TestVote_Integration_WindowClosedForOldUnscheduledMatch(t *testing.T) {
	env := newVoteEnv(t, "WindowClosedOld", 2, 0)
	voter, second := env.roster[0], env.roster[1]

	// Cast a vote while the window is still open (CreateMatch just stamped
	// CreatedAt as "now"), then backdate the match by more than 24h.
	otherVoter := env.roster[1]
	if err := env.votes.Vote(env.matchID, otherVoter, env.roster[0]); err != nil {
		t.Fatalf("Vote before backdating returned error: %v", err)
	}
	if err := env.tx.Model(&models.Match{}).Where("id = ?", env.matchID).
		Update("created_at", time.Now().Add(-25*time.Hour)).Error; err != nil {
		t.Fatalf("failed to backdate the match: %v", err)
	}

	if err := env.votes.Vote(env.matchID, voter, second); !errors.Is(err, services.ErrVotingClosed) {
		t.Errorf("Vote after the window closed = %v, want services.ErrVotingClosed", err)
	}
	if err := env.votes.Unvote(env.matchID, otherVoter); !errors.Is(err, services.ErrVotingClosed) {
		t.Errorf("Unvote after the window closed = %v, want services.ErrVotingClosed", err)
	}

	// The existing tally is still readable — ListVotes is never gated on the
	// window, since the result is finalized, not hidden.
	summary, err := env.votes.ListVotes(env.matchID, voter)
	if err != nil {
		t.Fatalf("ListVotes returned error: %v", err)
	}
	if len(summary.Tally) != 1 || summary.Tally[0].PlayerID != env.roster[0] || summary.Tally[0].Votes != 1 {
		t.Errorf("tally = %+v after the window closed, want the pre-existing vote for roster[0] untouched", summary.Tally)
	}
}

// TestVote_Integration_WindowStillOpenJustBeforeDeadline is the mirror check:
// a match one second shy of 24 hours old must still accept a vote.
func TestVote_Integration_WindowStillOpenJustBeforeDeadline(t *testing.T) {
	env := newVoteEnv(t, "WindowStillOpen", 2, 0)

	if err := env.tx.Model(&models.Match{}).Where("id = ?", env.matchID).
		Update("created_at", time.Now().Add(-24*time.Hour+time.Second)).Error; err != nil {
		t.Fatalf("failed to backdate the match: %v", err)
	}

	if err := env.votes.Vote(env.matchID, env.roster[0], env.roster[1]); err != nil {
		t.Errorf("Vote just before the 24h deadline returned error: %v, want nil", err)
	}
}

// TestVote_Integration_ScheduledMatchWindowAnchorsOnKickoff: a scheduled
// match's window is measured from ScheduledAt (kick-off), not from CreatedAt
// — a match created (and therefore logged) well over 24h ago but scheduled to
// kick off just now must still accept a vote.
func TestVote_Integration_ScheduledMatchWindowAnchorsOnKickoff(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	teamService := services.NewTeamService(tx)
	playerService := services.NewPlayerService(tx)
	matchService := services.NewMatchService(tx)
	voteService := services.NewMatchVoteService(tx)

	group, err := groupService.CreateGroup("Zzz Votes ScheduledAnchor", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	teams, err := teamService.GetTeamsByGroupID(group.ID)
	if err != nil {
		t.Fatalf("failed to load group's teams: %v", err)
	}
	black := teams[0]

	kickoff := time.Now().Add(-time.Hour)
	opensAt := kickoff.Add(-48 * time.Hour)
	maxPlayers := 10
	matchID, err := matchService.CreateMatch(services.MatchSpec{
		ScheduledAt:         &kickoff,
		RegistrationOpensAt: &opensAt,
		MaxPlayers:          &maxPlayers,
	}, group.ID)
	if err != nil {
		t.Fatalf("failed to create the scheduled match: %v", err)
	}

	alice, err := playerService.CreatePlayer("Zzz Vote ScheduledAnchor Alice " + uuid.NewString())
	if err != nil {
		t.Fatalf("failed to create player: %v", err)
	}
	bob, err := playerService.CreatePlayer("Zzz Vote ScheduledAnchor Bob " + uuid.NewString())
	if err != nil {
		t.Fatalf("failed to create player: %v", err)
	}
	if err := matchService.UpdateMatch(models.MatchWithDetails{
		ID:    matchID,
		Teams: []models.TeamWithPlayers{{ID: black.ID, Players: []models.PlayerCustom{{ID: alice}, {ID: bob}}}},
	}); err != nil {
		t.Fatalf("failed to compose roster: %v", err)
	}

	// Backdate the row itself (CreateMatch stamps CreatedAt as "now", long
	// before kick-off in this scenario) to prove the window uses ScheduledAt,
	// not CreatedAt, for a scheduled match.
	if err := tx.Model(&models.Match{}).Where("id = ?", matchID).
		Update("created_at", time.Now().Add(-30*24*time.Hour)).Error; err != nil {
		t.Fatalf("failed to backdate the match's CreatedAt: %v", err)
	}

	if err := voteService.Vote(matchID, alice, bob); err != nil {
		t.Errorf("Vote shortly after kick-off returned error: %v, want nil (window anchors on ScheduledAt)", err)
	}
}

// TestTallyVotesForMatches_Integration_GroupsPerMatch is what
// StandingsService.GetMotmStandings relies on: one query covering several
// matches at once, correctly keyed and with a match that has no votes at all
// simply absent. Both matches are built inside the *same* transaction (unlike
// newVoteEnv, which opens its own) precisely so this asserts what the query
// actually returns, rather than two independent transactions each unable to
// see the other's uncommitted data.
func TestTallyVotesForMatches_Integration_GroupsPerMatch(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	teamService := services.NewTeamService(tx)
	playerService := services.NewPlayerService(tx)
	matchService := services.NewMatchService(tx)
	voteService := services.NewMatchVoteService(tx)

	group, err := groupService.CreateGroup("Zzz Votes TallyMatches", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	teams, err := teamService.GetTeamsByGroupID(group.ID)
	if err != nil {
		t.Fatalf("failed to load group's teams: %v", err)
	}
	black := teams[0]

	newMatchWithRoster := func(label string) (uuid.UUID, uuid.UUID) {
		matchID, err := matchService.CreateMatch(services.MatchSpec{Date: models.Date(time.Now())}, group.ID)
		if err != nil {
			t.Fatalf("failed to create match %s: %v", label, err)
		}
		playerID, err := playerService.CreatePlayer("Zzz Vote TallyMatches " + label + " " + uuid.NewString())
		if err != nil {
			t.Fatalf("failed to create player for match %s: %v", label, err)
		}
		if err := matchService.UpdateMatch(models.MatchWithDetails{
			ID:    matchID,
			Teams: []models.TeamWithPlayers{{ID: black.ID, Players: []models.PlayerCustom{{ID: playerID}}}},
		}); err != nil {
			t.Fatalf("failed to compose roster for match %s: %v", label, err)
		}
		return matchID, playerID
	}

	matchA, playerA := newMatchWithRoster("A")
	matchB, _ := newMatchWithRoster("B")

	if err := voteService.Vote(matchA, uuid.New(), playerA); err != nil {
		t.Fatalf("Vote on match A returned error: %v", err)
	}

	byMatch, err := voteService.TallyVotesForMatches([]uuid.UUID{matchA, matchB})
	if err != nil {
		t.Fatalf("TallyVotesForMatches returned error: %v", err)
	}
	if len(byMatch[matchA]) != 1 || byMatch[matchA][0].PlayerID != playerA {
		t.Errorf("byMatch[A] = %+v, want a single vote for playerA", byMatch[matchA])
	}
	if len(byMatch[matchB]) != 0 {
		t.Errorf("byMatch[B] = %+v, want empty: nobody voted in match B", byMatch[matchB])
	}

	// An empty id list must not error and must return an empty map.
	empty, err := voteService.TallyVotesForMatches(nil)
	if err != nil {
		t.Fatalf("TallyVotesForMatches(nil) returned error: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("TallyVotesForMatches(nil) = %+v, want empty map", empty)
	}
}
