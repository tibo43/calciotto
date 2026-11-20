package services_test

import (
	"testing"

	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/google/uuid"
)

// favoriteOf looks up whether groupID is favoriteID's favorite among the
// groups returned by GetGroupsWithRoleByPlayerID — exercising the same query
// the frontend's group selector/profile carousel actually reads from,
// instead of poking at GroupMembership.IsFavorite directly.
func favoriteOf(t *testing.T, membershipService *services.GroupMembershipService, playerID, groupID uuid.UUID) bool {
	t.Helper()
	groups, err := membershipService.GetGroupsWithRoleByPlayerID(playerID)
	if err != nil {
		t.Fatalf("GetGroupsWithRoleByPlayerID returned error: %v", err)
	}
	for _, g := range groups {
		if g.ID == groupID {
			return g.IsFavorite
		}
	}
	t.Fatalf("group %s not found among player's groups", groupID)
	return false
}

// TestAddPlayerToGroupWithRole_Integration_FirstMembershipIsFavorite covers
// the "a player's very first group becomes their favorite automatically"
// rule: with only one candidate, there's no ambiguity about which group that
// should be.
func TestAddPlayerToGroupWithRole_Integration_FirstMembershipIsFavorite(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	playerService := services.NewPlayerService(tx)
	membershipService := services.NewGroupMembershipService(tx)

	groupA, err := groupService.CreateGroup("Zzz Favorite Group A", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group A: %v", err)
	}
	groupB, err := groupService.CreateGroup("Zzz Favorite Group B", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group B: %v", err)
	}

	playerID, err := playerService.CreatePlayer("Zzz Favorite Player")
	if err != nil {
		t.Fatalf("failed to create player: %v", err)
	}

	if err := membershipService.AddPlayerToGroup(groupA.ID, playerID); err != nil {
		t.Fatalf("failed to add player to group A: %v", err)
	}
	if !favoriteOf(t, membershipService, playerID, groupA.ID) {
		t.Error("player's very first group should be their favorite")
	}

	if err := membershipService.AddPlayerToGroup(groupB.ID, playerID); err != nil {
		t.Fatalf("failed to add player to group B: %v", err)
	}
	if favoriteOf(t, membershipService, playerID, groupB.ID) {
		t.Error("a second group should not become the favorite automatically")
	}
	if !favoriteOf(t, membershipService, playerID, groupA.ID) {
		t.Error("group A should still be the favorite after joining group B")
	}
}

