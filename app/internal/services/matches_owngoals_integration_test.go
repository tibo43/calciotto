package services_test

import (
	"testing"
	"time"

	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"
)

// TestMatchOwnGoals_Integration pins the whole own-goal path end to end:
// UpdateMatch persists MatchPlayer.OwnGoals, and both read paths
// (GetMatchDetailsByID and GetMatchesDetails) credit it to the OTHER team's
// Score while keeping the scoring player's own GoalsScored untouched — see
// CLAUDE.md's "Data model" section for why Team.Score is derived, never
// stored, and matches.go's own-goal crediting logic this pins.
func TestMatchOwnGoals_Integration(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	teamService := services.NewTeamService(tx)
	playerService := services.NewPlayerService(tx)
	matchService := services.NewMatchService(tx)

	group, err := groupService.CreateGroup("Zzz OwnGoals Group", services.DefaultTeamSpecs)
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
	black, white := &teams[0], &teams[1]

	aliceID, err := playerService.CreatePlayer("Zzz OwnGoals Alice")
	if err != nil {
		t.Fatalf("failed to create player alice: %v", err)
	}
	bobID, err := playerService.CreatePlayer("Zzz OwnGoals Bob")
	if err != nil {
		t.Fatalf("failed to create player bob: %v", err)
	}

	matchID, err := matchService.CreateMatch(services.MatchSpec{Date: models.Date(time.Now().AddDate(0, 0, -2))}, group.ID)
	if err != nil {
		t.Fatalf("failed to create match: %v", err)
	}

	// Alice (black) scores 2 real goals and concedes 1 own goal: the own
	// goal must land on white's Score, not black's, and must never inflate
	// Alice's own GoalsScored.
	teamsPayload := []models.TeamWithPlayers{
		{ID: black.ID, Players: []models.PlayerCustom{{ID: aliceID, GoalsScored: 2, OwnGoals: 1}}},
		{ID: white.ID, Players: []models.PlayerCustom{{ID: bobID, GoalsScored: 0}}},
	}
	if err := matchService.UpdateMatch(matchID, group.ID, teamsPayload); err != nil {
		t.Fatalf("UpdateMatch returned error: %v", err)
	}

	details, err := matchService.GetMatchDetailsByID(matchID, group.ID)
	if err != nil {
		t.Fatalf("GetMatchDetailsByID returned error: %v", err)
	}
	blackTeam, whiteTeam := teamByID(t, details.Teams, black.ID), teamByID(t, details.Teams, white.ID)

	if blackTeam.Score != 2 {
		t.Errorf("black team score = %d, want 2 (the own goal must not count toward the scoring player's own team)", blackTeam.Score)
	}
	if whiteTeam.Score != 1 {
		t.Errorf("white team score = %d, want 1 (credited from black's own goal)", whiteTeam.Score)
	}
	if blackTeam.Players[0].GoalsScored != 2 {
		t.Errorf("alice's GoalsScored = %d, want 2 (own goals must not be folded into it)", blackTeam.Players[0].GoalsScored)
	}
	if blackTeam.Players[0].OwnGoals != 1 {
		t.Errorf("alice's OwnGoals = %d, want 1", blackTeam.Players[0].OwnGoals)
	}

	// The list endpoint must derive the same scores, not just the by-ID one.
	all, err := matchService.GetMatchesDetails(group.ID, "")
	if err != nil {
		t.Fatalf("GetMatchesDetails returned error: %v", err)
	}
	found := matchByID(all, matchID)
	if found == nil {
		t.Fatal("match not found in GetMatchesDetails results")
	}
	blackTeam, whiteTeam = teamByID(t, found.Teams, black.ID), teamByID(t, found.Teams, white.ID)
	if blackTeam.Score != 2 || whiteTeam.Score != 1 {
		t.Errorf("GetMatchesDetails scores = black %d, white %d, want black 2, white 1", blackTeam.Score, whiteTeam.Score)
	}

	// Clearing the own goal on a later save must roll white's credited score
	// back down — Score is derived fresh on every read, never stored.
	teamsPayload[0].Players[0].OwnGoals = 0
	if err := matchService.UpdateMatch(matchID, group.ID, teamsPayload); err != nil {
		t.Fatalf("UpdateMatch (clearing own goal) returned error: %v", err)
	}
	details, err = matchService.GetMatchDetailsByID(matchID, group.ID)
	if err != nil {
		t.Fatalf("GetMatchDetailsByID (after clearing own goal) returned error: %v", err)
	}
	whiteTeam = teamByID(t, details.Teams, white.ID)
	if whiteTeam.Score != 0 {
		t.Errorf("white team score after clearing own goal = %d, want 0", whiteTeam.Score)
	}
}
