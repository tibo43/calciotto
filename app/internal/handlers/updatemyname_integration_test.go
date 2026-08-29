package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"app/internal/handlers"
	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/gin-gonic/gin"
)

const testUpdateMyNameJWTSecret = "zzz-integration-test-update-my-name-secret"

func newUpdateMyNameTestRouter(playerService *services.PlayerService, authService *services.AuthService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	playerHandler := handlers.NewPlayerHandler(playerService, nil, nil)
	router.PATCH("/players/me", handlers.AuthMiddleware(authService), playerHandler.UpdateMyName)
	return router
}

// authenticatedRenamePlayer creates a player and signs them up with a
// password so a real JWT can be issued for PATCH /players/me — the same
// create-then-Signup-then-Login shape newInviteEnv.authenticatedPlayer uses.
func authenticatedRenamePlayer(t *testing.T, playerService *services.PlayerService, authService *services.AuthService, name, email string) (string, string) {
	t.Helper()
	id, err := playerService.CreatePlayer(name)
	if err != nil {
		t.Fatalf("failed to create player %q: %v", name, err)
	}
	if err := authService.Signup(id, email, "s3cret-pass"); err != nil {
		t.Fatalf("failed to sign up %q: %v", name, err)
	}
	token, err := authService.Login(email, "s3cret-pass")
	if err != nil {
		t.Fatalf("failed to log in %q: %v", name, err)
	}
	return id.String(), token
}

func patchJSON(t *testing.T, router *gin.Engine, path, token string, body map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to encode request body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPatch, path, bytes.NewReader(encoded))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// TestUpdateMyName_Integration_Success covers the end-to-end happy path:
// a real JWT, a real request, and the rename actually persisted.
func TestUpdateMyName_Integration_Success(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	playerService := services.NewPlayerService(tx)
	authService := services.NewAuthService(tx, testUpdateMyNameJWTSecret)
	router := newUpdateMyNameTestRouter(playerService, authService)

	playerIDStr, token := authenticatedRenamePlayer(t, playerService, authService, "Zzz Integration HTTP Rename Before", "rename-http-before@example.com")

	rec := patchJSON(t, router, "/players/me", token, map[string]string{"name": "Zzz Integration HTTP Rename After"})
	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH /players/me returned status %d, want 200, body: %s", rec.Code, rec.Body.String())
	}

	var stored models.Player
	if err := tx.First(&stored, "id = ?", playerIDStr).Error; err != nil {
		t.Fatalf("failed to reload player: %v", err)
	}
	if stored.Name != "Zzz Integration HTTP Rename After" {
		t.Errorf("stored name = %q, want %q", stored.Name, "Zzz Integration HTTP Rename After")
	}
}

// TestUpdateMyName_Integration_DuplicateNameReturns400 covers the product
// requirement end to end: renaming into a name another player already holds
// anywhere in the system is rejected with 400, and the caller's own name is
// left unchanged.
func TestUpdateMyName_Integration_DuplicateNameReturns400(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	playerService := services.NewPlayerService(tx)
	authService := services.NewAuthService(tx, testUpdateMyNameJWTSecret)
	router := newUpdateMyNameTestRouter(playerService, authService)

	if _, err := playerService.CreatePlayer("Zzz Integration HTTP Rename Taken"); err != nil {
		t.Fatalf("failed to create the name-holding player: %v", err)
	}
	playerIDStr, token := authenticatedRenamePlayer(t, playerService, authService, "Zzz Integration HTTP Rename Requester", "rename-http-requester@example.com")

	rec := patchJSON(t, router, "/players/me", token, map[string]string{"name": "zzz integration http rename taken"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("PATCH /players/me returned status %d, want 400, body: %s", rec.Code, rec.Body.String())
	}

	var stored models.Player
	if err := tx.First(&stored, "id = ?", playerIDStr).Error; err != nil {
		t.Fatalf("failed to reload player: %v", err)
	}
	if stored.Name != "Zzz Integration HTTP Rename Requester" {
		t.Errorf("stored name = %q after rejected rename, want unchanged %q", stored.Name, "Zzz Integration HTTP Rename Requester")
	}
}

// TestUpdateMyName_Integration_NoToken covers the unauthenticated case:
// AuthMiddleware rejects before the handler runs.
func TestUpdateMyName_Integration_NoToken(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	playerService := services.NewPlayerService(tx)
	authService := services.NewAuthService(tx, testUpdateMyNameJWTSecret)
	router := newUpdateMyNameTestRouter(playerService, authService)

	rec := patchJSON(t, router, "/players/me", "", map[string]string{"name": "Zzz Integration HTTP Rename NoToken"})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("no-token rename returned status %d, want 401, body: %s", rec.Code, rec.Body.String())
	}
}
