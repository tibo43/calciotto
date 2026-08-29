package services_test

import (
	"errors"
	"testing"

	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"

	"gorm.io/gorm"
)

// TestRemoveMember_Integration_AdminRemovesPlainMember covers the core
// "admin removes someone else" rule: the target's membership disappears, the
// admin's own role is untouched, and no other member is affected.
func TestRemoveMember_Integration_AdminRemovesPlainMember(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	playerService := services.NewPlayerService(tx)
	membershipService := services.NewGroupMembershipService(tx)

	group, err := groupService.CreateGroup("Zzz Remove Member Group")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	adminID, err := playerService.CreatePlayer("Zzz Remove Member Admin")
	if err != nil {
		t.Fatalf("failed to create admin player: %v", err)
	}
	if err := membershipService.AddPlayerToGroupWithRole(group.ID, adminID, models.RoleAdmin); err != nil {
		t.Fatalf("failed to add admin: %v", err)
	}

	targetID, err := playerService.CreatePlayer("Zzz Remove Member Target")
	if err != nil {
		t.Fatalf("failed to create target player: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(group.ID, targetID); err != nil {
		t.Fatalf("failed to add target member: %v", err)
	}

	otherID, err := playerService.CreatePlayer("Zzz Remove Member Bystander")
	if err != nil {
		t.Fatalf("failed to create bystander player: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(group.ID, otherID); err != nil {
		t.Fatalf("failed to add bystander member: %v", err)
	}

	if err := membershipService.RemoveMember(group.ID, adminID, targetID); err != nil {
		t.Fatalf("RemoveMember returned error: %v", err)
	}

	isMember, err := membershipService.IsMember(group.ID, targetID)
	if err != nil {
		t.Fatalf("IsMember(target) returned error: %v", err)
	}
	if isMember {
		t.Error("target is still a member after RemoveMember")
	}

	adminRole, err := membershipService.GetRole(group.ID, adminID)
	if err != nil {
		t.Fatalf("GetRole(admin) returned error: %v", err)
	}
	if adminRole != models.RoleAdmin {
		t.Errorf("admin role = %q after removing someone else, want unchanged %q", adminRole, models.RoleAdmin)
	}

	bystanderIsMember, err := membershipService.IsMember(group.ID, otherID)
	if err != nil {
		t.Fatalf("IsMember(bystander) returned error: %v", err)
	}
	if !bystanderIsMember {
		t.Error("unrelated bystander lost membership as a side effect of RemoveMember")
	}
}

// TestRemoveMember_Integration_AdminCannotRemoveSelf covers the guard against
// an admin targeting their own membership through this route: it must fail
// with ErrCannotRemoveSelf, and the admin must remain a member/admin.
func TestRemoveMember_Integration_AdminCannotRemoveSelf(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	playerService := services.NewPlayerService(tx)
	membershipService := services.NewGroupMembershipService(tx)

	group, err := groupService.CreateGroup("Zzz Remove Self Group")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	adminID, err := playerService.CreatePlayer("Zzz Remove Self Admin")
	if err != nil {
		t.Fatalf("failed to create admin player: %v", err)
	}
	if err := membershipService.AddPlayerToGroupWithRole(group.ID, adminID, models.RoleAdmin); err != nil {
		t.Fatalf("failed to add admin: %v", err)
	}

	if err := membershipService.RemoveMember(group.ID, adminID, adminID); !errors.Is(err, services.ErrCannotRemoveSelf) {
		t.Fatalf("RemoveMember(self) error = %v, want ErrCannotRemoveSelf", err)
	}

	isMember, err := membershipService.IsMember(group.ID, adminID)
	if err != nil {
		t.Fatalf("IsMember(admin) returned error: %v", err)
	}
	if !isMember {
		t.Error("admin is no longer a member after a refused self-removal")
	}

	role, err := membershipService.GetRole(group.ID, adminID)
	if err != nil {
		t.Fatalf("GetRole(admin) returned error: %v", err)
	}
	if role != models.RoleAdmin {
		t.Errorf("admin role = %q after a refused self-removal, want unchanged %q", role, models.RoleAdmin)
	}
}

// TestRemoveMember_Integration_TargetNotMember covers targeting a player who
// simply isn't a member of the group: the underlying gorm.ErrRecordNotFound
// must propagate as-is, same handling as LeaveGroup.
func TestRemoveMember_Integration_TargetNotMember(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	playerService := services.NewPlayerService(tx)
	membershipService := services.NewGroupMembershipService(tx)

	group, err := groupService.CreateGroup("Zzz Remove NotMember Group")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	adminID, err := playerService.CreatePlayer("Zzz Remove NotMember Admin")
	if err != nil {
		t.Fatalf("failed to create admin player: %v", err)
	}
	if err := membershipService.AddPlayerToGroupWithRole(group.ID, adminID, models.RoleAdmin); err != nil {
		t.Fatalf("failed to add admin: %v", err)
	}

	outsiderID, err := playerService.CreatePlayer("Zzz Remove NotMember Outsider")
	if err != nil {
		t.Fatalf("failed to create outsider player: %v", err)
	}

	if err := membershipService.RemoveMember(group.ID, adminID, outsiderID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("RemoveMember(non-member target) error = %v, want gorm.ErrRecordNotFound", err)
	}

	adminRole, err := membershipService.GetRole(group.ID, adminID)
	if err != nil {
		t.Fatalf("GetRole(admin) returned error: %v", err)
	}
	if adminRole != models.RoleAdmin {
		t.Errorf("admin role = %q after a failed removal, want unchanged %q", adminRole, models.RoleAdmin)
	}
}
