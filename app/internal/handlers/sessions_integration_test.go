package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"app/internal/handlers"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestBrowserSession_Integration_RotationAndRevocation(t *testing.T) {
	tx := testutil.BeginTx(t, testutil.OpenDB(t))
	auth := services.NewAuthService(tx, testAuthJWTSecret)
	const email = "session-browser@example.test"
	const password = "browser-fixture-password"
	playerID, err := services.NewPlayerService(tx).CreatePlayer("Browser Session Fixture")
	if err != nil {
		t.Fatal(err)
	}
	if err := auth.Signup(playerID, email, password); err != nil {
		t.Fatal(err)
	}

	const origin = "https://app.example.test"
	handler := handlers.NewAuthHandler(auth)
	handler.SessionOrigins = []string{origin}
	handler.SecureCookies = true
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/auth/login", handler.Login)
	router.POST("/auth/refresh", handler.RefreshSession)
	router.POST("/auth/logout", handler.Logout)
	router.GET("/protected", handlers.AuthMiddleware(auth), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	readCredentials := func(rec *httptest.ResponseRecorder) (string, *http.Cookie) {
		t.Helper()
		if rec.Code != http.StatusOK || rec.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("session response has status %d and cache policy %q", rec.Code, rec.Header().Get("Cache-Control"))
		}
		var body struct{ Token string }
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.Token == "" {
			t.Fatal("session response did not contain an access token")
		}
		cookies := rec.Result().Cookies()
		if len(cookies) != 1 || !cookies[0].HttpOnly || !cookies[0].Secure ||
			cookies[0].SameSite != http.SameSiteStrictMode || cookies[0].Path != "/api/auth" {
			t.Fatal("refresh cookie must be secure, HttpOnly and scoped to the auth routes")
		}
		if len(cookies[0].Value) == 0 {
			t.Fatal("empty refresh cookie")
		}
		return body.Token, cookies[0]
	}
	sendSessionRequest := func(path, from string, cookie *http.Cookie, expected string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		req.Header.Set("Origin", from)
		req.Header.Set("X-Session-ID", expected)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	protectedStatus := func(token string) int {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code
	}

	body := map[string]string{"email": email, "password": password}
	first, cookie := readCredentials(postJSON(t, router, "/auth/login", body))
	id, _, _ := strings.Cut(cookie.Value, ".")
	sibling, _ := readCredentials(postJSON(t, router, "/auth/login", body))
	var claims jwt.RegisteredClaims
	if _, _, err := jwt.NewParser().ParseUnverified(first, &claims); err != nil || claims.ExpiresAt == nil ||
		claims.IssuedAt == nil || claims.ExpiresAt.Sub(claims.IssuedAt.Time) != 15*time.Minute {
		t.Fatal("expected a 15-minute access token")
	}
	if res := sendSessionRequest("/auth/refresh", "https://other.example.test", cookie, id); res.Code != http.StatusForbidden {
		t.Fatalf("foreign-origin refresh status = %d, want 403", res.Code)
	}
	if res := sendSessionRequest("/auth/logout", origin, cookie, uuid.NewString()); res.Code != http.StatusConflict {
		t.Fatalf("logout with another tab's session ID returned %d, want 409", res.Code)
	}
	next, rotated := readCredentials(sendSessionRequest("/auth/refresh", origin, cookie, id))
	if rotated.Value == cookie.Value || !rotated.Expires.Equal(cookie.Expires) {
		t.Fatal("renewal must rotate the credential within the fixed session lifetime")
	}
	if res := sendSessionRequest("/auth/refresh", origin, cookie, id); res.Code != http.StatusConflict {
		t.Fatalf("previous refresh credential status = %d, want 409", res.Code)
	}
	if protectedStatus(next) != http.StatusNoContent {
		t.Fatal("the new access token could not access a protected route")
	}
	if res := sendSessionRequest("/auth/logout", origin, rotated, id); res.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d, want 204", res.Code)
	}
	if protectedStatus(next) != http.StatusUnauthorized || protectedStatus(sibling) != http.StatusNoContent {
		t.Fatal("logout must revoke only the session belonging to this device")
	}
	if res := sendSessionRequest("/auth/refresh", origin, rotated, id); res.Code != http.StatusUnauthorized {
		t.Fatalf("renewal after logout status = %d, want 401", res.Code)
	}
}
