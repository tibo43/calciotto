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

const testNotificationsJWTSecret = "zzz-integration-test-notifications-secret"

func sendJSON(router *gin.Engine, method, path, authorization string, body any) *httptest.ResponseRecorder {
	encoded, _ := json.Marshal(body)
	req := httptest.NewRequest(method, path, bytes.NewReader(encoded))
	req.Header.Set("Content-Type", "application/json")
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// TestNotificationRoutes_Integration covers the HTTP side: the subscription
// routes' feature gate and the tick route's shared-secret check.
func TestNotificationRoutes_Integration(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	playerService := services.NewPlayerService(tx)
	authService := services.NewAuthService(tx, testNotificationsJWTSecret)
	notificationService := services.NewNotificationService(tx, nil, services.NewMatchService(tx))
	handler := handlers.NewNotificationHandler(notificationService, "public-key", "tick-secret")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	authRequired := handlers.AuthMiddleware(authService)
	router.GET("/push/public-key", authRequired, handler.GetPublicKey)
	router.POST("/players/me/push-subscription", authRequired, handler.Subscribe)
	router.DELETE("/players/me/push-subscription", authRequired, handler.Unsubscribe)
	router.POST("/internal/notifications/tick", handler.Tick)

	_, token := authenticatedRenamePlayer(t, playerService, authService, "Zzz Integration Push Owner", "push-owner@example.com")
	bearer := "Bearer " + token
	playerID, err := authService.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	device := map[string]any{"endpoint": "https://push.test/device", "keys": map[string]string{"p256dh": "k", "auth": "a"}}

	if rec := sendJSON(router, http.MethodGet, "/push/public-key", bearer, nil); rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte("public-key")) {
		t.Errorf("GET /push/public-key = %d %s", rec.Code, rec.Body.String())
	}
	if rec := sendJSON(router, http.MethodPost, "/players/me/push-subscription", bearer, device); rec.Code != http.StatusForbidden {
		t.Errorf("subscribe without the feature = %d, want 403", rec.Code)
	}

	if err := tx.Create(&models.PlayerFeature{PlayerID: playerID, Feature: models.FeaturePushNotifications}).Error; err != nil {
		t.Fatalf("enable feature: %v", err)
	}
	if rec := sendJSON(router, http.MethodPost, "/players/me/push-subscription", bearer, map[string]any{"endpoint": "x"}); rec.Code != http.StatusBadRequest {
		t.Errorf("subscribe without keys = %d, want 400", rec.Code)
	}
	if rec := sendJSON(router, http.MethodPost, "/players/me/push-subscription", bearer, device); rec.Code != http.StatusOK {
		t.Errorf("subscribe with the feature = %d %s, want 200", rec.Code, rec.Body.String())
	}
	if rec := sendJSON(router, http.MethodDelete, "/players/me/push-subscription", bearer, map[string]string{"endpoint": "https://push.test/device"}); rec.Code != http.StatusOK {
		t.Errorf("unsubscribe = %d, want 200", rec.Code)
	}

	for _, authorization := range []string{"", "Bearer wrong", "tick-secret", bearer} {
		if rec := sendJSON(router, http.MethodPost, "/internal/notifications/tick", authorization, nil); rec.Code != http.StatusUnauthorized {
			t.Errorf("tick with Authorization %q = %d, want 401", authorization, rec.Code)
		}
	}
	if rec := sendJSON(router, http.MethodPost, "/internal/notifications/tick", "Bearer tick-secret", nil); rec.Code != http.StatusOK {
		t.Errorf("tick with the secret = %d %s, want 200", rec.Code, rec.Body.String())
	}
}
