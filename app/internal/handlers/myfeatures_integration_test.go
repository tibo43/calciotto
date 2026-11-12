package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"app/internal/handlers"
	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const testMyFeaturesJWTSecret = "zzz-integration-test-my-features-secret"

func getMyFeatures(t *testing.T, router *gin.Engine, token string) (int, []string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/players/me/features", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		return rec.Code, nil
	}
	var body struct {
		Features []string `json:"features"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response %q: %v", rec.Body.String(), err)
	}
	return rec.Code, body.Features
}

// TestGetMyFeatures_Integration covers GET /players/me/features: an empty
// list (not null) by default, the caller's own features once rows exist, and
// never another player's.
func TestGetMyFeatures_Integration(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	playerService := services.NewPlayerService(tx)
	authService := services.NewAuthService(tx, testMyFeaturesJWTSecret)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/players/me/features", handlers.AuthMiddleware(authService), handlers.NewPlayerHandler(playerService).GetMyFeatures)

	_, token := authenticatedRenamePlayer(t, playerService, authService, "Zzz Integration Features Owner", "features-owner@example.com")
	otherIDStr, _ := authenticatedRenamePlayer(t, playerService, authService, "Zzz Integration Features Other", "features-other@example.com")

	if code, features := getMyFeatures(t, router, token); code != http.StatusOK || features == nil || len(features) != 0 {
		t.Fatalf("with no feature: status %d, features %#v, want 200 and an empty list", code, features)
	}

	ownerID, err := authService.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken returned error: %v", err)
	}
	otherID := uuid.MustParse(otherIDStr)
	for _, row := range []models.PlayerFeature{
		{PlayerID: ownerID, Feature: models.FeaturePWAInstall},
		{PlayerID: ownerID, Feature: "another-feature"},
		{PlayerID: otherID, Feature: "other-players-feature"},
	} {
		if err := tx.Create(&row).Error; err != nil {
			t.Fatalf("failed to enable feature %q: %v", row.Feature, err)
		}
	}

	code, features := getMyFeatures(t, router, token)
	if want := []string{"another-feature", models.FeaturePWAInstall}; code != http.StatusOK || !reflect.DeepEqual(features, want) {
		t.Errorf("with features: status %d, features %#v, want 200 and %#v", code, features, want)
	}

	if code, _ := getMyFeatures(t, router, ""); code != http.StatusUnauthorized {
		t.Errorf("without a token: status %d, want 401", code)
	}
}

// TestPlayerFeature_Integration_UnknownPlayerRejected pins the foreign key
// that makes a hand-written INSERT with a mistyped player id fail instead of
// silently enabling the feature for nobody.
func TestPlayerFeature_Integration_UnknownPlayerRejected(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	err := tx.Exec(`INSERT INTO player_features (player_id, feature) VALUES (?, ?)`, uuid.New(), models.FeaturePWAInstall).Error
	if err == nil {
		t.Error("enabling a feature for a player that does not exist succeeded, want a foreign-key error")
	}
}
