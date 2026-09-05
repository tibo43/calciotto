package handlers_test

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"app/internal/handlers"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/gin-gonic/gin"
)

func newPasswordResetTestRouter(authService *services.AuthService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	authHandler := handlers.NewAuthHandler(authService)
	router.POST("/auth/signup", authHandler.Signup)
	router.POST("/auth/login", authHandler.Login)
	router.POST("/auth/forgot-password", authHandler.ForgotPassword)
	router.POST("/auth/reset-password", authHandler.ResetPassword)
	return router
}

func postJSON(t *testing.T, router *gin.Engine, path string, body map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to encode request body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(encoded))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// The reset link is only logged for now (no mail transport), so the HTTP tests
// read the token back out of the server log just like the service tests do.
func forgotPasswordAndCaptureToken(t *testing.T, router *gin.Engine, email string) (*httptest.ResponseRecorder, string) {
	t.Helper()

	var logs bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&logs)
	rec := postJSON(t, router, "/auth/forgot-password", map[string]string{"email": email})
	log.SetOutput(previous)

	_, after, found := strings.Cut(logs.String(), "?token=")
	if !found {
		return rec, ""
	}
	token, _, _ := strings.Cut(after, "\n")
	return rec, strings.TrimSpace(token)
}

func TestPasswordResetEndpoints_Integration_ForgotThenReset(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testAuthJWTSecret)
	router := newPasswordResetTestRouter(authService)

	signupRec := postJSON(t, router, "/auth/signup", map[string]string{
		"name":        "Zzz Integration Handler Quinn",
		"email":       "quinn@example.com",
		"password":    "old-pass",
		"invite_code": newSignupInviteCode(t, tx),
	})
	if signupRec.Code != http.StatusOK {
		t.Fatalf("signup returned status %d, body: %s", signupRec.Code, signupRec.Body.String())
	}

	forgotRec, token := forgotPasswordAndCaptureToken(t, router, "quinn@example.com")
	if forgotRec.Code != http.StatusOK {
		t.Fatalf("forgot-password returned status %d, body: %s", forgotRec.Code, forgotRec.Body.String())
	}
	if token == "" {
		t.Fatal("no reset token was logged for a registered email")
	}

	// A bad token is a 400, and the message must not say why it failed.
	badRec := postJSON(t, router, "/auth/reset-password", map[string]string{
		"token":        "not-a-real-token",
		"new_password": "new-pass",
	})
	if badRec.Code != http.StatusBadRequest {
		t.Errorf("reset-password with a bad token returned status %d, want 400, body: %s", badRec.Code, badRec.Body.String())
	}

	goodRec := postJSON(t, router, "/auth/reset-password", map[string]string{
		"token":        token,
		"new_password": "new-pass",
	})
	if goodRec.Code != http.StatusOK {
		t.Fatalf("reset-password with a valid token returned status %d, body: %s", goodRec.Code, goodRec.Body.String())
	}

	loginRec := postJSON(t, router, "/auth/login", map[string]string{
		"email":    "quinn@example.com",
		"password": "new-pass",
	})
	if loginRec.Code != http.StatusOK {
		t.Errorf("login with the reset password returned status %d, want 200, body: %s", loginRec.Code, loginRec.Body.String())
	}

	// The consumed token must not work a second time.
	replayRec := postJSON(t, router, "/auth/reset-password", map[string]string{
		"token":        token,
		"new_password": "another-pass",
	})
	if replayRec.Code != http.StatusBadRequest {
		t.Errorf("replaying a consumed token returned status %d, want 400, body: %s", replayRec.Code, replayRec.Body.String())
	}
}

func TestPasswordResetEndpoints_Integration_ForgotPasswordAlwaysReturnsTheSameAnswer(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testAuthJWTSecret)
	router := newPasswordResetTestRouter(authService)

	signupRec := postJSON(t, router, "/auth/signup", map[string]string{
		"name":        "Zzz Integration Handler Rita",
		"email":       "rita@example.com",
		"password":    "old-pass",
		"invite_code": newSignupInviteCode(t, tx),
	})
	if signupRec.Code != http.StatusOK {
		t.Fatalf("signup returned status %d, body: %s", signupRec.Code, signupRec.Body.String())
	}

	knownRec, _ := forgotPasswordAndCaptureToken(t, router, "rita@example.com")
	unknownRec, unknownToken := forgotPasswordAndCaptureToken(t, router, "nobody-zzz-handler@example.com")

	if knownRec.Code != http.StatusOK {
		t.Errorf("forgot-password for a registered email returned status %d, want 200", knownRec.Code)
	}
	if unknownRec.Code != http.StatusOK {
		t.Errorf("forgot-password for an unregistered email returned status %d, want 200 (must not reveal that the email is unknown)", unknownRec.Code)
	}
	if knownRec.Body.String() != unknownRec.Body.String() {
		t.Errorf("forgot-password bodies differ between a known and an unknown email:\n known:   %s\n unknown: %s", knownRec.Body.String(), unknownRec.Body.String())
	}
	if unknownToken != "" {
		t.Error("forgot-password issued a reset link for an unregistered email")
	}
}

func TestPasswordResetEndpoints_Integration_EmptyPasswordReturns400(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testAuthJWTSecret)
	router := newPasswordResetTestRouter(authService)

	rec := postJSON(t, router, "/auth/reset-password", map[string]string{
		"token":        "whatever",
		"new_password": "",
	})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("reset-password with an empty password returned status %d, want 400, body: %s", rec.Code, rec.Body.String())
	}
}
