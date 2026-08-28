package services_test

import (
	"errors"
	"testing"
	"time"

	"app/internal/services"
	"app/internal/testutil"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const testJWTSecret = "zzz-integration-test-secret"

func TestSignup_Integration_Success(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	playerService := services.NewPlayerService(tx)
	authService := services.NewAuthService(tx, testJWTSecret)

	playerID, err := playerService.CreatePlayer("Zzz Integration Auth Alice")
	if err != nil {
		t.Fatalf("failed to create player: %v", err)
	}

	if err := authService.Signup(playerID, "  Alice@Example.com ", "s3cret-pass"); err != nil {
		t.Fatalf("Signup returned error: %v", err)
	}

	token, err := authService.Login("alice@example.com", "s3cret-pass")
	if err != nil {
		t.Fatalf("Login after successful signup returned error: %v", err)
	}
	if token == "" {
		t.Fatal("Login returned an empty token")
	}
}

func TestSignup_Integration_AlreadyClaimedPlayerFails(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	playerService := services.NewPlayerService(tx)
	authService := services.NewAuthService(tx, testJWTSecret)

	playerID, err := playerService.CreatePlayer("Zzz Integration Auth Bob")
	if err != nil {
		t.Fatalf("failed to create player: %v", err)
	}

	if err := authService.Signup(playerID, "bob@example.com", "s3cret-pass"); err != nil {
		t.Fatalf("first Signup returned error: %v", err)
	}

	err = authService.Signup(playerID, "bob-other@example.com", "another-pass")
	if !errors.Is(err, services.ErrPlayerAlreadyClaimed) {
		t.Errorf("second Signup on same player error = %v, want ErrPlayerAlreadyClaimed", err)
	}
}

func TestSignup_Integration_EmailAlreadyUsedByAnotherPlayerFails(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	playerService := services.NewPlayerService(tx)
	authService := services.NewAuthService(tx, testJWTSecret)

	carolID, err := playerService.CreatePlayer("Zzz Integration Auth Carol")
	if err != nil {
		t.Fatalf("failed to create player carol: %v", err)
	}
	daveID, err := playerService.CreatePlayer("Zzz Integration Auth Dave")
	if err != nil {
		t.Fatalf("failed to create player dave: %v", err)
	}

	if err := authService.Signup(carolID, "shared@example.com", "s3cret-pass"); err != nil {
		t.Fatalf("Signup for carol returned error: %v", err)
	}

	err = authService.Signup(daveID, "SHARED@example.com", "another-pass")
	if !errors.Is(err, services.ErrEmailAlreadyUsed) {
		t.Errorf("Signup for dave with carol's email error = %v, want ErrEmailAlreadyUsed", err)
	}
}

func TestSignup_Integration_UnknownPlayerFails(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testJWTSecret)

	err := authService.Signup(uuid.New(), "ghost@example.com", "s3cret-pass")
	if !errors.Is(err, services.ErrPlayerNotFound) {
		t.Errorf("Signup for unknown player error = %v, want ErrPlayerNotFound", err)
	}
}

func TestLogin_Integration_WrongPasswordFails(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	playerService := services.NewPlayerService(tx)
	authService := services.NewAuthService(tx, testJWTSecret)

	playerID, err := playerService.CreatePlayer("Zzz Integration Auth Erin")
	if err != nil {
		t.Fatalf("failed to create player: %v", err)
	}
	if err := authService.Signup(playerID, "erin@example.com", "correct-pass"); err != nil {
		t.Fatalf("Signup returned error: %v", err)
	}

	_, err = authService.Login("erin@example.com", "wrong-pass")
	if !errors.Is(err, services.ErrInvalidCredentials) {
		t.Errorf("Login with wrong password error = %v, want ErrInvalidCredentials", err)
	}
}

func TestLogin_Integration_UnknownEmailFails(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testJWTSecret)

	_, err := authService.Login("nobody-zzz@example.com", "whatever")
	if !errors.Is(err, services.ErrInvalidCredentials) {
		t.Errorf("Login with unknown email error = %v, want ErrInvalidCredentials (must not leak which case failed)", err)
	}
}

func TestLogin_Integration_TokenContainsPlayerID(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	playerService := services.NewPlayerService(tx)
	authService := services.NewAuthService(tx, testJWTSecret)

	playerID, err := playerService.CreatePlayer("Zzz Integration Auth Frank")
	if err != nil {
		t.Fatalf("failed to create player: %v", err)
	}
	if err := authService.Signup(playerID, "frank@example.com", "s3cret-pass"); err != nil {
		t.Fatalf("Signup returned error: %v", err)
	}

	token, err := authService.Login("frank@example.com", "s3cret-pass")
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}

	decodedID, err := authService.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken on a freshly issued token returned error: %v", err)
	}
	if decodedID != playerID {
		t.Errorf("ParseToken player_id = %s, want %s", decodedID, playerID)
	}
}

func TestParseToken_Integration_RejectsInvalidAndExpiredTokens(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testJWTSecret)

	if _, err := authService.ParseToken("not-a-valid-jwt"); !errors.Is(err, services.ErrInvalidToken) {
		t.Errorf("ParseToken(garbage) error = %v, want ErrInvalidToken", err)
	}

	expiredClaims := jwt.MapClaims{
		"player_id": uuid.New().String(),
		"exp":       jwt.NewNumericDate(time.Now().Add(-time.Hour)),
	}
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	signedExpired, err := expiredToken.SignedString([]byte(testJWTSecret))
	if err != nil {
		t.Fatalf("failed to sign expired test token: %v", err)
	}
	if _, err := authService.ParseToken(signedExpired); !errors.Is(err, services.ErrInvalidToken) {
		t.Errorf("ParseToken(expired) error = %v, want ErrInvalidToken", err)
	}

	wrongSecretToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"player_id": uuid.New().String(),
	})
	signedWrongSecret, err := wrongSecretToken.SignedString([]byte("a-different-secret"))
	if err != nil {
		t.Fatalf("failed to sign wrong-secret test token: %v", err)
	}
	if _, err := authService.ParseToken(signedWrongSecret); !errors.Is(err, services.ErrInvalidToken) {
		t.Errorf("ParseToken(wrong secret) error = %v, want ErrInvalidToken", err)
	}
}
