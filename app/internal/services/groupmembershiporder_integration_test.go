package services_test

import (
	"testing"
	"time"

	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// backdateMembership rewrites a GroupMembership row's CreatedAt so join order
// is deterministic without relying on real-time sleeps between each join —
// the same technique TestResetPassword_Integration_ExpiredTokenFails
// (passwordreset_integration_test.go) uses to pin an otherwise
// clock-dependent branch, there via ExpiresAt, here via CreatedAt.
func backdateMembership(t *testing.T, tx *gorm.DB, groupID, playerID uuid.UUID, at time.Time) {
	t.Helper()
	if err := tx.Model(&models.GroupMembership{}).
		Where("group_id = ? AND player_id = ?", groupID, playerID).
		Update("created_at", at).Error; err != nil {
		t.Fatalf("failed to backdate membership (group %s, player %s): %v", groupID, playerID, err)
	}
}

// TestGetGroupsWithRoleByPlayerID_Integration_OrderedByJoinDate pins
// GetGroupsWithRoleByPlayerID (GET /groups/me's backing query) to a stable,
// meaningful order: oldest membership first — the same ordering
// GetFirstGroupForPlayer already used, which is exactly why that method
// never suffered this bug. Before the fix, the query had no ORDER BY at all,
// so Postgres was free to return the three groups in any order (in practice
// usually physical/insertion order, which is *why* the bug went unnoticed —
// this test creates the memberships out of insertion order via backdating so
// an unordered query can't accidentally still pass) and the frontend's
// resolveActiveGroup() fallback (groups[0]) could flip across page loads.
//
// Run twice with different backdated timestamps within the same test to
// demonstrate the order tracks CreatedAt rather than any incidental
// database/insertion order.
func TestGetGroupsWithRoleByPlayerID_Integration_OrderedByJoinDate(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	playerService := services.NewPlayerService(tx)
	membershipService := services.NewGroupMembershipService(tx)

	playerID, err := playerService.CreatePlayer("Zzz Membership Order Player")
	if err != nil {
		t.Fatalf("failed to create player: %v", err)
	}

	groupA, err := groupService.CreateGroup("Zzz Membership Order Group A", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group A: %v", err)
	}
	groupB, err := groupService.CreateGroup("Zzz Membership Order Group B", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group B: %v", err)
	}
	groupC, err := groupService.CreateGroup("Zzz Membership Order Group C", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group C: %v", err)
	}

	// Join in an order (A, B, C) deliberately different from the backdated
	// CreatedAt order we then assign (C oldest, A middle, B newest), so a
	// query that silently fell back to insertion order (or any other
	// incidental order) would return the wrong sequence.
	for _, g := range []*models.Group{groupA, groupB, groupC} {
		if err := membershipService.AddPlayerToGroup(g.ID, playerID); err != nil {
			t.Fatalf("failed to add player to group %s: %v", g.Name, err)
		}
	}

	base := time.Now().Add(-time.Hour)
	backdateMembership(t, tx, groupC.ID, playerID, base)
	backdateMembership(t, tx, groupA.ID, playerID, base.Add(10*time.Minute))
	backdateMembership(t, tx, groupB.ID, playerID, base.Add(20*time.Minute))

	assertOrder := func(t *testing.T, want []uuid.UUID) {
		t.Helper()
		groups, err := membershipService.GetGroupsWithRoleByPlayerID(playerID)
		if err != nil {
			t.Fatalf("GetGroupsWithRoleByPlayerID returned error: %v", err)
		}
		if len(groups) != len(want) {
			t.Fatalf("GetGroupsWithRoleByPlayerID returned %d groups, want %d: %+v", len(groups), len(want), groups)
		}
		for i, g := range groups {
			if g.ID != want[i] {
				t.Fatalf("GetGroupsWithRoleByPlayerID[%d].ID = %s, want %s (oldest-membership-first order); full result: %+v",
					i, g.ID, want[i], groups)
			}
		}
	}

	wantOrder := []uuid.UUID{groupC.ID, groupA.ID, groupB.ID}

	// Run twice: a flaky/incidental "already sorted" result wouldn't survive
	// two independent calls the same way a real ORDER BY does.
	assertOrder(t, wantOrder)
	assertOrder(t, wantOrder)
}
