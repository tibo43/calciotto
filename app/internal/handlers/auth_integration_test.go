package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"app/internal/handlers"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/gin-gonic/gin"
)

const testAuthJWTSecret = "zzz-integration-test-secret"

func newAuthTestRouter(authService *services.AuthService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	authHandler := handlers.NewAuthHandler(authService)
	router.POST("/auth/signup", authHandler.Signup)
	router.POST("/auth/login", authHandler.Login)
	router.GET("/protected", handlers.AuthMiddleware(authService), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"player_id": c.MustGet("player_id")})
	})
	return router
}

func TestAuthEndpoints_Integration_SignupThenLogin(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	playerService := services.NewPlayerService(tx)
	authService := services.NewAuthService(tx, testAuthJWTSecret)
	router := newAuthTestRouter(authService)

	playerID, err := playerService.CreatePlayer("Zzz Integration Handler Grace")
	if err != nil {
		t.Fatalf("failed to create player: %v", err)
	}

	signupBody, _ := json.Marshal(map[string]string{
		"player_id": playerID.String(),
		"email":     "grace@example.com",
		"password":  "s3cret-pass",
	})
	signupReq := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader(signupBody))
	signupReq.Header.Set("Content-Type", "application/json")
	signupRec := httptest.NewRecorder()
	router.ServeHTTP(signupRec, signupReq)
	if signupRec.Code != http.StatusOK {
		t.Fatalf("signup returned status %d, body: %s", signupRec.Code, signupRec.Body.String())
	}

	loginBody, _ := json.Marshal(map[string]string{
		"email":    "grace@example.com",
		"password": "s3cret-pass",
	})
	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login returned status %d, body: %s", loginRec.Code, loginRec.Body.String())
	}

	var loginResp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(loginRec.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("failed to decode login response: %v", err)
	}
	if loginResp.Token == "" {
		t.Fatal("login response has an empty token")
	}

	// A valid token must be accepted by the middleware, with the right player_id injected.
	protectedReq := httptest.NewRequest(http.MethodGet, "/protected", nil)
	protectedReq.Header.Set("Authorization", "Bearer "+loginResp.Token)
	protectedRec := httptest.NewRecorder()
	router.ServeHTTP(protectedRec, protectedReq)
	if protectedRec.Code != http.StatusOK {
		t.Fatalf("protected route with valid token returned status %d, body: %s", protectedRec.Code, protectedRec.Body.String())
	}
	var protectedResp struct {
		PlayerID string `json:"player_id"`
	}
	if err := json.Unmarshal(protectedRec.Body.Bytes(), &protectedResp); err != nil {
		t.Fatalf("failed to decode protected response: %v", err)
	}
	if protectedResp.PlayerID != playerID.String() {
		t.Errorf("protected route injected player_id = %q, want %q", protectedResp.PlayerID, playerID.String())
	}
}

func TestAuthMiddleware_Integration_RejectsMissingAndInvalidTokens(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testAuthJWTSecret)
	router := newAuthTestRouter(authService)

	noTokenReq := httptest.NewRequest(http.MethodGet, "/protected", nil)
	noTokenRec := httptest.NewRecorder()
	router.ServeHTTP(noTokenRec, noTokenReq)
	if noTokenRec.Code != http.StatusUnauthorized {
		t.Errorf("protected route with no token returned status %d, want 401", noTokenRec.Code)
	}

	invalidTokenReq := httptest.NewRequest(http.MethodGet, "/protected", nil)
	invalidTokenReq.Header.Set("Authorization", "Bearer not-a-valid-jwt")
	invalidTokenRec := httptest.NewRecorder()
	router.ServeHTTP(invalidTokenRec, invalidTokenReq)
	if invalidTokenRec.Code != http.StatusUnauthorized {
		t.Errorf("protected route with invalid token returned status %d, want 401", invalidTokenRec.Code)
	}
}

func TestAuthHandler_Integration_SignupErrors(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	playerService := services.NewPlayerService(tx)
	authService := services.NewAuthService(tx, testAuthJWTSecret)
	router := newAuthTestRouter(authService)

	// Unknown player_id -> 404.
	unknownBody, _ := json.Marshal(map[string]string{
		"player_id": "00000000-0000-0000-0000-000000000000",
		"email":     "ghost@example.com",
		"password":  "s3cret-pass",
	})
	unknownReq := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader(unknownBody))
	unknownReq.Header.Set("Content-Type", "application/json")
	unknownRec := httptest.NewRecorder()
	router.ServeHTTP(unknownRec, unknownReq)
	if unknownRec.Code != http.StatusNotFound {
		t.Errorf("signup for unknown player returned status %d, want 404, body: %s", unknownRec.Code, unknownRec.Body.String())
	}

	// Already-claimed player -> 400.
	playerID, err := playerService.CreatePlayer("Zzz Integration Handler Heidi")
	if err != nil {
		t.Fatalf("failed to create player: %v", err)
	}
	firstBody, _ := json.Marshal(map[string]string{
		"player_id": playerID.String(),
		"email":     "heidi@example.com",
		"password":  "s3cret-pass",
	})
	firstReq := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader(firstBody))
	firstReq.Header.Set("Content-Type", "application/json")
	firstRec := httptest.NewRecorder()
	router.ServeHTTP(firstRec, firstReq)
	if firstRec.Code != http.StatusOK {
		t.Fatalf("first signup returned status %d, body: %s", firstRec.Code, firstRec.Body.String())
	}

	secondBody, _ := json.Marshal(map[string]string{
		"player_id": playerID.String(),
		"email":     "heidi-other@example.com",
		"password":  "another-pass",
	})
	secondReq := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader(secondBody))
	secondReq.Header.Set("Content-Type", "application/json")
	secondRec := httptest.NewRecorder()
	router.ServeHTTP(secondRec, secondReq)
	if secondRec.Code != http.StatusBadRequest {
		t.Errorf("signup on already-claimed player returned status %d, want 400, body: %s", secondRec.Code, secondRec.Body.String())
	}
}

func TestAuthHandler_Integration_LoginWrongPasswordReturns401(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	playerService := services.NewPlayerService(tx)
	authService := services.NewAuthService(tx, testAuthJWTSecret)
	router := newAuthTestRouter(authService)

	playerID, err := playerService.CreatePlayer("Zzz Integration Handler Ivan")
	if err != nil {
		t.Fatalf("failed to create player: %v", err)
	}
	signupBody, _ := json.Marshal(map[string]string{
		"player_id": playerID.String(),
		"email":     "ivan@example.com",
		"password":  "correct-pass",
	})
	signupReq := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader(signupBody))
	signupReq.Header.Set("Content-Type", "application/json")
	signupRec := httptest.NewRecorder()
	router.ServeHTTP(signupRec, signupReq)
	if signupRec.Code != http.StatusOK {
		t.Fatalf("signup returned status %d, body: %s", signupRec.Code, signupRec.Body.String())
	}

	loginBody, _ := json.Marshal(map[string]string{
		"email":    "ivan@example.com",
		"password": "wrong-pass",
	})
	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusUnauthorized {
		t.Errorf("login with wrong password returned status %d, want 401, body: %s", loginRec.Code, loginRec.Body.String())
	}
}
