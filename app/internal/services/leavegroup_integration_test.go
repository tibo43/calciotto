package services_test

import (
	"errors"
	"testing"

	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"
)

// TestLeaveGroup_Integration_AdminPromotesOldestRemainingMember covers the
// core "leave a group" rule for an admin: when other members remain, the
// longest-standing one among them (by GroupMembership.CreatedAt, the same
// ordering GetFirstGroupForPlayer uses) must become the new admin, and the
// departing admin must end up with no membership at all.
func TestLeaveGroup_Integration_AdminPromotesOldestRemainingMember(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	playerService := services.NewPlayerService(tx)
	membershipService := services.NewGroupMembershipService(tx)

	group, err := groupService.CreateGroup("Zzz Leave Admin Group", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	adminID, err := playerService.CreatePlayer("Zzz Leave Admin")
	if err != nil {
		t.Fatalf("failed to create admin player: %v", err)
	}
	if err := membershipService.AddPlayerToGroupWithRole(group.ID, adminID, models.RoleAdmin); err != nil {
		t.Fatalf("failed to add admin: %v", err)
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

	if err := membershipService.LeaveGroup(group.ID, adminID); err != nil {
		t.Fatalf("LeaveGroup returned error: %v", err)
	}

	isMember, err := membershipService.IsMember(group.ID, adminID)
	if err != nil {
		t.Fatalf("IsMember(admin) returned error: %v", err)
	}
	if isMember {
		t.Error("former admin is still a member after LeaveGroup")
	}

	newRole, err := membershipService.GetRole(group.ID, earlierMemberID)
	if err != nil {
		t.Fatalf("GetRole(earlierMember) returned error: %v", err)
	}
	if newRole != models.RoleAdmin {
		t.Errorf("earlier member role = %q, want %q (should be promoted)", newRole, models.RoleAdmin)
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
// non-admin leaving: only their own membership should disappear, and the
// existing admin (and any other member) must be left untouched.
func TestLeaveGroup_Integration_PlainMemberLeavesWithoutSideEffects(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	playerService := services.NewPlayerService(tx)
	membershipService := services.NewGroupMembershipService(tx)

	group, err := groupService.CreateGroup("Zzz Leave Member Group", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	adminID, err := playerService.CreatePlayer("Zzz Leave Member Admin")
	if err != nil {
		t.Fatalf("failed to create admin player: %v", err)
	}
	if err := membershipService.AddPlayerToGroupWithRole(group.ID, adminID, models.RoleAdmin); err != nil {
		t.Fatalf("failed to add admin: %v", err)
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

	adminRole, err := membershipService.GetRole(group.ID, adminID)
	if err != nil {
		t.Fatalf("GetRole(admin) returned error: %v", err)
	}
	if adminRole != models.RoleAdmin {
		t.Errorf("admin role changed to %q after an unrelated member left, want unchanged %q", adminRole, models.RoleAdmin)
	}
}

// TestLeaveGroup_Integration_LastMemberCannotLeave covers the block on a
// group's sole member (admin or not) leaving: LeaveGroup must fail with
// ErrLastMember and the membership must remain intact.
func TestLeaveGroup_Integration_LastMemberCannotLeave(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	playerService := services.NewPlayerService(tx)
	membershipService := services.NewGroupMembershipService(tx)

	group, err := groupService.CreateGroup("Zzz Leave Last Member Group", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	soleAdminID, err := playerService.CreatePlayer("Zzz Leave Sole Admin")
	if err != nil {
		t.Fatalf("failed to create sole admin player: %v", err)
	}
	if err := membershipService.AddPlayerToGroupWithRole(group.ID, soleAdminID, models.RoleAdmin); err != nil {
		t.Fatalf("failed to add sole admin: %v", err)
	}

	if err := membershipService.LeaveGroup(group.ID, soleAdminID); !errors.Is(err, services.ErrLastMember) {
		t.Fatalf("LeaveGroup(sole member) error = %v, want ErrLastMember", err)
	}

	isMember, err := membershipService.IsMember(group.ID, soleAdminID)
	if err != nil {
		t.Fatalf("IsMember returned error: %v", err)
	}
	if !isMember {
		t.Error("sole admin is no longer a member after a refused LeaveGroup")
	}
}

// TestLeaveGroup_Integration_AdminLeavesWhileAnotherAdminRemains covers what
// multiple admins per group changed: when the departing admin isn't the last
// one, there is nothing to hand over — the remaining admin keeps their role
// and no plain member gets promoted behind their back.
func TestLeaveGroup_Integration_AdminLeavesWhileAnotherAdminRemains(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	playerService := services.NewPlayerService(tx)
	membershipService := services.NewGroupMembershipService(tx)

	group, err := groupService.CreateGroup("Zzz Leave Two Admins Group", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	leavingAdminID, err := playerService.CreatePlayer("Zzz Leave Two Admins Leaving")
	if err != nil {
		t.Fatalf("failed to create leaving admin player: %v", err)
	}
	if err := membershipService.AddPlayerToGroupWithRole(group.ID, leavingAdminID, models.RoleAdmin); err != nil {
		t.Fatalf("failed to add leaving admin: %v", err)
	}

	// A plain member joined before the second admin, so it is the one
	// LeaveGroup would promote if it wrongly considered the group adminless.
	memberID, err := playerService.CreatePlayer("Zzz Leave Two Admins Member")
	if err != nil {
		t.Fatalf("failed to create member player: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(group.ID, memberID); err != nil {
		t.Fatalf("failed to add member: %v", err)
	}

	remainingAdminID, err := playerService.CreatePlayer("Zzz Leave Two Admins Remaining")
	if err != nil {
		t.Fatalf("failed to create remaining admin player: %v", err)
	}
	if err := membershipService.AddPlayerToGroupWithRole(group.ID, remainingAdminID, models.RoleAdmin); err != nil {
		t.Fatalf("failed to add remaining admin: %v", err)
	}

	if err := membershipService.LeaveGroup(group.ID, leavingAdminID); err != nil {
		t.Fatalf("LeaveGroup returned error: %v", err)
	}

	isMember, err := membershipService.IsMember(group.ID, leavingAdminID)
	if err != nil {
		t.Fatalf("IsMember(leavingAdmin) returned error: %v", err)
	}
	if isMember {
		t.Error("departing admin is still a member after LeaveGroup")
	}

	remainingRole, err := membershipService.GetRole(group.ID, remainingAdminID)
	if err != nil {
		t.Fatalf("GetRole(remainingAdmin) returned error: %v", err)
	}
	if remainingRole != models.RoleAdmin {
		t.Errorf("remaining admin role = %q, want unchanged %q", remainingRole, models.RoleAdmin)
	}

	memberRole, err := membershipService.GetRole(group.ID, memberID)
	if err != nil {
		t.Fatalf("GetRole(member) returned error: %v", err)
	}
	if memberRole != models.RoleMember {
		t.Errorf("plain member role = %q, want unchanged %q — no promotion was needed", memberRole, models.RoleMember)
	}
}
