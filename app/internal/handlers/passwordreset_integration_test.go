package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func forgotPasswordAndCaptureToken(t *testing.T, authService *services.AuthService, router *gin.Engine, email string) (*httptest.ResponseRecorder, string) {
	t.Helper()

	var token string
	authService.PasswordResetMailer = func(_, link, _ string) error {
		resetURL, err := url.Parse(link)
		if err != nil {
			t.Fatal("the test mailer received an invalid reset link")
		}
		token = resetURL.Query().Get("token")
		return nil
	}
	rec := postJSON(t, router, "/auth/forgot-password", map[string]string{"email": email})
	return rec, token
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

	forgotRec, token := forgotPasswordAndCaptureToken(t, authService, router, "quinn@example.com")
	if forgotRec.Code != http.StatusOK {
		t.Fatalf("forgot-password returned status %d, body: %s", forgotRec.Code, forgotRec.Body.String())
	}
	if token == "" {
		t.Fatal("the test mailer did not receive a reset token for a registered email")
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

	knownRec, _ := forgotPasswordAndCaptureToken(t, authService, router, "rita@example.com")
	unknownRec, unknownToken := forgotPasswordAndCaptureToken(t, authService, router, "nobody-zzz-handler@example.com")

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

func TestForgotPasswordWithoutMailerReturnsServiceUnavailable(t *testing.T) {
	t.Setenv("BREVO_API_KEY", "")
	// Missing configuration must be detected before any database lookup.
	router := newPasswordResetTestRouter(services.NewAuthService(nil, testAuthJWTSecret))
	rec := postJSON(t, router, "/auth/forgot-password", map[string]string{"email": "player@example.com"})
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("forgot-password returned status %d, want 503", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid error response: %v", err)
	}
	if body["error"] != services.ErrPasswordResetUnavailable.Error() {
		t.Fatal("forgot-password returned an unexpected error")
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

// TestPasswordResetEndpoints_Integration_TooShortPasswordReturns400 is the
// reset-side mirror of the empty-password case above: a 7-character new
// password is rejected with 400 and the message naming the minimum, not the
// default 500 an unmapped sentinel would produce. The token is deliberately a
// dummy — the length check runs before the token is even looked up, so this
// pins the mapping without needing a real reset link.
func TestPasswordResetEndpoints_Integration_TooShortPasswordReturns400(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testAuthJWTSecret)
	router := newPasswordResetTestRouter(authService)

	rec := postJSON(t, router, "/auth/reset-password", map[string]string{
		"token":        "whatever",
		"new_password": "7chars!",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("reset-password with a 7-character password returned status %d, want 400, body: %s", rec.Code, rec.Body.String())
	}
	var errBody map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &errBody); err != nil {
		t.Fatalf("failed to unmarshal the error body: %v", err)
	}
	if errBody["error"] != services.ErrPasswordTooShort.Error() {
		t.Errorf("error body = %q, want %q", errBody["error"], services.ErrPasswordTooShort.Error())
	}
}

// While two frontends run side by side, the reset link goes back to the
// allowed frontend the request came from; any other origin gets
// FRONTEND_BASE_URL.
func TestPasswordResetEndpoints_Integration_LinkFollowsAllowedOrigin(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	t.Setenv("FRONTEND_BASE_URL", "https://calcio-otto.vercel.app")

	authService := services.NewAuthService(tx, testAuthJWTSecret)
	// A fresh handler per request: the per-account reset limit allows three.
	newRouter := func() *gin.Engine {
		gin.SetMode(gin.TestMode)
		router := gin.New()
		authHandler := handlers.NewAuthHandler(authService)
		authHandler.SessionOrigins = []string{"https://calcio-otto.vercel.app", "https://club.agoran.app"}
		router.POST("/auth/signup", authHandler.Signup)
		router.POST("/auth/forgot-password", authHandler.ForgotPassword)
		return router
	}

	signupRec := postJSON(t, newRouter(), "/auth/signup", map[string]string{
		"name":        "Zzz Integration Handler Domain",
		"email":       "domain-reset@example.com",
		"password":    "old-pass-domain",
		"invite_code": newSignupInviteCode(t, tx),
	})
	if signupRec.Code != http.StatusOK {
		t.Fatalf("signup returned status %d, body: %s", signupRec.Code, signupRec.Body.String())
	}

	for _, tt := range []struct{ origin, wantBase string }{
		{"https://club.agoran.app", "https://club.agoran.app"},
		{"https://calcio-otto.vercel.app", "https://calcio-otto.vercel.app"},
		{"https://evil.example", "https://calcio-otto.vercel.app"},
		{"", "https://calcio-otto.vercel.app"},
	} {
		var link string
		authService.PasswordResetMailer = func(_, delivered, _ string) error {
			link = delivered
			return nil
		}
		encoded, _ := json.Marshal(map[string]string{"email": "domain-reset@example.com"})
		req := httptest.NewRequest(http.MethodPost, "/auth/forgot-password", bytes.NewReader(encoded))
		req.Header.Set("Content-Type", "application/json")
		if tt.origin != "" {
			req.Header.Set("Origin", tt.origin)
		}
		rec := httptest.NewRecorder()
		newRouter().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("origin %q: forgot-password returned status %d", tt.origin, rec.Code)
		}
		parsed, err := url.Parse(link)
		if err != nil || parsed.Scheme+"://"+parsed.Host != tt.wantBase || parsed.Path != "/reset-password" {
			t.Fatalf("origin %q: reset link %q, want it on %s/reset-password", tt.origin, link, tt.wantBase)
		}
	}
}

// The reset email is written in the language of the browser that asked for it
// (the signed-out screens follow the browser), English when it names none the
// app speaks.
func TestPasswordResetEndpoints_Integration_EmailFollowsTheBrowserLanguage(t *testing.T) {
	tx := testutil.BeginTx(t, testutil.OpenDB(t))
	authService := services.NewAuthService(tx, testAuthJWTSecret)
	router := newPasswordResetTestRouter(authService)
	signupRec := postJSON(t, router, "/auth/signup", map[string]string{
		"name":        "Zzz Integration Handler Lang",
		"email":       "lang-reset@example.com",
		"password":    "old-pass",
		"invite_code": newSignupInviteCode(t, tx),
	})
	if signupRec.Code != http.StatusOK {
		t.Fatalf("signup returned status %d, body: %s", signupRec.Code, signupRec.Body.String())
	}

	for _, tt := range []struct{ header, want string }{
		{"de-DE,de;q=0.9,en;q=0.5", "de"},
		{"pt-BR,pt;q=0.9", "en"},
		{"", "en"},
	} {
		var locale string
		authService.PasswordResetMailer = func(_, _, got string) error {
			locale = got
			return nil
		}
		encoded, _ := json.Marshal(map[string]string{"email": "lang-reset@example.com"})
		req := httptest.NewRequest(http.MethodPost, "/auth/forgot-password", bytes.NewReader(encoded))
		req.Header.Set("Content-Type", "application/json")
		if tt.header != "" {
			req.Header.Set("Accept-Language", tt.header)
		}
		rec := httptest.NewRecorder()
		newPasswordResetTestRouter(authService).ServeHTTP(rec, req)
		if rec.Code != http.StatusOK || locale != tt.want {
			t.Errorf("Accept-Language %q: status %d, email language %q, want %q", tt.header, rec.Code, locale, tt.want)
		}
	}
}
