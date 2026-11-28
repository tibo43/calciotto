package services_test

import (
	"bytes"
	"errors"
	"log"
	"net/url"
	"strings"
	"testing"
	"time"

	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// requestResetToken captures the link from the test mailer.
func requestResetToken(t *testing.T, authService *services.AuthService, email string) string {
	t.Helper()

	var link string
	authService.PasswordResetMailer = func(_, deliveredLink, _ string) error {
		link = deliveredLink
		return nil
	}
	if err := authService.ForgotPassword(email); err != nil {
		t.Fatalf("ForgotPassword returned error: %v", err)
	}
	resetURL, err := url.Parse(link)
	if err != nil {
		t.Fatal("the test mailer received an invalid reset link")
	}
	token := resetURL.Query().Get("token")
	if token == "" {
		t.Fatal("the test mailer did not receive a reset token")
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
	authService.PasswordResetMailer = func(_, _, _ string) error {
		t.Fatal("the test mailer was called for an unknown email")
		return nil
	}
	if err := authService.ForgotPassword("nobody-zzz-reset@example.com"); err != nil {
		t.Fatalf("ForgotPassword on an unknown email returned error %v, want nil", err)
	}
}

func TestForgotPassword_Integration_MailerFailureKeepsLinkOutOfLogs(t *testing.T) {
	tx := testutil.BeginTx(t, testutil.OpenDB(t))
	const email = "reset-delivery-failure@example.com"
	newClaimedPlayer(t, tx, "Zzz Reset Delivery Failure", email, "old-pass")

	authService := services.NewAuthService(tx, testJWTSecret)
	authService.PasswordResetMailer = func(recipient, link, _ string) error {
		return errors.New(recipient + " " + link)
	}
	logs := captureLogs(t, func() {
		if err := authService.ForgotPassword(email); err != nil {
			t.Fatalf("ForgotPassword returned error: %v", err)
		}
	})
	if !strings.Contains(logs, "email delivery failed") || strings.Contains(logs, "token=") || strings.Contains(logs, email) {
		t.Fatal("delivery failures must log only a generic error")
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

// TestResetPassword_Integration_InvalidatesExistingTokens pins the actual
// point of bumping TokenVersion on a successful reset: a JWT issued *before*
// the reset must stop authenticating afterwards, the same way it would if it
// had simply expired. Without this, a stolen token would survive its
// victim's own password reset for the rest of its 7-day life.
func TestResetPassword_Integration_InvalidatesExistingTokens(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testJWTSecret)
	playerID := newClaimedPlayer(t, tx, "Zzz Integration Reset Mira", "mira@example.com", "old-pass")

	tokenBeforeReset, err := authService.Login("mira@example.com", "old-pass")
	if err != nil {
		t.Fatalf("Login before reset returned error: %v", err)
	}
	if _, err := authService.ParseToken(tokenBeforeReset); err != nil {
		t.Fatalf("ParseToken on the pre-reset token returned error: %v", err)
	}

	resetToken := requestResetToken(t, authService, "mira@example.com")
	if err := authService.ResetPassword(resetToken, "new-pass"); err != nil {
		t.Fatalf("ResetPassword returned error: %v", err)
	}

	if _, err := authService.ParseToken(tokenBeforeReset); !errors.Is(err, services.ErrInvalidToken) {
		t.Errorf("ParseToken on the pre-reset token after a reset error = %v, want ErrInvalidToken", err)
	}

	tokenAfterReset, err := authService.Login("mira@example.com", "new-pass")
	if err != nil {
		t.Fatalf("Login after reset returned error: %v", err)
	}
	if gotID, err := authService.ParseToken(tokenAfterReset); err != nil {
		t.Errorf("ParseToken on the post-reset token returned error: %v", err)
	} else if gotID != playerID {
		t.Errorf("ParseToken on the post-reset token = %s, want %s", gotID, playerID)
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

// TestResetPassword_Integration_PasswordPolicy is the reset half of the
// minimum-length rule: a 7-character new password is ErrPasswordTooShort and
// leaves the old password (and the token) usable, an 8-character one goes
// through. The token must survive the rejection — burning it on a validation
// error would force the player to request a whole new email over a typo.
func TestResetPassword_Integration_PasswordPolicy(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testJWTSecret)
	newClaimedPlayer(t, tx, "Zzz Integration Reset Short", "reset-short@example.com", "old-pass")

	token := requestResetToken(t, authService, "reset-short@example.com")

	if err := authService.ResetPassword(token, "7chars!"); !errors.Is(err, services.ErrPasswordTooShort) {
		t.Errorf("ResetPassword with a 7-character password error = %v, want ErrPasswordTooShort", err)
	}
	if _, err := authService.Login("reset-short@example.com", "old-pass"); err != nil {
		t.Errorf("the old password stopped working after a rejected reset: %v", err)
	}

	if err := authService.ResetPassword(token, "8chars!!"); err != nil {
		t.Fatalf("ResetPassword with an 8-character password returned error: %v", err)
	}
	if _, err := authService.Login("reset-short@example.com", "8chars!!"); err != nil {
		t.Errorf("login with the new 8-character password returned error: %v", err)
	}
}

func TestForgotPassword_Integration_EmailInThePlayersLanguage(t *testing.T) {
	tx := testutil.BeginTx(t, testutil.OpenDB(t))
	const email = "reset-language@example.com"
	player := newClaimedPlayer(t, tx, "Zzz Reset Language", email, "old-pass")
	authService := services.NewAuthService(tx, testJWTSecret)

	var locale string
	authService.PasswordResetMailer = func(_, _, got string) error {
		locale = got
		return nil
	}
	ask := func(acceptLanguage string) string {
		locale = ""
		if err := authService.ForgotPasswordFrom(email, "", acceptLanguage); err != nil {
			t.Fatalf("ForgotPasswordFrom returned error: %v", err)
		}
		return locale
	}

	if got := ask(""); got != "en" {
		t.Errorf("no language anywhere: %q, want en", got)
	}
	if got := ask("fr-FR,fr;q=0.9"); got != "fr" {
		t.Errorf("browser language: %q, want fr", got)
	}
	german := "de"
	if _, err := services.NewPlayerService(tx).UpdateLocale(player, &german); err != nil {
		t.Fatalf("save language: %v", err)
	}
	if got := ask("fr-FR,fr;q=0.9"); got != "de" {
		t.Errorf("saved language beats the browser's: %q, want de", got)
	}
}
