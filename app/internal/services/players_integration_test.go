package services_test

import (
	"errors"
	"testing"

	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/google/uuid"
)

func TestCreatePlayer_Integration_Success(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	s := services.NewPlayerService(tx)

	id, err := s.CreatePlayer("  Zzz Integration Alice  ")
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}
	if id == uuid.Nil {
		t.Fatal("CreatePlayer returned a nil UUID")
	}

	var stored models.Player
	if err := tx.First(&stored, "id = ?", id).Error; err != nil {
		t.Fatalf("failed to load created player: %v", err)
	}
	if stored.Name != "Zzz Integration Alice" {
		t.Errorf("stored name = %q, want trimmed %q", stored.Name, "Zzz Integration Alice")
	}
}

// TestCreatePlayer_Integration_SameNameAnywhereAllowed pins the behavior
// change from the global duplicate-name rejection: Player.Name is no longer
// unique across the whole database (see AuthService.SignupNewPlayer's design
// decision) — PlayerService.CreatePlayer must happily create a second player
// with the same name, case-insensitive collision included. Per-group
// duplicate protection, where it still matters, lives one layer up in
// GroupMembershipService.HasMemberNamed.
func TestCreatePlayer_Integration_SameNameAnywhereAllowed(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	s := services.NewPlayerService(tx)

	firstID, err := s.CreatePlayer("Zzz Integration Bob")
	if err != nil {
		t.Fatalf("first CreatePlayer returned error: %v", err)
	}

	secondID, err := s.CreatePlayer("zzz integration bob")
	if err != nil {
		t.Fatalf("CreatePlayer(same name, different case) returned error: %v, want no error", err)
	}
	if firstID == secondID {
		t.Fatal("expected two distinct players, got the same ID twice")
	}
}

// TestUpdateName_Integration_Success covers the happy path: renaming to a
// genuinely free name succeeds and the change is visible via GetPlayerByID.
func TestUpdateName_Integration_Success(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	s := services.NewPlayerService(tx)

	id, err := s.CreatePlayer("Zzz Integration Rename Before")
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	if err := s.UpdateName(id, "  Zzz Integration Rename After  "); err != nil {
		t.Fatalf("UpdateName returned error: %v, want success", err)
	}

	updated, err := s.GetPlayerByID(id)
	if err != nil {
		t.Fatalf("GetPlayerByID returned error: %v", err)
	}
	if updated.Name != "Zzz Integration Rename After" {
		t.Errorf("Name after rename = %q, want trimmed %q", updated.Name, "Zzz Integration Rename After")
	}
}

// TestUpdateName_Integration_RejectsNameAlreadyUsedByAnotherPlayer covers the
// core product requirement: global uniqueness on rename, including a
// case-only difference — unlike CreatePlayer, which allows this on purpose.
// The original name must remain unchanged.
func TestUpdateName_Integration_RejectsNameAlreadyUsedByAnotherPlayer(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	s := services.NewPlayerService(tx)

	if _, err := s.CreatePlayer("Zzz Integration Rename Marco"); err != nil {
		t.Fatalf("failed to create first player: %v", err)
	}
	targetID, err := s.CreatePlayer("Zzz Integration Rename Luca")
	if err != nil {
		t.Fatalf("failed to create second player: %v", err)
	}

	err = s.UpdateName(targetID, "ZZZ INTEGRATION RENAME MARCO")
	if !errors.Is(err, services.ErrPlayerNameAlreadyUsed) {
		t.Fatalf("UpdateName error = %v, want ErrPlayerNameAlreadyUsed", err)
	}

	unchanged, err := s.GetPlayerByID(targetID)
	if err != nil {
		t.Fatalf("GetPlayerByID returned error: %v", err)
	}
	if unchanged.Name != "Zzz Integration Rename Luca" {
		t.Errorf("Name after rejected rename = %q, want unchanged %q", unchanged.Name, "Zzz Integration Rename Luca")
	}
}

// TestUpdateName_Integration_SameNameExcludesSelf covers the id != ? exclusion:
// renaming to your own current name, or the same name in a different case,
// must not be rejected as a false "already used" collision against yourself.
func TestUpdateName_Integration_SameNameExcludesSelf(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	s := services.NewPlayerService(tx)

	id, err := s.CreatePlayer("Zzz Integration Rename Self")
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	if err := s.UpdateName(id, "Zzz Integration Rename Self"); err != nil {
		t.Fatalf("UpdateName(same name) returned error: %v, want success", err)
	}
	if err := s.UpdateName(id, "ZZZ INTEGRATION RENAME SELF"); err != nil {
		t.Fatalf("UpdateName(same name, different case) returned error: %v, want success", err)
	}

	final, err := s.GetPlayerByID(id)
	if err != nil {
		t.Fatalf("GetPlayerByID returned error: %v", err)
	}
	if final.Name != "ZZZ INTEGRATION RENAME SELF" {
		t.Errorf("Name after self-rename = %q, want %q", final.Name, "ZZZ INTEGRATION RENAME SELF")
	}
}

// TestUpdateName_Integration_RejectsEmptyName covers the trim/empty guard,
// shared with CreatePlayer via the same ErrEmptyPlayerName sentinel.
func TestUpdateName_Integration_RejectsEmptyName(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	s := services.NewPlayerService(tx)

	id, err := s.CreatePlayer("Zzz Integration Rename Empty")
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	for _, name := range []string{"", "   ", "\t\n"} {
		if err := s.UpdateName(id, name); !errors.Is(err, services.ErrEmptyPlayerName) {
			t.Errorf("UpdateName(%q) error = %v, want ErrEmptyPlayerName", name, err)
		}
	}
}
