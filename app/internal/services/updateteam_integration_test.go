package services_test

import (
	"errors"
	"testing"

	"app/internal/services"
	"app/internal/testutil"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TestUpdateTeam_Integration_Success covers the happy path: an admin renames
// a team and changes its colour, and both changes are persisted.
func TestUpdateTeam_Integration_Success(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	teamService := services.NewTeamService(tx)

	group, err := groupService.CreateGroup("Zzz UpdateTeam Group", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("CreateGroup returned error: %v", err)
	}
	teams, err := teamService.GetTeamsByGroupID(group.ID)
	if err != nil || len(teams) != 2 {
		t.Fatalf("failed to load group teams: err=%v teams=%+v", err, teams)
	}

	updated, err := teamService.UpdateTeam(teams[0].ID, group.ID, "Les Rouges", "red")
	if err != nil {
		t.Fatalf("UpdateTeam returned error: %v", err)
	}
	if updated.Name != "Les Rouges" || updated.Colour != "red" {
		t.Errorf("UpdateTeam returned %+v, want Name=%q Colour=%q", updated, "Les Rouges", "red")
	}

	reloaded, err := teamService.GetTeamByID(teams[0].ID)
	if err != nil {
		t.Fatalf("GetTeamByID returned error: %v", err)
	}
	if reloaded.Name != "Les Rouges" || reloaded.Colour != "red" {
		t.Errorf("reloaded team = %+v, want Name=%q Colour=%q", reloaded, "Les Rouges", "red")
	}
}

// TestUpdateTeam_Integration_CrossGroupNotFound pins the security-relevant
// behaviour: a team that exists but belongs to a *different* group than the
// one passed in must behave as gorm.ErrRecordNotFound, not be silently
// updatable. Without this, an admin of group A who merely guesses or learns
// group B's team UUID could rename/recolour it.
func TestUpdateTeam_Integration_CrossGroupNotFound(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	teamService := services.NewTeamService(tx)

	groupA, err := groupService.CreateGroup("Zzz UpdateTeam Group A", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group A: %v", err)
	}
	groupB, err := groupService.CreateGroup("Zzz UpdateTeam Group B", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group B: %v", err)
	}

	teamsA, err := teamService.GetTeamsByGroupID(groupA.ID)
	if err != nil || len(teamsA) != 2 {
		t.Fatalf("failed to load group A teams: err=%v teams=%+v", err, teamsA)
	}

	if _, err := teamService.UpdateTeam(teamsA[0].ID, groupB.ID, "Stolen Name", "purple"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("UpdateTeam(teamA, groupB) error = %v, want gorm.ErrRecordNotFound", err)
	}

	// The team must be untouched by the rejected attempt.
	reloaded, err := teamService.GetTeamByID(teamsA[0].ID)
	if err != nil {
		t.Fatalf("GetTeamByID returned error: %v", err)
	}
	if reloaded.Name == "Stolen Name" || reloaded.Colour == "purple" {
		t.Errorf("cross-group UpdateTeam mutated the team: %+v", reloaded)
	}
}

// TestUpdateTeam_Integration_RequiresNameAndColour covers both validation
// sentinels, matching the style of PlayerService's ErrEmptyPlayerName.
func TestUpdateTeam_Integration_RequiresNameAndColour(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	teamService := services.NewTeamService(tx)

	group, err := groupService.CreateGroup("Zzz UpdateTeam Validation Group", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	teams, err := teamService.GetTeamsByGroupID(group.ID)
	if err != nil || len(teams) != 2 {
		t.Fatalf("failed to load group teams: err=%v teams=%+v", err, teams)
	}

	if _, err := teamService.UpdateTeam(teams[0].ID, group.ID, "   ", "red"); !errors.Is(err, services.ErrTeamNameRequired) {
		t.Errorf("empty name error = %v, want ErrTeamNameRequired", err)
	}
	if _, err := teamService.UpdateTeam(teams[0].ID, group.ID, "Les Rouges", "  "); !errors.Is(err, services.ErrTeamColourRequired) {
		t.Errorf("empty colour error = %v, want ErrTeamColourRequired", err)
	}

	// Neither rejected call should have changed anything.
	reloaded, err := teamService.GetTeamByID(teams[0].ID)
	if err != nil {
		t.Fatalf("GetTeamByID returned error: %v", err)
	}
	if reloaded.Name == "Les Rouges" {
		t.Errorf("validation failure still mutated the team: %+v", reloaded)
	}
}

// TestUpdateTeam_Integration_UnknownTeamNotFound covers a team id that
// doesn't exist at all, as opposed to existing in a different group.
func TestUpdateTeam_Integration_UnknownTeamNotFound(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	teamService := services.NewTeamService(tx)

	group, err := groupService.CreateGroup("Zzz UpdateTeam Unknown Group", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	if _, err := teamService.UpdateTeam(uuid.New(), group.ID, "Name", "colour"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("unknown team id error = %v, want gorm.ErrRecordNotFound", err)
	}
}
