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
	"gorm.io/gorm"
)

const testAuthJWTSecret = "zzz-integration-test-secret"

// newSignupInviteCode creates a throwaway group (bypassing the disabled
// POST /groups route — see GroupHandler.CreateGroup) purely to get a valid
// invite code: signup now requires one on every call, since self-service
// group creation/joining is disabled and signup is the only way left in.
func newSignupInviteCode(t *testing.T, tx *gorm.DB) string {
	t.Helper()
	group, err := services.NewGroupService(tx).CreateGroup("Zzz Auth Handler Group", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create a group for its invite code: %v", err)
	}
	return group.InviteCode
}

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

	authService := services.NewAuthService(tx, testAuthJWTSecret)
	router := newAuthTestRouter(authService)
	inviteCode := newSignupInviteCode(t, tx)

	signupBody, _ := json.Marshal(map[string]string{
		"name":        "Zzz Integration Handler Grace",
		"email":       "grace@example.com",
		"password":    "s3cret-pass",
		"invite_code": inviteCode,
	})
	signupReq := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader(signupBody))
	signupReq.Header.Set("Content-Type", "application/json")
	signupRec := httptest.NewRecorder()
	router.ServeHTTP(signupRec, signupReq)
	if signupRec.Code != http.StatusOK {
		t.Fatalf("signup returned status %d, body: %s", signupRec.Code, signupRec.Body.String())
	}
	var signupResp struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(signupRec.Body.Bytes(), &signupResp); err != nil {
		t.Fatalf("failed to decode signup response: %v", err)
	}
	if signupResp.ID == "" {
		t.Fatal("signup response has an empty id")
	}
	playerID := signupResp.ID

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
	if protectedResp.PlayerID != playerID {
		t.Errorf("protected route injected player_id = %q, want %q", protectedResp.PlayerID, playerID)
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

	authService := services.NewAuthService(tx, testAuthJWTSecret)
	router := newAuthTestRouter(authService)
	inviteCode := newSignupInviteCode(t, tx)

	// Empty name -> 400.
	emptyNameBody, _ := json.Marshal(map[string]string{
		"name":        "  ",
		"email":       "ghost@example.com",
		"password":    "s3cret-pass",
		"invite_code": inviteCode,
	})
	emptyNameReq := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader(emptyNameBody))
	emptyNameReq.Header.Set("Content-Type", "application/json")
	emptyNameRec := httptest.NewRecorder()
	router.ServeHTTP(emptyNameRec, emptyNameReq)
	if emptyNameRec.Code != http.StatusBadRequest {
		t.Errorf("signup with an empty name returned status %d, want 400, body: %s", emptyNameRec.Code, emptyNameRec.Body.String())
	}

	// Empty email -> 400.
	emptyEmailBody, _ := json.Marshal(map[string]string{
		"name":        "Zzz Integration Handler Heidi",
		"email":       "",
		"password":    "s3cret-pass",
		"invite_code": inviteCode,
	})
	emptyEmailReq := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader(emptyEmailBody))
	emptyEmailReq.Header.Set("Content-Type", "application/json")
	emptyEmailRec := httptest.NewRecorder()
	router.ServeHTTP(emptyEmailRec, emptyEmailReq)
	if emptyEmailRec.Code != http.StatusBadRequest {
		t.Errorf("signup with an empty email returned status %d, want 400, body: %s", emptyEmailRec.Code, emptyEmailRec.Body.String())
	}

	// Empty password -> 400.
	emptyPasswordBody, _ := json.Marshal(map[string]string{
		"name":        "Zzz Integration Handler Heidi",
		"email":       "heidi@example.com",
		"password":    "",
		"invite_code": inviteCode,
	})
	emptyPasswordReq := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader(emptyPasswordBody))
	emptyPasswordReq.Header.Set("Content-Type", "application/json")
	emptyPasswordRec := httptest.NewRecorder()
	router.ServeHTTP(emptyPasswordRec, emptyPasswordReq)
	if emptyPasswordRec.Code != http.StatusBadRequest {
		t.Errorf("signup with an empty password returned status %d, want 400, body: %s", emptyPasswordRec.Code, emptyPasswordRec.Body.String())
	}

	// Empty invite code -> 400: self-service group creation/joining is
	// disabled, so signup is the only way into a group and can no longer be
	// skipped.
	emptyInviteCodeBody, _ := json.Marshal(map[string]string{
		"name":     "Zzz Integration Handler Heidi",
		"email":    "heidi-no-code@example.com",
		"password": "s3cret-pass",
	})
	emptyInviteCodeReq := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader(emptyInviteCodeBody))
	emptyInviteCodeReq.Header.Set("Content-Type", "application/json")
	emptyInviteCodeRec := httptest.NewRecorder()
	router.ServeHTTP(emptyInviteCodeRec, emptyInviteCodeReq)
	if emptyInviteCodeRec.Code != http.StatusBadRequest {
		t.Errorf("signup with no invite code returned status %d, want 400, body: %s", emptyInviteCodeRec.Code, emptyInviteCodeRec.Body.String())
	}

	// A first, valid signup succeeds...
	firstBody, _ := json.Marshal(map[string]string{
		"name":        "Zzz Integration Handler Heidi",
		"email":       "heidi@example.com",
		"password":    "s3cret-pass",
		"invite_code": inviteCode,
	})
	firstReq := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader(firstBody))
	firstReq.Header.Set("Content-Type", "application/json")
	firstRec := httptest.NewRecorder()
	router.ServeHTTP(firstRec, firstReq)
	if firstRec.Code != http.StatusOK {
		t.Fatalf("first signup returned status %d, body: %s", firstRec.Code, firstRec.Body.String())
	}

	// ...but re-using the same email fails, even with a different name.
	duplicateEmailBody, _ := json.Marshal(map[string]string{
		"name":        "Zzz Integration Handler Heidi Two",
		"email":       "heidi@example.com",
		"password":    "another-pass",
		"invite_code": inviteCode,
	})
	duplicateEmailReq := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader(duplicateEmailBody))
	duplicateEmailReq.Header.Set("Content-Type", "application/json")
	duplicateEmailRec := httptest.NewRecorder()
	router.ServeHTTP(duplicateEmailRec, duplicateEmailReq)
	if duplicateEmailRec.Code != http.StatusBadRequest {
		t.Errorf("signup with an already-used email returned status %d, want 400, body: %s", duplicateEmailRec.Code, duplicateEmailRec.Body.String())
	}

	// Re-using the same name (with a different email) is fine — names are not
	// unique, unlike the old admin-created ghost-player flow (removed).
	duplicateNameBody, _ := json.Marshal(map[string]string{
		"name":        "Zzz Integration Handler Heidi",
		"email":       "heidi-second-account@example.com",
		"password":    "another-pass",
		"invite_code": inviteCode,
	})
	duplicateNameReq := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader(duplicateNameBody))
	duplicateNameReq.Header.Set("Content-Type", "application/json")
	duplicateNameRec := httptest.NewRecorder()
	router.ServeHTTP(duplicateNameRec, duplicateNameReq)
	if duplicateNameRec.Code != http.StatusOK {
		t.Errorf("signup re-using an existing display name returned status %d, want 200, body: %s", duplicateNameRec.Code, duplicateNameRec.Body.String())
	}
}

func TestAuthHandler_Integration_LoginWrongPasswordReturns401(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testAuthJWTSecret)
	router := newAuthTestRouter(authService)
	inviteCode := newSignupInviteCode(t, tx)

	signupBody, _ := json.Marshal(map[string]string{
		"name":        "Zzz Integration Handler Ivan",
		"email":       "ivan@example.com",
		"password":    "correct-pass",
		"invite_code": inviteCode,
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
