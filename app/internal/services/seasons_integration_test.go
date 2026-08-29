package services_test

import (
	"testing"
	"time"

	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"
)

// TestStandings_Integration_ScopedPerSeason is the season-axis counterpart of
// TestMatchesAndStandings_Integration_ScopedPerGroup: one group, two matches
// on either side of a September 1st boundary, and standings that must only
// ever count the season asked for.
func TestStandings_Integration_ScopedPerSeason(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	teamService := services.NewTeamService(tx)
	playerService := services.NewPlayerService(tx)
	matchService := services.NewMatchService(tx)
	standingsService := services.NewStandingsService(tx, services.NewGroupMembershipService(tx))

	group, err := groupService.CreateGroup("Zzz Integration Season Group", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	teams, err := teamService.GetTeamsByGroupID(group.ID)
	if err != nil || len(teams) != 2 {
		t.Fatalf("failed to load group teams: err=%v teams=%+v", err, teams)
	}

	aliceID, err := playerService.CreatePlayer("Zzz Integration Season Alice")
	if err != nil {
		t.Fatalf("failed to create player alice: %v", err)
	}
	bobID, err := playerService.CreatePlayer("Zzz Integration Season Bob")
	if err != nil {
		t.Fatalf("failed to create player bob: %v", err)
	}

	const (
		seasonOld = "2023-2024"
		seasonNew = "2024-2025"
	)
	// Two matches in the same group, in two different seasons: alice wins 3-0
	// in the old season, bob wins 1-0 in the new one.
	oldDate := models.Date(time.Date(2024, time.August, 25, 0, 0, 0, 0, time.UTC))
	newDate := models.Date(time.Date(2024, time.September, 8, 0, 0, 0, 0, time.UTC))

	oldMatchID, err := matchService.CreateMatch(oldDate, group.ID)
	if err != nil {
		t.Fatalf("failed to create the old-season match: %v", err)
	}
	if err := matchService.UpdateMatch(models.MatchWithDetails{
		ID: oldMatchID,
		Teams: []models.TeamWithPlayers{
			{ID: teams[0].ID, Players: []models.PlayerCustom{{ID: aliceID, GoalsScored: 3}}},
			{ID: teams[1].ID, Players: []models.PlayerCustom{{ID: bobID, GoalsScored: 0}}},
		},
	}); err != nil {
		t.Fatalf("UpdateMatch (old season) returned error: %v", err)
	}

	newMatchID, err := matchService.CreateMatch(newDate, group.ID)
	if err != nil {
		t.Fatalf("failed to create the new-season match: %v", err)
	}
	if err := matchService.UpdateMatch(models.MatchWithDetails{
		ID: newMatchID,
		Teams: []models.TeamWithPlayers{
			{ID: teams[0].ID, Players: []models.PlayerCustom{{ID: aliceID, GoalsScored: 0}}},
			{ID: teams[1].ID, Players: []models.PlayerCustom{{ID: bobID, GoalsScored: 1}}},
		},
	}); err != nil {
		t.Fatalf("UpdateMatch (new season) returned error: %v", err)
	}

	// Old season only: alice won 3-0, bob lost.
	pointsOld, err := standingsService.GetPointsStandings(group.ID, seasonOld)
	if err != nil {
		t.Fatalf("GetPointsStandings(%s) returned error: %v", seasonOld, err)
	}
	aliceOld := pointsRowByID(pointsOld, aliceID)
	if aliceOld == nil || aliceOld.Played != 1 || aliceOld.Points != 3 || aliceOld.GoalsFor != 3 {
		t.Errorf("alice's %s standing = %+v, want 1 played / 3 points / 3 goals", seasonOld, aliceOld)
	}
	bobOld := pointsRowByID(pointsOld, bobID)
	if bobOld == nil || bobOld.Played != 1 || bobOld.Points != 0 {
		t.Errorf("bob's %s standing = %+v, want 1 played / 0 points", seasonOld, bobOld)
	}

	// New season only: the result is the other way round.
	pointsNew, err := standingsService.GetPointsStandings(group.ID, seasonNew)
	if err != nil {
		t.Fatalf("GetPointsStandings(%s) returned error: %v", seasonNew, err)
	}
	aliceNew := pointsRowByID(pointsNew, aliceID)
	if aliceNew == nil || aliceNew.Played != 1 || aliceNew.Points != 0 || aliceNew.GoalsFor != 0 {
		t.Errorf("alice's %s standing = %+v, want 1 played / 0 points / 0 goals", seasonNew, aliceNew)
	}
	bobNew := pointsRowByID(pointsNew, bobID)
	if bobNew == nil || bobNew.Played != 1 || bobNew.Points != 3 {
		t.Errorf("bob's %s standing = %+v, want 1 played / 3 points", seasonNew, bobNew)
	}

	// No season filter still aggregates both, as before this feature existed.
	pointsAll, err := standingsService.GetPointsStandings(group.ID, "")
	if err != nil {
		t.Fatalf("GetPointsStandings(no season) returned error: %v", err)
	}
	aliceAll := pointsRowByID(pointsAll, aliceID)
	if aliceAll == nil || aliceAll.Played != 2 || aliceAll.Points != 3 {
		t.Errorf("alice's unfiltered standing = %+v, want 2 played / 3 points", aliceAll)
	}

	// An unknown season yields no rows rather than everything.
	pointsUnknown, err := standingsService.GetPointsStandings(group.ID, "1999-2000")
	if err != nil {
		t.Fatalf("GetPointsStandings(unknown season) returned error: %v", err)
	}
	if len(pointsUnknown) != 0 {
		t.Errorf("standings for a season with no matches = %+v, want none", pointsUnknown)
	}

	// Scorers are filtered on the same axis.
	scorersOld, err := standingsService.GetScorers(group.ID, seasonOld)
	if err != nil {
		t.Fatalf("GetScorers(%s) returned error: %v", seasonOld, err)
	}
	aliceScorerOld := scorerRowByID(scorersOld, aliceID)
	if aliceScorerOld == nil || aliceScorerOld.Goals != 3 {
		t.Errorf("alice's %s scorer row = %+v, want 3 goals", seasonOld, aliceScorerOld)
	}
	scorersNew, err := standingsService.GetScorers(group.ID, seasonNew)
	if err != nil {
		t.Fatalf("GetScorers(%s) returned error: %v", seasonNew, err)
	}
	aliceScorerNew := scorerRowByID(scorersNew, aliceID)
	if aliceScorerNew == nil || aliceScorerNew.Goals != 0 {
		t.Errorf("alice's %s scorer row = %+v, want 0 goals", seasonNew, aliceScorerNew)
	}

	// The group's available seasons are derived from those two match dates.
	seasons, err := standingsService.GetSeasons(group.ID)
	if err != nil {
		t.Fatalf("GetSeasons returned error: %v", err)
	}
	if len(seasons) != 2 || seasons[0] != seasonOld || seasons[1] != seasonNew {
		t.Errorf("GetSeasons = %v, want [%s %s]", seasons, seasonOld, seasonNew)
	}
}
