package services_test

import (
	"bytes"
	"errors"
	"log"
	"strings"
	"testing"
	"time"

	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// requestResetToken calls ForgotPassword and digs the raw token back out of the
// server log. That indirection is the point of the current design: the token is
// never returned to the caller and only its SHA-256 lives in the database, so
// the log line written by sendPasswordResetLink is the only place the raw value
// exists — exactly where a developer would look for it in dev today, and the
// line a real mailer will eventually replace.
func requestResetToken(t *testing.T, authService *services.AuthService, email string) string {
	t.Helper()

	logs := captureLogs(t, func() {
		if err := authService.ForgotPassword(email); err != nil {
			t.Fatalf("ForgotPassword(%q) returned error: %v", email, err)
		}
	})

	_, after, found := strings.Cut(logs, "?token=")
	if !found {
		t.Fatalf("no reset link logged for %q, log output was: %q", email, logs)
	}
	token, _, _ := strings.Cut(after, "\n")
	token = strings.TrimSpace(token)
	if token == "" {
		t.Fatalf("logged reset link for %q carries an empty token: %q", email, logs)
	}
	return token
}

func captureLogs(t *testing.T, fn func()) string {
	t.Helper()

	var buf bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(previous)

	fn()
	return buf.String()
}

// newClaimedPlayer creates a player and attaches credentials to it, the state
// every password-reset scenario starts from.
func newClaimedPlayer(t *testing.T, tx *gorm.DB, name, email, password string) uuid.UUID {
	t.Helper()

	playerID, err := services.NewPlayerService(tx).CreatePlayer(name)
	if err != nil {
		t.Fatalf("failed to create player %q: %v", name, err)
	}
	if err := services.NewAuthService(tx, testJWTSecret).Signup(playerID, email, password); err != nil {
		t.Fatalf("failed to sign up player %q: %v", name, err)
	}
	return playerID
}

func TestForgotPassword_Integration_KnownEmailIssuesUsableToken(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testJWTSecret)
	newClaimedPlayer(t, tx, "Zzz Integration Reset Judy", "judy@example.com", "old-pass")

	token := requestResetToken(t, authService, "judy@example.com")

	if err := authService.ResetPassword(token, "brand-new-pass"); err != nil {
		t.Fatalf("ResetPassword with the freshly issued token returned error: %v", err)
	}
}

func TestForgotPassword_Integration_UnknownEmailStaysSilent(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testJWTSecret)

	logs := captureLogs(t, func() {
		if err := authService.ForgotPassword("nobody-zzz-reset@example.com"); err != nil {
			t.Fatalf("ForgotPassword on an unknown email returned error %v, want nil (must not reveal that the email is unregistered)", err)
		}
	})

	if strings.Contains(logs, "?token=") {
		t.Errorf("ForgotPassword issued a reset link for an unregistered email, log output was: %q", logs)
	}
}

func TestResetPassword_Integration_ChangesThePassword(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testJWTSecret)
	newClaimedPlayer(t, tx, "Zzz Integration Reset Karl", "karl@example.com", "old-pass")

	token := requestResetToken(t, authService, "karl@example.com")
	if err := authService.ResetPassword(token, "new-pass"); err != nil {
		t.Fatalf("ResetPassword returned error: %v", err)
	}

	if _, err := authService.Login("karl@example.com", "new-pass"); err != nil {
		t.Errorf("Login with the new password returned error: %v", err)
	}
	if _, err := authService.Login("karl@example.com", "old-pass"); !errors.Is(err, services.ErrInvalidCredentials) {
		t.Errorf("Login with the old password error = %v, want ErrInvalidCredentials", err)
	}
}

func TestResetPassword_Integration_TokenIsSingleUse(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testJWTSecret)
	newClaimedPlayer(t, tx, "Zzz Integration Reset Leo", "leo@example.com", "old-pass")

	token := requestResetToken(t, authService, "leo@example.com")
	if err := authService.ResetPassword(token, "first-new-pass"); err != nil {
		t.Fatalf("first ResetPassword returned error: %v", err)
	}

	err := authService.ResetPassword(token, "second-new-pass")
	if !errors.Is(err, services.ErrInvalidResetToken) {
		t.Errorf("reusing a consumed token error = %v, want ErrInvalidResetToken", err)
	}
	if _, loginErr := authService.Login("leo@example.com", "second-new-pass"); !errors.Is(loginErr, services.ErrInvalidCredentials) {
		t.Error("the second reset went through despite the token already being used")
	}
}

