package services_test

import (
	"errors"
	"testing"
	"time"

	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/google/uuid"
)

func TestMatchLifecycle_Integration(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	teamService := services.NewTeamService(tx)
	playerService := services.NewPlayerService(tx)
	matchService := services.NewMatchService(tx)
	standingsService := services.NewStandingsService(tx, services.NewGroupMembershipService(tx))

	group, err := groupService.CreateGroup("Zzz Integration Group", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	// GetMatchesDetails/GetMatchDetailsByID backfill every match with every
	// team *in that match's group* (see matches.go), and CreateGroup always
	// creates exactly the 2 default teams (black/white) for a fresh group.
	teams, err := teamService.GetTeamsByGroupID(group.ID)
	if err != nil {
		t.Fatalf("failed to load group's teams: %v", err)
	}
	if len(teams) != 2 {
		t.Fatalf("expected CreateGroup to create exactly 2 teams, got %d", len(teams))
	}
	black, white := &teams[0], &teams[1]

	aliceID, err := playerService.CreatePlayer("Zzz Integration Alice")
	if err != nil {
		t.Fatalf("failed to create player alice: %v", err)
	}
	bobID, err := playerService.CreatePlayer("Zzz Integration Bob")
	if err != nil {
		t.Fatalf("failed to create player bob: %v", err)
	}

	// Two days ago, not today: IsMatchCompleted (standings.go) only counts
	// a match starting the day after it was played, and "exactly one day
	// ago" is too close to that boundary to be reliable across timezones
	// (a test run shortly after local midnight can still be "today" in
	// UTC, where the match's Date actually gets stored).
	matchID, err := matchService.CreateMatch(services.MatchSpec{Date: models.Date(time.Now().AddDate(0, 0, -2))}, group.ID)
	if err != nil {
		t.Fatalf("failed to create match: %v", err)
	}

	// Step 1: only black has a roster so far — white should still show up, empty.
	step1 := []models.TeamWithPlayers{
		{ID: black.ID, Players: []models.PlayerCustom{{ID: aliceID, GoalsScored: 2}}},
		{ID: white.ID, Players: []models.PlayerCustom{}},
	}
	if err := matchService.UpdateMatch(matchID, group.ID, step1); err != nil {
		t.Fatalf("UpdateMatch (step 1) returned error: %v", err)
	}

	details, err := matchService.GetMatchDetailsByID(matchID, group.ID)
	if err != nil {
		t.Fatalf("GetMatchDetailsByID (step 1) returned error: %v", err)
	}
	if len(details.Teams) != 2 {
		t.Fatalf("expected both teams present even though white has no roster yet, got %d teams: %+v", len(details.Teams), details.Teams)
	}
	blackTeam, whiteTeam := teamByID(t, details.Teams, black.ID), teamByID(t, details.Teams, white.ID)
	if len(blackTeam.Players) != 1 || blackTeam.Score != 2 {
		t.Errorf("black team = %+v, want 1 player and score 2", blackTeam)
	}
	if len(whiteTeam.Players) != 0 || whiteTeam.Score != 0 {
		t.Errorf("white team = %+v, want 0 players and score 0", whiteTeam)
	}

	// The list endpoint must show the same shape, not just the by-ID one.
	all, err := matchService.GetMatchesDetails(group.ID, "")
	if err != nil {
		t.Fatalf("GetMatchesDetails returned error: %v", err)
	}
	found := matchByID(all, matchID)
	if found == nil {
		t.Fatal("created match not found in GetMatchesDetails results")
	}
	if len(found.Teams) != 2 {
		t.Errorf("GetMatchesDetails: expected both teams present, got %d: %+v", len(found.Teams), found.Teams)
	}

	// Step 2: white gets a player, black's player's goal tally is updated.
	step2 := []models.TeamWithPlayers{
		{ID: black.ID, Players: []models.PlayerCustom{{ID: aliceID, GoalsScored: 3}}},
		{ID: white.ID, Players: []models.PlayerCustom{{ID: bobID, GoalsScored: 1}}},
	}
	if err := matchService.UpdateMatch(matchID, group.ID, step2); err != nil {
		t.Fatalf("UpdateMatch (step 2) returned error: %v", err)
	}

	details, err = matchService.GetMatchDetailsByID(matchID, group.ID)
	if err != nil {
		t.Fatalf("GetMatchDetailsByID (step 2) returned error: %v", err)
	}
	blackTeam, whiteTeam = teamByID(t, details.Teams, black.ID), teamByID(t, details.Teams, white.ID)
	if blackTeam.Score != 3 || blackTeam.Players[0].GoalsScored != 3 {
		t.Errorf("black team after update = %+v, want score 3", blackTeam)
	}
	if len(whiteTeam.Players) != 1 || whiteTeam.Score != 1 {
		t.Errorf("white team after update = %+v, want 1 player and score 1", whiteTeam)
	}

	// The points standings and scorers should now reflect this match: black
	// (3-1) won, so alice gets 3 points, bob gets 0; both have their goals.
	points, err := standingsService.GetPointsStandings(group.ID, "")
	if err != nil {
		t.Fatalf("GetPointsStandings returned error: %v", err)
	}
	alicePoints, bobPoints := pointsRowByID(points, aliceID), pointsRowByID(points, bobID)
	if alicePoints == nil || alicePoints.Points != 3 || alicePoints.GoalsFor != 3 {
		t.Errorf("alice points row = %+v, want 3 points / 3 goals", alicePoints)
	}
	if bobPoints == nil || bobPoints.Points != 0 || bobPoints.GoalsFor != 1 {
		t.Errorf("bob points row = %+v, want 0 points / 1 goal", bobPoints)
	}

	scorers, err := standingsService.GetScorers(group.ID, "")
	if err != nil {
		t.Fatalf("GetScorers returned error: %v", err)
	}
	if row := scorerRowByID(scorers, aliceID); row == nil || row.Goals != 3 {
		t.Errorf("alice scorer row = %+v, want 3 goals", row)
	}

	// Step 3: bob is removed from white — the match_player row must actually be deleted.
	step3 := []models.TeamWithPlayers{
		{ID: black.ID, Players: []models.PlayerCustom{{ID: aliceID, GoalsScored: 3}}},
		{ID: white.ID, Players: []models.PlayerCustom{}},
	}
	if err := matchService.UpdateMatch(matchID, group.ID, step3); err != nil {
		t.Fatalf("UpdateMatch (step 3) returned error: %v", err)
	}

	var remaining int64
	if err := tx.Model(&models.MatchPlayer{}).
		Where("match_id = ? AND player_id = ?", matchID, bobID).
		Count(&remaining).Error; err != nil {
		t.Fatalf("failed to count match_players: %v", err)
	}
	if remaining != 0 {
		t.Errorf("expected bob's match_player row to be deleted, found %d remaining", remaining)
	}
}

func TestGetMatchDetailsByID_Integration_NotFound(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	matchService := services.NewMatchService(tx)

	_, err := matchService.GetMatchDetailsByID(uuid.New(), uuid.New())
	if !errors.Is(err, services.ErrMatchNotFound) {
		t.Errorf("GetMatchDetailsByID(random id) error = %v, want services.ErrMatchNotFound", err)
	}
}

// A match ID that exists, but in a different group than the one requested,
// must be treated as not found — it must not leak across group boundaries.
func TestGetMatchDetailsByID_Integration_WrongGroupNotFound(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	matchService := services.NewMatchService(tx)

	groupA, err := groupService.CreateGroup("Zzz Integration Group A", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group A: %v", err)
	}
	groupB, err := groupService.CreateGroup("Zzz Integration Group B", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group B: %v", err)
	}

	matchID, err := matchService.CreateMatch(services.MatchSpec{Date: models.Date(time.Now())}, groupA.ID)
	if err != nil {
		t.Fatalf("failed to create match: %v", err)
	}

	_, err = matchService.GetMatchDetailsByID(matchID, groupB.ID)
	if !errors.Is(err, services.ErrMatchNotFound) {
		t.Errorf("GetMatchDetailsByID(match from group A, group B) error = %v, want services.ErrMatchNotFound", err)
	}
}

// TestGetPointsStandings_Integration_ExcludesUpcomingComposedMatch is the
// real product scenario real feedback flagged: an admin can compose a
// scheduled match's roster ahead of its own kick-off (via "Fill teams from
// sign-ups"), which already satisfies ComputePointsStandings'/
// ComputeScorers' own "both teams have players" check — without
// FilterCompletedMatches (standings.go), that match's goals would count
// immediately, before it has actually been played.
func TestGetPointsStandings_Integration_ExcludesUpcomingComposedMatch(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	teamService := services.NewTeamService(tx)
	playerService := services.NewPlayerService(tx)
	matchService := services.NewMatchService(tx)
	standingsService := services.NewStandingsService(tx, services.NewGroupMembershipService(tx))

	group, err := groupService.CreateGroup("Zzz Integration Upcoming Composed", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	teams, err := teamService.GetTeamsByGroupID(group.ID)
	if err != nil || len(teams) != 2 {
		t.Fatalf("failed to load group's 2 teams: %v (got %d)", err, len(teams))
	}
	black, white := teams[0], teams[1]

	aliceID, err := playerService.CreatePlayer("Zzz Integration Upcoming Alice")
	if err != nil {
		t.Fatalf("failed to create player alice: %v", err)
	}
	bobID, err := playerService.CreatePlayer("Zzz Integration Upcoming Bob")
	if err != nil {
		t.Fatalf("failed to create player bob: %v", err)
	}

	// A scheduled match kicking off tomorrow, with sign-ups already closed —
	// the state "Fill teams from sign-ups" is offered in.
	kickoff := time.Now().Add(24 * time.Hour)
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

	// Composed ahead of kick-off, same as "Fill teams from sign-ups" would do.
	if err := matchService.UpdateMatch(matchID, group.ID, []models.TeamWithPlayers{
		{ID: black.ID, Players: []models.PlayerCustom{{ID: aliceID, GoalsScored: 2}}},
		{ID: white.ID, Players: []models.PlayerCustom{{ID: bobID, GoalsScored: 1}}},
	}); err != nil {
		t.Fatalf("failed to compose the roster: %v", err)
	}

	points, err := standingsService.GetPointsStandings(group.ID, "")
	if err != nil {
		t.Fatalf("GetPointsStandings returned error: %v", err)
	}
	if row := pointsRowByID(points, aliceID); row != nil {
		t.Errorf("alice already appears in points standings for a match that hasn't been played yet: %+v", row)
	}
	scorers, err := standingsService.GetScorers(group.ID, "")
	if err != nil {
		t.Fatalf("GetScorers returned error: %v", err)
	}
	if row := scorerRowByID(scorers, aliceID); row != nil {
		t.Errorf("alice already appears in scorers for a match that hasn't been played yet: %+v", row)
	}

	// Once the match's own Date is safely in the past, the exact same roster
	// must count — nothing here changes except how much time has passed.
	if err := tx.Model(&models.Match{}).Where("id = ?", matchID).
		Update("date", time.Now().AddDate(0, 0, -2)).Error; err != nil {
		t.Fatalf("failed to backdate the match: %v", err)
	}

	points, err = standingsService.GetPointsStandings(group.ID, "")
	if err != nil {
		t.Fatalf("GetPointsStandings returned error: %v", err)
	}
	if row := pointsRowByID(points, aliceID); row == nil || row.Points != 3 || row.GoalsFor != 2 {
		t.Errorf("alice points row after completion = %+v, want 3 points / 2 goals", row)
	}
}

func teamByID(t *testing.T, teams []models.TeamWithPlayers, id uuid.UUID) models.TeamWithPlayers {
	t.Helper()
	for _, team := range teams {
		if team.ID == id {
			return team
		}
	}
	t.Fatalf("team %s not found in %+v", id, teams)
	return models.TeamWithPlayers{}
}

func matchByID(matches []models.MatchWithDetails, id uuid.UUID) *models.MatchWithDetails {
	for i := range matches {
		if matches[i].ID == id {
			return &matches[i]
		}
	}
	return nil
}

func pointsRowByID(rows []models.PointsStandingRow, id uuid.UUID) *models.PointsStandingRow {
	for i := range rows {
		if rows[i].PlayerID == id {
			return &rows[i]
		}
	}
	return nil
}

func scorerRowByID(rows []models.ScorerRow, id uuid.UUID) *models.ScorerRow {
	for i := range rows {
		if rows[i].PlayerID == id {
			return &rows[i]
		}
	}
	return nil
}

// TestUpdateMatch_Integration_ScopedToGroup pins the two checks UpdateMatch
// gained alongside its explicit (matchID, groupID) parameters — the service
// half of the cross-tenant hole PUT /matches/:id used to have, where both ids
// came from the request body:
//
//   - a match that exists but belongs to another group is ErrMatchNotFound,
//     the same "absent, not forbidden" answer GetMatchDetailsByID and
//     DeleteMatch already give;
//   - a team from another group is ErrTeamNotInGroup, even when the match
//     itself is the caller's own — otherwise an admin could still attach
//     their own team to someone else's match, which is the same corruption
//     by a different route.
//
// Both must also leave match_players untouched: rejecting the call after
// having written half the diff would be no better than accepting it.
func TestUpdateMatch_Integration_ScopedToGroup(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	teamService := services.NewTeamService(tx)
	playerService := services.NewPlayerService(tx)
	matchService := services.NewMatchService(tx)

	groupA, err := groupService.CreateGroup("Zzz Update Scope A", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group A: %v", err)
	}
	groupB, err := groupService.CreateGroup("Zzz Update Scope B", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group B: %v", err)
	}
	teamsA, err := teamService.GetTeamsByGroupID(groupA.ID)
	if err != nil {
		t.Fatalf("failed to load group A's teams: %v", err)
	}
	teamsB, err := teamService.GetTeamsByGroupID(groupB.ID)
	if err != nil {
		t.Fatalf("failed to load group B's teams: %v", err)
	}

	playerID, err := playerService.CreatePlayer("Zzz Update Scope Player")
	if err != nil {
		t.Fatalf("failed to create player: %v", err)
	}
	matchAID, err := matchService.CreateMatch(services.MatchSpec{Date: models.Date(time.Now())}, groupA.ID)
	if err != nil {
		t.Fatalf("failed to create group A's match: %v", err)
	}

	rosterA := []models.TeamWithPlayers{{
		ID:      teamsA[0].ID,
		Players: []models.PlayerCustom{{ID: playerID, GoalsScored: 2}},
	}}
	rosterB := []models.TeamWithPlayers{{
		ID:      teamsB[0].ID,
		Players: []models.PlayerCustom{{ID: playerID, GoalsScored: 2}},
	}}

	countRows := func() int64 {
		t.Helper()
		var rows int64
		if err := tx.Model(&models.MatchPlayer{}).Where("match_id = ?", matchAID).Count(&rows).Error; err != nil {
			t.Fatalf("counting match_players returned error: %v", err)
		}
		return rows
	}

	if err := matchService.UpdateMatch(matchAID, groupB.ID, rosterB); !errors.Is(err, services.ErrMatchNotFound) {
		t.Errorf("UpdateMatch(group A's match, under group B) error = %v, want services.ErrMatchNotFound", err)
	}
	if rows := countRows(); rows != 0 {
		t.Errorf("a rejected cross-group UpdateMatch wrote %d match_player row(s)", rows)
	}

	if err := matchService.UpdateMatch(matchAID, groupA.ID, rosterB); !errors.Is(err, services.ErrTeamNotInGroup) {
		t.Errorf("UpdateMatch(group A's match, with group B's team) error = %v, want services.ErrTeamNotInGroup", err)
	}
	if rows := countRows(); rows != 0 {
		t.Errorf("a rejected foreign-team UpdateMatch wrote %d match_player row(s)", rows)
	}

	// The legitimate call still works, so the two checks above reject the
	// forged shapes rather than the operation itself.
	if err := matchService.UpdateMatch(matchAID, groupA.ID, rosterA); err != nil {
		t.Fatalf("UpdateMatch(group A's own match and team) returned error: %v", err)
	}
	if rows := countRows(); rows != 1 {
		t.Errorf("after a legitimate UpdateMatch the match has %d match_player row(s), want 1", rows)
	}
}
