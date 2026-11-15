package services_test

import (
	"testing"
	"time"

	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/google/uuid"
)

// countingFixture is a group with two members, alice and bob, ready for a
// match between them.
type countingFixture struct {
	group        uuid.UUID
	black, white uuid.UUID
	alice, bob   uuid.UUID
	matches      *services.MatchService
	votes        *services.MatchVoteService
	standings    *services.StandingsService
}

func newCountingFixture(t *testing.T, name string) (countingFixture, func(matchID uuid.UUID, column string, value interface{})) {
	t.Helper()
	tx := testutil.BeginTx(t, testutil.OpenDB(t))
	membership := services.NewGroupMembershipService(tx)
	group, err := services.NewGroupService(tx).CreateGroup("Zzz Integration Counting "+name, services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	teams, err := services.NewTeamService(tx).GetTeamsByGroupID(group.ID)
	if err != nil || len(teams) != 2 {
		t.Fatalf("failed to load the group's 2 teams: %v (got %d)", err, len(teams))
	}
	players := services.NewPlayerService(tx)
	alice, err := players.CreatePlayer("Zzz Integration Counting Alice " + name)
	if err != nil {
		t.Fatalf("failed to create alice: %v", err)
	}
	bob, err := players.CreatePlayer("Zzz Integration Counting Bob " + name)
	if err != nil {
		t.Fatalf("failed to create bob: %v", err)
	}
	for _, id := range []uuid.UUID{alice, bob} {
		if err := membership.AddPlayerToGroup(group.ID, id); err != nil {
			t.Fatalf("failed to add a member: %v", err)
		}
	}
	set := func(matchID uuid.UUID, column string, value interface{}) {
		t.Helper()
		if err := tx.Model(&models.Match{}).Where("id = ?", matchID).Update(column, value).Error; err != nil {
			t.Fatalf("failed to set %s: %v", column, err)
		}
	}
	return countingFixture{
		group: group.ID, black: teams[0].ID, white: teams[1].ID, alice: alice, bob: bob,
		matches: services.NewMatchService(tx), votes: services.NewMatchVoteService(tx),
		standings: services.NewStandingsService(tx, membership),
	}, set
}

// play records alice 2 – 1 bob on a match.
func (f countingFixture) play(t *testing.T, matchID uuid.UUID) {
	t.Helper()
	if err := f.matches.UpdateMatch(matchID, f.group, []models.TeamWithPlayers{
		{ID: f.black, Players: []models.PlayerCustom{{ID: f.alice, GoalsScored: 2}}},
		{ID: f.white, Players: []models.PlayerCustom{{ID: f.bob, GoalsScored: 1}}},
	}); err != nil {
		t.Fatalf("failed to save the score: %v", err)
	}
}

func (f countingFixture) alicePoints(t *testing.T) (*models.PointsStandingRow, int) {
	t.Helper()
	points, err := f.standings.GetPointsStandings(f.group, "")
	if err != nil {
		t.Fatalf("GetPointsStandings: %v", err)
	}
	profile, err := f.standings.GetPlayerProfile(f.alice, "")
	if err != nil {
		t.Fatalf("GetPlayerProfile: %v", err)
	}
	return pointsRowByID(points, f.alice), profile.Overall.Played
}

func (f countingFixture) bobAwards(t *testing.T) (int, int) {
	t.Helper()
	rows, err := f.standings.GetMotmStandings(f.group, "")
	if err != nil {
		t.Fatalf("GetMotmStandings: %v", err)
	}
	awards := 0
	for _, row := range rows {
		if row.PlayerID == f.bob {
			awards = row.Awards
		}
	}
	profile, err := f.standings.GetPlayerProfile(f.bob, "")
	if err != nil {
		t.Fatalf("GetPlayerProfile: %v", err)
	}
	return awards, profile.Overall.MotmAwards
}

func (f countingFixture) eloRated(t *testing.T) bool {
	t.Helper()
	rows, err := f.standings.GetEloStandings(f.group)
	if err != nil {
		t.Fatalf("GetEloStandings: %v", err)
	}
	for _, row := range rows {
		if row.PlayerID == f.alice && row.GamesPlayed > 0 {
			return true
		}
	}
	return false
}

// A scheduled match counts in the points, the profile and the Elo the
// moment its score is saved after kick-off, on its own day.
func TestStandings_Integration_ScheduledMatchCountsOnceTheScoreIsSaved(t *testing.T) {
	f, _ := newCountingFixture(t, "Scheduled")
	kickoff := time.Now().Add(-2 * time.Hour)
	opensAt := kickoff.Add(-48 * time.Hour)
	maxPlayers := 10
	matchID, err := f.matches.CreateMatch(services.MatchSpec{ScheduledAt: &kickoff, RegistrationOpensAt: &opensAt, MaxPlayers: &maxPlayers}, f.group)
	if err != nil {
		t.Fatalf("failed to create the scheduled match: %v", err)
	}
	if row, played := f.alicePoints(t); row != nil || played != 0 || f.eloRated(t) {
		t.Fatalf("counted before any score: row %+v, profile played %d", row, played)
	}
	f.play(t, matchID)
	if row, played := f.alicePoints(t); row == nil || row.Points != 3 || played != 1 {
		t.Errorf("once the score is saved: row %+v, profile played %d; want 3 points, 1 played", row, played)
	}
	if !f.eloRated(t) {
		t.Errorf("Elo doesn't count the match once its score is saved")
	}
}

// A match created without a schedule never gets a ResultRecordedAt: it
// keeps the midnight-after rule.
func TestStandings_Integration_UnscheduledMatchCountsFromTheDayAfter(t *testing.T) {
	f, set := newCountingFixture(t, "Unscheduled")
	matchID, err := f.matches.CreateMatch(services.MatchSpec{Date: models.Date(time.Now())}, f.group)
	if err != nil {
		t.Fatalf("failed to create the match: %v", err)
	}
	f.play(t, matchID)
	if row, played := f.alicePoints(t); row != nil || played != 0 || f.eloRated(t) {
		t.Fatalf("counted on its own day: row %+v, profile played %d", row, played)
	}
	set(matchID, "date", time.Now().AddDate(0, 0, -1))
	if row, played := f.alicePoints(t); row == nil || row.Points != 3 || played != 1 || !f.eloRated(t) {
		t.Errorf("the day after: row %+v, profile played %d; want 3 points, 1 played, rated", row, played)
	}
}

// The MOTM award counts only once the vote is closed, in the ranking and
// in the profile.
func TestStandings_Integration_MotmCountsOnceTheVoteIsClosed(t *testing.T) {
	f, set := newCountingFixture(t, "Motm")
	matchID, err := f.matches.CreateMatch(services.MatchSpec{Date: models.Date(time.Now().AddDate(0, 0, -1))}, f.group)
	if err != nil {
		t.Fatalf("failed to create the match: %v", err)
	}
	f.play(t, matchID)
	if err := f.votes.Vote(matchID, f.alice, f.bob); err != nil {
		t.Fatalf("failed to vote: %v", err)
	}
	if ranking, profile := f.bobAwards(t); ranking != 0 || profile != 0 {
		t.Errorf("vote still open: ranking %d, profile %d awards; want none", ranking, profile)
	}
	if row, _ := f.alicePoints(t); row == nil {
		t.Errorf("the points should already count the day after an unscheduled match")
	}
	set(matchID, "date", time.Now().AddDate(0, 0, -2))
	if ranking, profile := f.bobAwards(t); ranking != 1 || profile != 1 {
		t.Errorf("vote closed: ranking %d, profile %d awards; want 1", ranking, profile)
	}
}

// A scheduled match whose score was never recorded wins no MOTM award, even
// after its vote has closed.
func TestStandings_Integration_MotmNeedsARecordedResult(t *testing.T) {
	f, set := newCountingFixture(t, "MotmUnrecorded")
	kickoff := time.Now().Add(-2 * time.Hour)
	opensAt := kickoff.Add(-48 * time.Hour)
	maxPlayers := 10
	matchID, err := f.matches.CreateMatch(services.MatchSpec{ScheduledAt: &kickoff, RegistrationOpensAt: &opensAt, MaxPlayers: &maxPlayers}, f.group)
	if err != nil {
		t.Fatalf("failed to create the scheduled match: %v", err)
	}
	f.play(t, matchID)
	if err := f.votes.Vote(matchID, f.alice, f.bob); err != nil {
		t.Fatalf("failed to vote: %v", err)
	}
	set(matchID, "result_recorded_at", nil)
	set(matchID, "date", time.Now().AddDate(0, 0, -3))
	if ranking, profile := f.bobAwards(t); ranking != 0 || profile != 0 {
		t.Errorf("no recorded result: ranking %d, profile %d awards; want none", ranking, profile)
	}
	set(matchID, "result_recorded_at", kickoff.Add(time.Hour))
	if ranking, profile := f.bobAwards(t); ranking != 1 || profile != 1 {
		t.Errorf("recorded result, vote closed: ranking %d, profile %d awards; want 1", ranking, profile)
	}
}