func TestResetPassword_Integration_ExpiredTokenFails(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testJWTSecret)
	playerID := newClaimedPlayer(t, tx, "Zzz Integration Reset Mia", "mia@example.com", "old-pass")

	token := requestResetToken(t, authService, "mia@example.com")

	// Backdating the row is simpler than waiting out the one-hour TTL, and it
	// exercises the same branch the clock would.
	if err := tx.Model(&models.PasswordResetToken{}).
		Where("player_id = ?", playerID).
		Update("expires_at", time.Now().Add(-time.Minute)).Error; err != nil {
		t.Fatalf("failed to backdate the reset token: %v", err)
	}

	err := authService.ResetPassword(token, "new-pass")
	if !errors.Is(err, services.ErrInvalidResetToken) {
		t.Errorf("ResetPassword with an expired token error = %v, want ErrInvalidResetToken", err)
	}
	if _, loginErr := authService.Login("mia@example.com", "old-pass"); loginErr != nil {
		t.Errorf("the expired reset changed the password anyway: Login with the old password returned %v", loginErr)
	}
}

func TestResetPassword_Integration_UnknownTokenFails(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testJWTSecret)

	if err := authService.ResetPassword("not-a-real-token", "new-pass"); !errors.Is(err, services.ErrInvalidResetToken) {
		t.Errorf("ResetPassword with an unknown token error = %v, want ErrInvalidResetToken", err)
	}
	if err := authService.ResetPassword("", "new-pass"); !errors.Is(err, services.ErrInvalidResetToken) {
		t.Errorf("ResetPassword with an empty token error = %v, want ErrInvalidResetToken", err)
	}
}

func TestResetPassword_Integration_EmptyPasswordRejected(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testJWTSecret)
	newClaimedPlayer(t, tx, "Zzz Integration Reset Nina", "nina@example.com", "old-pass")

	token := requestResetToken(t, authService, "nina@example.com")

	if err := authService.ResetPassword(token, ""); !errors.Is(err, services.ErrPasswordRequired) {
		t.Errorf("ResetPassword with an empty password error = %v, want ErrPasswordRequired", err)
	}
	// The rejected attempt must not have burned the token.
	if err := authService.ResetPassword(token, "new-pass"); err != nil {
		t.Errorf("ResetPassword after an empty-password rejection returned error: %v", err)
	}
}

func TestResetPassword_Integration_SuccessInvalidatesOtherOutstandingTokens(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testJWTSecret)
	newClaimedPlayer(t, tx, "Zzz Integration Reset Otto", "otto@example.com", "old-pass")

	firstToken := requestResetToken(t, authService, "otto@example.com")
	secondToken := requestResetToken(t, authService, "otto@example.com")
	if firstToken == secondToken {
		t.Fatal("two ForgotPassword calls produced the same token")
	}

	if err := authService.ResetPassword(firstToken, "new-pass"); err != nil {
		t.Fatalf("ResetPassword with the first token returned error: %v", err)
	}

	err := authService.ResetPassword(secondToken, "hijacked-pass")
	if !errors.Is(err, services.ErrInvalidResetToken) {
		t.Errorf("second outstanding token after a successful reset error = %v, want ErrInvalidResetToken", err)
	}
	if _, loginErr := authService.Login("otto@example.com", "new-pass"); loginErr != nil {
		t.Errorf("Login with the password set by the first reset returned error: %v", loginErr)
	}
}

func TestForgotPassword_Integration_StoresOnlyTheTokenHash(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testJWTSecret)
	playerID := newClaimedPlayer(t, tx, "Zzz Integration Reset Pia", "pia@example.com", "old-pass")

	token := requestResetToken(t, authService, "pia@example.com")

	var stored models.PasswordResetToken
	if err := tx.Where("player_id = ?", playerID).First(&stored).Error; err != nil {
		t.Fatalf("failed to load the stored reset token: %v", err)
	}
	if stored.TokenHash == token {
		t.Error("the raw token was stored in the database, only its SHA-256 digest should be")
	}
	if len(stored.TokenHash) != 64 {
		t.Errorf("stored token hash is %d characters, want 64 (hex-encoded SHA-256)", len(stored.TokenHash))
	}
	if stored.UsedAt != nil {
		t.Errorf("a freshly issued token has UsedAt = %v, want nil", stored.UsedAt)
	}
}
