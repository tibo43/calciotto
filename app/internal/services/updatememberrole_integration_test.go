package services_test

import (
	"errors"
	"testing"

	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// roleTestGroup builds a group with one admin and one plain member, the
// starting point every UpdateMemberRole case below varies from.
func roleTestGroup(t *testing.T, tx *gorm.DB, name string) (*services.GroupMembershipService, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()

	groupService := services.NewGroupService(tx)
	playerService := services.NewPlayerService(tx)
	membershipService := services.NewGroupMembershipService(tx)

	group, err := groupService.CreateGroup(name)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	adminID, err := playerService.CreatePlayer(name + " Admin")
	if err != nil {
		t.Fatalf("failed to create admin player: %v", err)
	}
	if err := membershipService.AddPlayerToGroupWithRole(group.ID, adminID, models.RoleAdmin); err != nil {
		t.Fatalf("failed to add admin: %v", err)
	}

	memberID, err := playerService.CreatePlayer(name + " Member")
	if err != nil {
		t.Fatalf("failed to create member player: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(group.ID, memberID); err != nil {
		t.Fatalf("failed to add member: %v", err)
	}

	return membershipService, group.ID, adminID, memberID
}

// TestUpdateMemberRole_Integration_PromoteAndDemote covers the round trip an
// admin can perform on someone else: promoting a plain member to admin (the
// only way a group gets a second admin at all), then demoting them back while
// the original admin still holds the role.
func TestUpdateMemberRole_Integration_PromoteAndDemote(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	memberships, groupID, adminID, memberID := roleTestGroup(t, tx, "Zzz Role Update Promote")

	if err := memberships.UpdateMemberRole(groupID, adminID, memberID, models.RoleAdmin); err != nil {
		t.Fatalf("UpdateMemberRole(promote) returned error: %v", err)
	}
	role, err := memberships.GetRole(groupID, memberID)
	if err != nil {
		t.Fatalf("GetRole(member) returned error: %v", err)
	}
	if role != models.RoleAdmin {
		t.Fatalf("promoted member role = %q, want %q", role, models.RoleAdmin)
	}

	// The promoting admin keeps their own role: promotion isn't a hand-off.
	adminRole, err := memberships.GetRole(groupID, adminID)
	if err != nil {
		t.Fatalf("GetRole(admin) returned error: %v", err)
	}
	if adminRole != models.RoleAdmin {
		t.Errorf("acting admin role = %q after promoting someone, want unchanged %q", adminRole, models.RoleAdmin)
	}

	if err := memberships.UpdateMemberRole(groupID, adminID, memberID, models.RoleMember); err != nil {
		t.Fatalf("UpdateMemberRole(demote) returned error: %v", err)
	}
	role, err = memberships.GetRole(groupID, memberID)
	if err != nil {
		t.Fatalf("GetRole(member) returned error: %v", err)
	}
	if role != models.RoleMember {
		t.Errorf("demoted member role = %q, want %q", role, models.RoleMember)
	}
}

// TestUpdateMemberRole_Integration_SameRoleIsNoOp pins the documented choice
// for a redundant request: setting the role the target already has succeeds
// and changes nothing, so a retried call never fails for no reason.
func TestUpdateMemberRole_Integration_SameRoleIsNoOp(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	memberships, groupID, adminID, memberID := roleTestGroup(t, tx, "Zzz Role Update NoOp")

	if err := memberships.UpdateMemberRole(groupID, adminID, memberID, models.RoleMember); err != nil {
		t.Fatalf("UpdateMemberRole(member -> member) returned error: %v, want nil", err)
	}
	role, err := memberships.GetRole(groupID, memberID)
	if err != nil {
		t.Fatalf("GetRole(member) returned error: %v", err)
	}
	if role != models.RoleMember {
		t.Errorf("member role = %q after a no-op update, want unchanged %q", role, models.RoleMember)
	}
}

// TestUpdateMemberRole_Integration_InvalidRole covers a role value outside the
// two the model allows: refused with ErrInvalidRole before any write.
func TestUpdateMemberRole_Integration_InvalidRole(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	memberships, groupID, adminID, memberID := roleTestGroup(t, tx, "Zzz Role Update Invalid")

	if err := memberships.UpdateMemberRole(groupID, adminID, memberID, "superadmin"); !errors.Is(err, services.ErrInvalidRole) {
		t.Fatalf("UpdateMemberRole(invalid role) error = %v, want ErrInvalidRole", err)
	}
	role, err := memberships.GetRole(groupID, memberID)
	if err != nil {
		t.Fatalf("GetRole(member) returned error: %v", err)
	}
	if role != models.RoleMember {
		t.Errorf("member role = %q after a refused update, want unchanged %q", role, models.RoleMember)
	}
}

// TestUpdateMemberRole_Integration_CannotTargetSelf covers the self-targeting
// guard: there is no self-service step down, so an admin naming themselves is
// refused with ErrCannotChangeOwnRole.
func TestUpdateMemberRole_Integration_CannotTargetSelf(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	memberships, groupID, adminID, _ := roleTestGroup(t, tx, "Zzz Role Update Self")

	if err := memberships.UpdateMemberRole(groupID, adminID, adminID, models.RoleMember); !errors.Is(err, services.ErrCannotChangeOwnRole) {
		t.Fatalf("UpdateMemberRole(self) error = %v, want ErrCannotChangeOwnRole", err)
	}
	role, err := memberships.GetRole(groupID, adminID)
	if err != nil {
		t.Fatalf("GetRole(admin) returned error: %v", err)
	}
	if role != models.RoleAdmin {
		t.Errorf("admin role = %q after a refused self-update, want unchanged %q", role, models.RoleAdmin)
	}
}

// TestUpdateMemberRole_Integration_TargetNotMember covers a target who isn't
// in the group at all: gorm.ErrRecordNotFound propagates as-is, same as
// RemoveMember, and the handler turns it into a 404.
func TestUpdateMemberRole_Integration_TargetNotMember(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	memberships, groupID, adminID, _ := roleTestGroup(t, tx, "Zzz Role Update Outsider")

	outsiderID, err := services.NewPlayerService(tx).CreatePlayer("Zzz Role Update Outsider Player")
	if err != nil {
		t.Fatalf("failed to create outsider player: %v", err)
	}

	if err := memberships.UpdateMemberRole(groupID, adminID, outsiderID, models.RoleAdmin); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("UpdateMemberRole(non-member target) error = %v, want gorm.ErrRecordNotFound", err)
	}
}

// TestUpdateMemberRole_Integration_CannotDemoteLastAdmin covers the invariant
// LeaveGroup also upholds: a group with members always keeps at least one
// admin. Note this can't be reached through PATCH
// /groups/:id/members/:playerId/role — the acting admin is themselves an admin
// and can't target themselves, so another admin always survives — so it's
// exercised at the service level, with a plain member as the acting player.
func TestUpdateMemberRole_Integration_CannotDemoteLastAdmin(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	memberships, groupID, adminID, memberID := roleTestGroup(t, tx, "Zzz Role Update Last Admin")

	if err := memberships.UpdateMemberRole(groupID, memberID, adminID, models.RoleMember); !errors.Is(err, services.ErrLastAdmin) {
		t.Fatalf("UpdateMemberRole(demote last admin) error = %v, want ErrLastAdmin", err)
	}
	role, err := memberships.GetRole(groupID, adminID)
	if err != nil {
		t.Fatalf("GetRole(admin) returned error: %v", err)
	}
	if role != models.RoleAdmin {
		t.Errorf("admin role = %q after a refused demotion, want unchanged %q", role, models.RoleAdmin)
	}

	// With a second admin in place the same demotion is allowed: it's the
	// count of remaining admins that decides, not the identity of the target.
	if err := memberships.UpdateMemberRole(groupID, adminID, memberID, models.RoleAdmin); err != nil {
		t.Fatalf("UpdateMemberRole(promote) returned error: %v", err)
	}
	if err := memberships.UpdateMemberRole(groupID, memberID, adminID, models.RoleMember); err != nil {
		t.Fatalf("UpdateMemberRole(demote with another admin left) returned error: %v", err)
	}
	role, err = memberships.GetRole(groupID, adminID)
	if err != nil {
		t.Fatalf("GetRole(admin) returned error: %v", err)
	}
	if role != models.RoleMember {
		t.Errorf("demoted admin role = %q, want %q", role, models.RoleMember)
	}
}
