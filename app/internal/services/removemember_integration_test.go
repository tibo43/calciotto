package services_test

import (
	"errors"
	"testing"

	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"

	"gorm.io/gorm"
)

// TestRemoveMember_Integration_OwnerRemovesPlainMember covers the core
// "owner removes someone else" rule: the target's membership disappears, the
// owner's own role is untouched, and no other member is affected.
func TestRemoveMember_Integration_OwnerRemovesPlainMember(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	playerService := services.NewPlayerService(tx)
	membershipService := services.NewGroupMembershipService(tx)

	group, err := groupService.CreateGroup("Zzz Remove Member Group")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	ownerID, err := playerService.CreatePlayer("Zzz Remove Member Owner")
	if err != nil {
		t.Fatalf("failed to create owner player: %v", err)
	}
	if err := membershipService.AddPlayerToGroupWithRole(group.ID, ownerID, models.RoleOwner); err != nil {
		t.Fatalf("failed to add owner: %v", err)
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

	if err := membershipService.RemoveMember(group.ID, ownerID, targetID); err != nil {
		t.Fatalf("RemoveMember returned error: %v", err)
	}

	isMember, err := membershipService.IsMember(group.ID, targetID)
	if err != nil {
		t.Fatalf("IsMember(target) returned error: %v", err)
	}
	if isMember {
		t.Error("target is still a member after RemoveMember")
	}

	ownerRole, err := membershipService.GetRole(group.ID, ownerID)
	if err != nil {
		t.Fatalf("GetRole(owner) returned error: %v", err)
	}
	if ownerRole != models.RoleOwner {
		t.Errorf("owner role = %q after removing someone else, want unchanged %q", ownerRole, models.RoleOwner)
	}

	bystanderIsMember, err := membershipService.IsMember(group.ID, otherID)
	if err != nil {
		t.Fatalf("IsMember(bystander) returned error: %v", err)
	}
	if !bystanderIsMember {
		t.Error("unrelated bystander lost membership as a side effect of RemoveMember")
	}
}

// TestRemoveMember_Integration_OwnerCannotRemoveSelf covers the guard against
// an owner targeting their own membership through this route: it must fail
// with ErrCannotRemoveSelf, and the owner must remain a member/owner.
func TestRemoveMember_Integration_OwnerCannotRemoveSelf(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	playerService := services.NewPlayerService(tx)
	membershipService := services.NewGroupMembershipService(tx)

	group, err := groupService.CreateGroup("Zzz Remove Self Group")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	ownerID, err := playerService.CreatePlayer("Zzz Remove Self Owner")
	if err != nil {
		t.Fatalf("failed to create owner player: %v", err)
	}
	if err := membershipService.AddPlayerToGroupWithRole(group.ID, ownerID, models.RoleOwner); err != nil {
		t.Fatalf("failed to add owner: %v", err)
	}

	if err := membershipService.RemoveMember(group.ID, ownerID, ownerID); !errors.Is(err, services.ErrCannotRemoveSelf) {
		t.Fatalf("RemoveMember(self) error = %v, want ErrCannotRemoveSelf", err)
	}

	isMember, err := membershipService.IsMember(group.ID, ownerID)
	if err != nil {
		t.Fatalf("IsMember(owner) returned error: %v", err)
	}
	if !isMember {
		t.Error("owner is no longer a member after a refused self-removal")
	}

	role, err := membershipService.GetRole(group.ID, ownerID)
	if err != nil {
		t.Fatalf("GetRole(owner) returned error: %v", err)
	}
	if role != models.RoleOwner {
		t.Errorf("owner role = %q after a refused self-removal, want unchanged %q", role, models.RoleOwner)
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

	ownerID, err := playerService.CreatePlayer("Zzz Remove NotMember Owner")
	if err != nil {
		t.Fatalf("failed to create owner player: %v", err)
	}
	if err := membershipService.AddPlayerToGroupWithRole(group.ID, ownerID, models.RoleOwner); err != nil {
		t.Fatalf("failed to add owner: %v", err)
	}

	outsiderID, err := playerService.CreatePlayer("Zzz Remove NotMember Outsider")
	if err != nil {
		t.Fatalf("failed to create outsider player: %v", err)
	}

	if err := membershipService.RemoveMember(group.ID, ownerID, outsiderID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("RemoveMember(non-member target) error = %v, want gorm.ErrRecordNotFound", err)
	}

	ownerRole, err := membershipService.GetRole(group.ID, ownerID)
	if err != nil {
		t.Fatalf("GetRole(owner) returned error: %v", err)
	}
	if ownerRole != models.RoleOwner {
		t.Errorf("owner role = %q after a failed removal, want unchanged %q", ownerRole, models.RoleOwner)
	}
}
