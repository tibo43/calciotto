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

// TestDeleteMatch_Integration_Success covers the happy path: deleting a match
// that has match_players rows must remove both the match itself and every
// match_players row for it (MatchPlayer.MatchID has no ON DELETE CASCADE, so
// skipping that cleanup would leave orphaned rows or fail outright).
func TestDeleteMatch_Integration_Success(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	teamService := services.NewTeamService(tx)
	playerService := services.NewPlayerService(tx)
	matchService := services.NewMatchService(tx)

	group, err := groupService.CreateGroup("Zzz Delete Match Group", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	teams, err := teamService.GetTeamsByGroupID(group.ID)
	if err != nil {
		t.Fatalf("failed to load group's teams: %v", err)
	}
	if len(teams) != 2 {
		t.Fatalf("expected 2 teams, got %d", len(teams))
	}

	aliceID, err := playerService.CreatePlayer("Zzz Delete Match Alice")
	if err != nil {
		t.Fatalf("failed to create player: %v", err)
	}

	matchID, err := matchService.CreateMatch(services.MatchSpec{Date: models.Date(time.Now())}, group.ID)
	if err != nil {
		t.Fatalf("failed to create match: %v", err)
	}

	if err := matchService.UpdateMatch(models.MatchWithDetails{
		ID: matchID,
		Teams: []models.TeamWithPlayers{
			{ID: teams[0].ID, Players: []models.PlayerCustom{{ID: aliceID, GoalsScored: 2}}},
			{ID: teams[1].ID, Players: []models.PlayerCustom{}},
		},
	}); err != nil {
		t.Fatalf("failed to populate match roster: %v", err)
	}

	var rosterCount int64
	if err := tx.Model(&models.MatchPlayer{}).Where("match_id = ?", matchID).Count(&rosterCount).Error; err != nil {
		t.Fatalf("failed to count match_players before delete: %v", err)
	}
	if rosterCount == 0 {
		t.Fatal("expected at least one match_player row before delete")
	}

	if err := matchService.DeleteMatch(matchID, group.ID); err != nil {
		t.Fatalf("DeleteMatch returned error: %v", err)
	}

	var matchCount int64
	if err := tx.Model(&models.Match{}).Where("id = ?", matchID).Count(&matchCount).Error; err != nil {
		t.Fatalf("failed to count matches after delete: %v", err)
	}
	if matchCount != 0 {
		t.Errorf("expected the match row to be gone after delete, found %d", matchCount)
	}

	var remainingRoster int64
	if err := tx.Model(&models.MatchPlayer{}).Where("match_id = ?", matchID).Count(&remainingRoster).Error; err != nil {
		t.Fatalf("failed to count match_players after delete: %v", err)
	}
	if remainingRoster != 0 {
		t.Errorf("expected all match_player rows to be gone after delete, found %d", remainingRoster)
	}

	if _, err := matchService.GetMatchDetailsByID(matchID, group.ID); !errors.Is(err, services.ErrMatchNotFound) {
		t.Errorf("GetMatchDetailsByID after delete = %v, want services.ErrMatchNotFound", err)
	}
}

// TestDeleteMatch_Integration_WrongGroupNotFound mirrors
// TestGetMatchDetailsByID_Integration_WrongGroupNotFound: a match ID that
// exists but belongs to a different group must not be deletable (or even
// detectable) through that other group's scope.
func TestDeleteMatch_Integration_WrongGroupNotFound(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	matchService := services.NewMatchService(tx)

	groupA, err := groupService.CreateGroup("Zzz Delete Match Group A", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group A: %v", err)
	}
	groupB, err := groupService.CreateGroup("Zzz Delete Match Group B", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group B: %v", err)
	}

	matchID, err := matchService.CreateMatch(services.MatchSpec{Date: models.Date(time.Now())}, groupA.ID)
	if err != nil {
		t.Fatalf("failed to create match: %v", err)
	}

	if err := matchService.DeleteMatch(matchID, groupB.ID); !errors.Is(err, services.ErrMatchNotFound) {
		t.Errorf("DeleteMatch(match from group A, group B) error = %v, want services.ErrMatchNotFound", err)
	}

	var matchCount int64
	if err := tx.Model(&models.Match{}).Where("id = ?", matchID).Count(&matchCount).Error; err != nil {
		t.Fatalf("failed to count matches: %v", err)
	}
	if matchCount != 1 {
		t.Errorf("expected the match to survive a delete scoped to the wrong group, found %d rows", matchCount)
	}
}

// TestDeleteMatch_Integration_UnknownNotFound covers a match id that doesn't
// exist at all, as opposed to one scoped to the wrong group above.
func TestDeleteMatch_Integration_UnknownNotFound(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	matchService := services.NewMatchService(tx)

	if err := matchService.DeleteMatch(uuid.New(), uuid.New()); !errors.Is(err, services.ErrMatchNotFound) {
		t.Errorf("DeleteMatch(random id) error = %v, want services.ErrMatchNotFound", err)
	}
}
