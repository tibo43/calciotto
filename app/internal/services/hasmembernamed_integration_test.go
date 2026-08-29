package services_test

import (
	"testing"

	"app/internal/services"
	"app/internal/testutil"
)

// TestHasMemberNamed_Integration covers GroupMembershipService.HasMemberNamed's
// contract as the soft, per-group duplicate guard PlayerHandler.CreatePlayer
// relies on: a case-insensitive match within the group is a hit, the same
// name sitting only in a different group is not, and an empty group reports
// no match at all.
func TestHasMemberNamed_Integration(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	playerService := services.NewPlayerService(tx)
	membershipService := services.NewGroupMembershipService(tx)

	groupA, err := groupService.CreateGroup("Zzz HasMemberNamed Group A", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group A: %v", err)
	}
	groupB, err := groupService.CreateGroup("Zzz HasMemberNamed Group B", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group B: %v", err)
	}

	marcoID, err := playerService.CreatePlayer("Zzz HasMemberNamed Marco")
	if err != nil {
		t.Fatalf("failed to create player marco: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(groupA.ID, marcoID); err != nil {
		t.Fatalf("failed to add marco to group A: %v", err)
	}

	// True: a case-insensitive match exists in the group.
	has, err := membershipService.HasMemberNamed(groupA.ID, "zzz hasmembernamed marco")
	if err != nil {
		t.Fatalf("HasMemberNamed(groupA, case-insensitive match) returned error: %v", err)
	}
	if !has {
		t.Error("HasMemberNamed(groupA, case-insensitive match) = false, want true")
	}

	// False: the same name exists, but only in a different group.
	has, err = membershipService.HasMemberNamed(groupB.ID, "Zzz HasMemberNamed Marco")
	if err != nil {
		t.Fatalf("HasMemberNamed(groupB, name only in group A) returned error: %v", err)
	}
	if has {
		t.Error("HasMemberNamed(groupB, name only in group A) = true, want false")
	}

	// False: the group has no such member at all.
	has, err = membershipService.HasMemberNamed(groupA.ID, "Zzz HasMemberNamed Nobody")
	if err != nil {
		t.Fatalf("HasMemberNamed(groupA, no such member) returned error: %v", err)
	}
	if has {
		t.Error("HasMemberNamed(groupA, no such member) = true, want false")
	}
}
