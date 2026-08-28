package services_test

import (
	"errors"
	"testing"

	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"
)

// TestLeaveGroup_Integration_OwnerPromotesOldestRemainingMember covers the
// core "leave a group" rule for an owner: when other members remain, the
// longest-standing one among them (by GroupMembership.CreatedAt, the same
// ordering GetFirstGroupForPlayer uses) must become the new owner, and the
// departing owner must end up with no membership at all.
func TestLeaveGroup_Integration_OwnerPromotesOldestRemainingMember(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	playerService := services.NewPlayerService(tx)
	membershipService := services.NewGroupMembershipService(tx)

	group, err := groupService.CreateGroup("Zzz Leave Owner Group")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	ownerID, err := playerService.CreatePlayer("Zzz Leave Owner")
	if err != nil {
		t.Fatalf("failed to create owner player: %v", err)
	}
	if err := membershipService.AddPlayerToGroupWithRole(group.ID, ownerID, models.RoleOwner); err != nil {
		t.Fatalf("failed to add owner: %v", err)
	}

	// Two remaining members added in order — earlierMemberID joins first, so
	// it must be the one promoted, not laterMemberID.
	earlierMemberID, err := playerService.CreatePlayer("Zzz Leave Earlier Member")
	if err != nil {
		t.Fatalf("failed to create earlier member player: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(group.ID, earlierMemberID); err != nil {
		t.Fatalf("failed to add earlier member: %v", err)
	}

	laterMemberID, err := playerService.CreatePlayer("Zzz Leave Later Member")
	if err != nil {
		t.Fatalf("failed to create later member player: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(group.ID, laterMemberID); err != nil {
		t.Fatalf("failed to add later member: %v", err)
	}

	if err := membershipService.LeaveGroup(group.ID, ownerID); err != nil {
		t.Fatalf("LeaveGroup returned error: %v", err)
	}

	isMember, err := membershipService.IsMember(group.ID, ownerID)
	if err != nil {
		t.Fatalf("IsMember(owner) returned error: %v", err)
	}
	if isMember {
		t.Error("former owner is still a member after LeaveGroup")
	}

	newRole, err := membershipService.GetRole(group.ID, earlierMemberID)
	if err != nil {
		t.Fatalf("GetRole(earlierMember) returned error: %v", err)
	}
	if newRole != models.RoleOwner {
		t.Errorf("earlier member role = %q, want %q (should be promoted)", newRole, models.RoleOwner)
	}

	laterRole, err := membershipService.GetRole(group.ID, laterMemberID)
	if err != nil {
		t.Fatalf("GetRole(laterMember) returned error: %v", err)
	}
	if laterRole != models.RoleMember {
		t.Errorf("later member role = %q, want %q (should stay a plain member)", laterRole, models.RoleMember)
	}
}

// TestLeaveGroup_Integration_PlainMemberLeavesWithoutSideEffects covers a
// non-owner leaving: only their own membership should disappear, and the
// existing owner (and any other member) must be left untouched.
func TestLeaveGroup_Integration_PlainMemberLeavesWithoutSideEffects(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	playerService := services.NewPlayerService(tx)
	membershipService := services.NewGroupMembershipService(tx)

	group, err := groupService.CreateGroup("Zzz Leave Member Group")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	ownerID, err := playerService.CreatePlayer("Zzz Leave Member Owner")
	if err != nil {
		t.Fatalf("failed to create owner player: %v", err)
	}
	if err := membershipService.AddPlayerToGroupWithRole(group.ID, ownerID, models.RoleOwner); err != nil {
		t.Fatalf("failed to add owner: %v", err)
	}

	leavingMemberID, err := playerService.CreatePlayer("Zzz Leave Member Leaving")
	if err != nil {
		t.Fatalf("failed to create leaving member player: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(group.ID, leavingMemberID); err != nil {
		t.Fatalf("failed to add leaving member: %v", err)
	}

	if err := membershipService.LeaveGroup(group.ID, leavingMemberID); err != nil {
		t.Fatalf("LeaveGroup returned error: %v", err)
	}

	isMember, err := membershipService.IsMember(group.ID, leavingMemberID)
	if err != nil {
		t.Fatalf("IsMember(leavingMember) returned error: %v", err)
	}
	if isMember {
		t.Error("member is still a member after LeaveGroup")
	}

	ownerRole, err := membershipService.GetRole(group.ID, ownerID)
	if err != nil {
		t.Fatalf("GetRole(owner) returned error: %v", err)
	}
	if ownerRole != models.RoleOwner {
		t.Errorf("owner role changed to %q after an unrelated member left, want unchanged %q", ownerRole, models.RoleOwner)
	}
}

// TestLeaveGroup_Integration_LastMemberCannotLeave covers the block on a
// group's sole member (owner or not) leaving: LeaveGroup must fail with
// ErrLastMember and the membership must remain intact.
func TestLeaveGroup_Integration_LastMemberCannotLeave(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	playerService := services.NewPlayerService(tx)
	membershipService := services.NewGroupMembershipService(tx)

	group, err := groupService.CreateGroup("Zzz Leave Last Member Group")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	soleOwnerID, err := playerService.CreatePlayer("Zzz Leave Sole Owner")
	if err != nil {
		t.Fatalf("failed to create sole owner player: %v", err)
	}
	if err := membershipService.AddPlayerToGroupWithRole(group.ID, soleOwnerID, models.RoleOwner); err != nil {
		t.Fatalf("failed to add sole owner: %v", err)
	}

	if err := membershipService.LeaveGroup(group.ID, soleOwnerID); !errors.Is(err, services.ErrLastMember) {
		t.Fatalf("LeaveGroup(sole member) error = %v, want ErrLastMember", err)
	}

	isMember, err := membershipService.IsMember(group.ID, soleOwnerID)
	if err != nil {
		t.Fatalf("IsMember returned error: %v", err)
	}
	if !isMember {
		t.Error("sole owner is no longer a member after a refused LeaveGroup")
	}
}
