package services_test

import (
	"testing"

	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"
	"app/pkg/database"
)

// TestReplaceTeamColourWithPosition_Integration replays the deploy that drops
// teams.colour: each group's teams get position 0 and 1 in the order of their
// old colours, which is the order matches listed them in, so no match swaps
// sides; the colour column is gone afterwards, and a later start does nothing.
//
// The reads are raw statements of their own: position is dropped and added
// back, so a statement prepared on teams earlier in this run would fail with
// "cached plan must not change result type". The rollback restores the table.
func TestReplaceTeamColourWithPosition_Integration(t *testing.T) {
	tx := testutil.BeginTx(t, testutil.OpenDB(t))
	groupService := services.NewGroupService(tx)
	teamService := services.NewTeamService(tx)

	// Created "Les Rouges" first, but blue sorted before red: the match page
	// showed "Les Bleus" on the left, and must keep doing so.
	group, err := groupService.CreateGroup("Zzz Team Position Migration", [2]services.TeamSpec{{Name: "Les Rouges"}, {Name: "Les Bleus"}})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	teams, err := teamService.GetTeamsByGroupID(group.ID)
	if err != nil || len(teams) != 2 {
		t.Fatalf("load teams: err=%v teams=%+v", err, teams)
	}
	rouges, bleus := teams[0].ID, teams[1].ID

	// The schema as it was before the deploy.
	for _, statement := range []string{
		`ALTER TABLE teams ADD COLUMN colour text`,
		`ALTER TABLE teams DROP COLUMN position`,
	} {
		if err := tx.Exec(statement).Error; err != nil {
			t.Fatalf("%s: %v", statement, err)
		}
	}
	if err := tx.Exec(`UPDATE teams SET colour = CASE WHEN id = ? THEN 'red' ELSE 'blue' END WHERE group_id = ?`, rouges, group.ID).Error; err != nil {
		t.Fatalf("set colours: %v", err)
	}

	if err := database.ReplaceTeamColourWithPosition(tx); err != nil {
		t.Fatalf("ReplaceTeamColourWithPosition: %v", err)
	}
	positionOf := func(name string) int {
		t.Helper()
		var positions []int
		if err := tx.Raw(`SELECT position FROM teams WHERE group_id = ? AND name = ? /* team position test */`, group.ID, name).Pluck("position", &positions).Error; err != nil || len(positions) != 1 {
			t.Fatalf("read position of %s: err=%v positions=%v", name, err, positions)
		}
		return positions[0]
	}
	if got := positionOf("Les Bleus"); got != 0 {
		t.Errorf("Les Bleus (blue) position = %d, want 0", got)
	}
	if got := positionOf("Les Rouges"); got != 1 {
		t.Errorf("Les Rouges (red) position = %d, want 1", got)
	}
	if tx.Migrator().HasColumn(&models.Team{}, "colour") {
		t.Errorf("teams.colour still exists")
	}

	// A later start: no colour column left, nothing to do.
	if err := tx.Exec(`UPDATE teams SET position = 5 WHERE id = ?`, bleus).Error; err != nil {
		t.Fatalf("set position: %v", err)
	}
	if err := database.ReplaceTeamColourWithPosition(tx); err != nil {
		t.Fatalf("ReplaceTeamColourWithPosition (second start): %v", err)
	}
	if got := positionOf("Les Bleus"); got != 5 {
		t.Errorf("a later start rewrote the position to %d", got)
	}
}
