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

func TestCreatePlayer_Integration_DuplicateNameCaseInsensitive(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	s := services.NewPlayerService(tx)

	if _, err := s.CreatePlayer("Zzz Integration Bob"); err != nil {
		t.Fatalf("first CreatePlayer returned error: %v", err)
	}

	_, err := s.CreatePlayer("zzz integration bob")
	if !errors.Is(err, services.ErrPlayerAlreadyExists) {
		t.Errorf("CreatePlayer(duplicate, different case) error = %v, want ErrPlayerAlreadyExists", err)
	}
}
