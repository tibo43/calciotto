package services_test

import (
	"testing"
	"time"

	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"
)

func lastSeenOf(t *testing.T, s *services.AuthService, name string) *time.Time {
	t.Helper()
	var player models.Player
	if err := s.DB.Select("last_seen_at").Take(&player, "name = ?", name).Error; err != nil {
		t.Fatalf("failed to load player: %v", err)
	}
	return player.LastSeenAt
}

// TouchLastSeen writes the first activity, skips writes within
// LastSeenThrottle, and writes again once that has passed.
func TestTouchLastSeen_Integration(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	const name = "Zzz Integration Last Seen"
	playerID, err := services.NewPlayerService(tx).CreatePlayer(name)
	if err != nil {
		t.Fatalf("CreatePlayer: %v", err)
	}
	auth := services.NewAuthService(tx, "test-secret")

	if got := lastSeenOf(t, auth, name); got != nil {
		t.Fatalf("a new player has last_seen_at = %v, want NULL", got)
	}

	first := time.Date(2026, 11, 27, 10, 0, 0, 0, time.UTC)
	if err := auth.TouchLastSeen(playerID, first); err != nil {
		t.Fatalf("TouchLastSeen: %v", err)
	}
	if got := lastSeenOf(t, auth, name); got == nil || !got.Equal(first) {
		t.Fatalf("last_seen_at = %v, want %v", got, first)
	}

	// Within the throttle, from this process or another one (a fresh
	// service has an empty cache, so the SQL guard is what holds).
	soon := first.Add(services.LastSeenThrottle - time.Second)
	for _, s := range []*services.AuthService{auth, services.NewAuthService(tx, "test-secret")} {
		if err := s.TouchLastSeen(playerID, soon); err != nil {
			t.Fatalf("TouchLastSeen: %v", err)
		}
		if got := lastSeenOf(t, auth, name); !got.Equal(first) {
			t.Fatalf("last_seen_at = %v within the throttle, want it kept at %v", got, first)
		}
	}

	later := first.Add(services.LastSeenThrottle + time.Second)
	if err := auth.TouchLastSeen(playerID, later); err != nil {
		t.Fatalf("TouchLastSeen: %v", err)
	}
	if got := lastSeenOf(t, auth, name); got == nil || !got.Equal(later) {
		t.Fatalf("last_seen_at = %v after the throttle, want %v", got, later)
	}
}
