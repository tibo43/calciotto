package services

import (
	"errors"
	"testing"
)

func TestCreatePlayer_EmptyName(t *testing.T) {
	// The empty/whitespace check happens before any DB access in CreatePlayer,
	// so this is safe to exercise without a real *gorm.DB.
	s := &PlayerService{}

	for _, name := range []string{"", "   ", "\t\n"} {
		if _, err := s.CreatePlayer(name); !errors.Is(err, ErrEmptyPlayerName) {
			t.Errorf("CreatePlayer(%q) error = %v, want ErrEmptyPlayerName", name, err)
		}
	}
}