// TestSetFavoriteGroup_Integration_MovesTheFlag covers the normal path: the
// caller explicitly picks a new favorite among groups they already belong
// to, and the old one is unset in the same move — never leaving two, or
// zero, favorites at once.
func TestSetFavoriteGroup_Integration_MovesTheFlag(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	playerService := services.NewPlayerService(tx)
	membershipService := services.NewGroupMembershipService(tx)

	groupA, err := groupService.CreateGroup("Zzz SetFavorite Group A", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group A: %v", err)
	}
	groupB, err := groupService.CreateGroup("Zzz SetFavorite Group B", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group B: %v", err)
	}

	playerID, err := playerService.CreatePlayer("Zzz SetFavorite Player")
	if err != nil {
		t.Fatalf("failed to create player: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(groupA.ID, playerID); err != nil {
		t.Fatalf("failed to add player to group A: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(groupB.ID, playerID); err != nil {
		t.Fatalf("failed to add player to group B: %v", err)
	}

	if err := membershipService.SetFavoriteGroup(playerID, groupB.ID); err != nil {
		t.Fatalf("SetFavoriteGroup returned error: %v", err)
	}

	if favoriteOf(t, membershipService, playerID, groupA.ID) {
		t.Error("group A should no longer be the favorite")
	}
	if !favoriteOf(t, membershipService, playerID, groupB.ID) {
		t.Error("group B should now be the favorite")
	}
}

// TestLeaveGroup_Integration_ReassignsFavoriteWhenDeparting covers the
// invariant "a player with at least one group always has exactly one
// favorite" surviving LeaveGroup: leaving a favorited group must promote
// another one of the player's remaining memberships, not leave them with
// zero favorites.
func TestLeaveGroup_Integration_ReassignsFavoriteWhenDeparting(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	playerService := services.NewPlayerService(tx)
	membershipService := services.NewGroupMembershipService(tx)

	favoriteGroup, err := groupService.CreateGroup("Zzz LeaveFavorite Group A", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create favorite group: %v", err)
	}
	otherGroup, err := groupService.CreateGroup("Zzz LeaveFavorite Group B", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create other group: %v", err)
	}

	playerID, err := playerService.CreatePlayer("Zzz LeaveFavorite Player")
	if err != nil {
		t.Fatalf("failed to create player: %v", err)
	}
	// Joins favoriteGroup first (becomes favorite by construction), then
	// otherGroup — both need at least one other member so leaving is allowed
	// (LeaveGroup refuses ErrLastMember otherwise).
	if err := membershipService.AddPlayerToGroupWithRole(favoriteGroup.ID, playerID, models.RoleAdmin); err != nil {
		t.Fatalf("failed to add player to favorite group: %v", err)
	}
	if err := membershipService.AddPlayerToGroupWithRole(otherGroup.ID, playerID, models.RoleAdmin); err != nil {
		t.Fatalf("failed to add player to other group: %v", err)
	}

	otherMemberID, err := playerService.CreatePlayer("Zzz LeaveFavorite Other Member")
	if err != nil {
		t.Fatalf("failed to create other member: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(favoriteGroup.ID, otherMemberID); err != nil {
		t.Fatalf("failed to add other member to favorite group: %v", err)
	}

	if !favoriteOf(t, membershipService, playerID, favoriteGroup.ID) {
		t.Fatal("sanity check failed: favoriteGroup should be the favorite before leaving")
	}

	if err := membershipService.LeaveGroup(favoriteGroup.ID, playerID); err != nil {
		t.Fatalf("LeaveGroup returned error: %v", err)
	}

	if !favoriteOf(t, membershipService, playerID, otherGroup.ID) {
		t.Error("otherGroup should have become the favorite after leaving the previous favorite")
	}
}

// TestRemoveMember_Integration_ReassignsFavoriteWhenRemoved covers the same
// invariant as LeaveGroup, but via an admin removing someone else — the
// removed player's own favorite must still move to a remaining group of
// theirs, not just silently disappear.
func TestRemoveMember_Integration_ReassignsFavoriteWhenRemoved(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	playerService := services.NewPlayerService(tx)
	membershipService := services.NewGroupMembershipService(tx)

	favoriteGroup, err := groupService.CreateGroup("Zzz RemoveFavorite Group A", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create favorite group: %v", err)
	}
	otherGroup, err := groupService.CreateGroup("Zzz RemoveFavorite Group B", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create other group: %v", err)
	}

	adminID, err := playerService.CreatePlayer("Zzz RemoveFavorite Admin")
	if err != nil {
		t.Fatalf("failed to create admin: %v", err)
	}
	if err := membershipService.AddPlayerToGroupWithRole(favoriteGroup.ID, adminID, models.RoleAdmin); err != nil {
		t.Fatalf("failed to add admin to favorite group: %v", err)
	}

	targetID, err := playerService.CreatePlayer("Zzz RemoveFavorite Target")
	if err != nil {
		t.Fatalf("failed to create target player: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(favoriteGroup.ID, targetID); err != nil {
		t.Fatalf("failed to add target to favorite group: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(otherGroup.ID, targetID); err != nil {
		t.Fatalf("failed to add target to other group: %v", err)
	}

	if !favoriteOf(t, membershipService, targetID, favoriteGroup.ID) {
		t.Fatal("sanity check failed: favoriteGroup should be the target's favorite before removal")
	}

	if err := membershipService.RemoveMember(favoriteGroup.ID, adminID, targetID); err != nil {
		t.Fatalf("RemoveMember returned error: %v", err)
	}

	if !favoriteOf(t, membershipService, targetID, otherGroup.ID) {
		t.Error("otherGroup should have become the target's favorite after being removed from the previous favorite")
	}
}
