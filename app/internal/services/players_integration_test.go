package services_test

import (
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
