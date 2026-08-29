package services_test

import (
	"errors"
	"strings"
	"testing"

	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// newGhostPlayer creates a Player with a name but no credentials — the state an
// admin-created roster entry is in, and the only state InviteExistingPlayer
// accepts.
func newGhostPlayer(t *testing.T, tx *gorm.DB, name string) uuid.UUID {
	t.Helper()

	playerID, err := services.NewPlayerService(tx).CreatePlayer(name)
	if err != nil {
		t.Fatalf("failed to create ghost player %q: %v", name, err)
	}
	return playerID
}

// inviteAndCaptureToken calls InviteExistingPlayer and digs the raw token back
// out of the server log, for the same reason requestResetToken does: the token
// is never returned to the caller, and only its digest reaches the database.
func inviteAndCaptureToken(t *testing.T, authService *services.AuthService, playerID uuid.UUID, email string) string {
	t.Helper()

	logs := captureLogs(t, func() {
		if err := authService.InviteExistingPlayer(playerID, email); err != nil {
			t.Fatalf("InviteExistingPlayer(%v, %q) returned error: %v", playerID, email, err)
		}
	})

	_, after, found := strings.Cut(logs, "?token=")
	if !found {
		t.Fatalf("no invite link logged for %q, log output was: %q", email, logs)
	}
	token, _, _ := strings.Cut(after, "\n")
	token = strings.TrimSpace(token)
	if token == "" {
		t.Fatalf("logged invite link for %q carries an empty token: %q", email, logs)
	}
	return token
}

// TestInviteExistingPlayer_Integration_GhostBecomesAnAccount is the whole point
// of the feature: after the invite, the ghost's link sets a password through
// the ordinary ResetPassword path and the player can log in.
func TestInviteExistingPlayer_Integration_GhostBecomesAnAccount(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testJWTSecret)
	playerID := newGhostPlayer(t, tx, "Zzz Integration Invite Rosa")

	token := inviteAndCaptureToken(t, authService, playerID, "invite-rosa@example.com")

	if err := authService.ResetPassword(token, "chosen-pass"); err != nil {
		t.Fatalf("ResetPassword with the invite token returned error: %v", err)
	}
	if _, err := authService.Login("invite-rosa@example.com", "chosen-pass"); err != nil {
		t.Errorf("Login after claiming the invite returned error: %v", err)
	}

	var player models.Player
	if err := tx.First(&player, "id = ?", playerID).Error; err != nil {
		t.Fatalf("failed to reload the invited player: %v", err)
	}
	if player.Email == nil || *player.Email != "invite-rosa@example.com" {
		t.Errorf("invited player email = %v, want %q", player.Email, "invite-rosa@example.com")
	}
}

// TestInviteExistingPlayer_Integration_NormalizesEmail covers the same
// normalizeEmail contract every other AuthService entry point honours — an
// admin typing a capitalized address must still be able to log in with the
// lowercase one.
func TestInviteExistingPlayer_Integration_NormalizesEmail(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testJWTSecret)
	playerID := newGhostPlayer(t, tx, "Zzz Integration Invite Sven")

	token := inviteAndCaptureToken(t, authService, playerID, "  Invite-Sven@Example.COM ")
	if err := authService.ResetPassword(token, "chosen-pass"); err != nil {
		t.Fatalf("ResetPassword with the invite token returned error: %v", err)
	}
	if _, err := authService.Login("invite-sven@example.com", "chosen-pass"); err != nil {
		t.Errorf("Login with the normalized email returned error: %v", err)
	}
}

// TestInviteExistingPlayer_Integration_AlreadyClaimedRejected covers the
// takeover guard: a player who already has an account must not have their email
// rewritten, nor a claim link issued, by an admin of one of their groups.
func TestInviteExistingPlayer_Integration_AlreadyClaimedRejected(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testJWTSecret)
	playerID := newClaimedPlayer(t, tx, "Zzz Integration Invite Tara", "invite-tara@example.com", "her-own-pass")

	logs := captureLogs(t, func() {
		err := authService.InviteExistingPlayer(playerID, "attacker@example.com")
		if !errors.Is(err, services.ErrPlayerAlreadyClaimed) {
			t.Fatalf("InviteExistingPlayer on a claimed player error = %v, want ErrPlayerAlreadyClaimed", err)
		}
	})
	if strings.Contains(logs, "?token=") {
		t.Errorf("a rejected invite still issued a link, log output was: %q", logs)
	}

	var player models.Player
	if err := tx.First(&player, "id = ?", playerID).Error; err != nil {
		t.Fatalf("failed to reload the player: %v", err)
	}
	if player.Email == nil || *player.Email != "invite-tara@example.com" {
		t.Errorf("player email = %v after a refused invite, want unchanged %q", player.Email, "invite-tara@example.com")
	}

	var tokenCount int64
	if err := tx.Model(&models.PasswordResetToken{}).Where("player_id = ?", playerID).Count(&tokenCount).Error; err != nil {
		t.Fatalf("failed to count reset tokens: %v", err)
	}
	if tokenCount != 0 {
		t.Errorf("a refused invite issued %d reset token(s), want 0", tokenCount)
	}
	if _, err := authService.Login("invite-tara@example.com", "her-own-pass"); err != nil {
		t.Errorf("the original account stopped working after a refused invite: %v", err)
	}
}

// TestInviteExistingPlayer_Integration_EmailAlreadyUsed covers the same
// uniqueness guard Signup and SignupNewPlayer apply: two players can't share an
// address, and the failure must leave the ghost untouched.
func TestInviteExistingPlayer_Integration_EmailAlreadyUsed(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testJWTSecret)
	newClaimedPlayer(t, tx, "Zzz Integration Invite Ulf", "invite-ulf@example.com", "his-own-pass")
	ghostID := newGhostPlayer(t, tx, "Zzz Integration Invite Vera")

	err := authService.InviteExistingPlayer(ghostID, "invite-ulf@example.com")
	if !errors.Is(err, services.ErrEmailAlreadyUsed) {
		t.Fatalf("InviteExistingPlayer with a taken email error = %v, want ErrEmailAlreadyUsed", err)
	}

	var ghost models.Player
	if err := tx.First(&ghost, "id = ?", ghostID).Error; err != nil {
		t.Fatalf("failed to reload the ghost player: %v", err)
	}
	if ghost.Email != nil {
		t.Errorf("ghost email = %v after a refused invite, want nil", *ghost.Email)
	}
}

func TestInviteExistingPlayer_Integration_EmptyEmailRejected(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testJWTSecret)
	ghostID := newGhostPlayer(t, tx, "Zzz Integration Invite Wilma")

	for _, email := range []string{"", "   "} {
		if err := authService.InviteExistingPlayer(ghostID, email); !errors.Is(err, services.ErrEmailRequired) {
			t.Errorf("InviteExistingPlayer with email %q error = %v, want ErrEmailRequired", email, err)
		}
	}
}

func TestInviteExistingPlayer_Integration_UnknownPlayerRejected(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testJWTSecret)

	err := authService.InviteExistingPlayer(uuid.New(), "invite-nobody@example.com")
	if !errors.Is(err, services.ErrPlayerNotFound) {
		t.Errorf("InviteExistingPlayer for an unknown player error = %v, want ErrPlayerNotFound", err)
	}
}
