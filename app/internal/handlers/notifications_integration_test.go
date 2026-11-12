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

func preferenceBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]bool {
	t.Helper()
	var body struct {
		Preferences map[string]bool `json:"preferences"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode preferences: %v (%s)", err, rec.Body.String())
	}
	return body.Preferences
}

func TestNotificationPreferences_Integration(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	playerService := services.NewPlayerService(tx)
	authService := services.NewAuthService(tx, testNotificationsJWTSecret)
	notificationService := services.NewNotificationService(tx, nil, services.NewMatchService(tx))
	handler := handlers.NewNotificationHandler(notificationService, "public-key", "tick-secret")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	authRequired := handlers.AuthMiddleware(authService)
	router.GET("/players/me/notification-preferences", authRequired, handler.GetPreferences)
	router.PUT("/players/me/notification-preferences", authRequired, handler.UpdatePreferences)

	_, token := authenticatedRenamePlayer(t, playerService, authService, "Zzz Integration Prefs Owner", "prefs-owner@example.com")
	bearer := "Bearer " + token
	playerID, err := authService.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}

	if rec := sendJSON(router, http.MethodGet, "/players/me/notification-preferences", "", nil); rec.Code != http.StatusUnauthorized {
		t.Errorf("GET without a token = %d, want 401", rec.Code)
	}
	if rec := sendJSON(router, http.MethodGet, "/players/me/notification-preferences", bearer, nil); rec.Code != http.StatusForbidden {
		t.Errorf("GET without the feature = %d, want 403", rec.Code)
	}
	if rec := sendJSON(router, http.MethodPut, "/players/me/notification-preferences", bearer, map[string]any{"preferences": map[string]bool{"kickoff_reminder": false}}); rec.Code != http.StatusForbidden {
		t.Errorf("PUT without the feature = %d, want 403", rec.Code)
	}

	if err := tx.Create(&models.PlayerFeature{PlayerID: playerID, Feature: models.FeaturePushNotifications}).Error; err != nil {
		t.Fatalf("enable feature: %v", err)
	}

	rec := sendJSON(router, http.MethodGet, "/players/me/notification-preferences", bearer, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET = %d %s, want 200", rec.Code, rec.Body.String())
	}
	for _, kind := range services.AllPushTypes {
		if !preferenceBody(t, rec)[kind] {
			t.Errorf("default %s = false, want true (no row yet)", kind)
		}
	}

	if rec := sendJSON(router, http.MethodPut, "/players/me/notification-preferences", bearer, map[string]any{}); rec.Code != http.StatusBadRequest {
		t.Errorf("PUT without preferences = %d, want 400", rec.Code)
	}
	if rec := sendJSON(router, http.MethodPut, "/players/me/notification-preferences", bearer, map[string]any{"preferences": map[string]any{"nope": false}}); rec.Code != http.StatusBadRequest || !bytes.Contains(rec.Body.Bytes(), []byte("unknown notification type")) {
		t.Errorf("PUT unknown type = %d %s, want 400", rec.Code, rec.Body.String())
	}
	if rec := sendJSON(router, http.MethodPut, "/players/me/notification-preferences", bearer, map[string]any{"preferences": map[string]any{"kickoff_reminder": "off"}}); rec.Code != http.StatusBadRequest {
		t.Errorf("PUT non-bool = %d, want 400", rec.Code)
	}

	rec = sendJSON(router, http.MethodPut, "/players/me/notification-preferences", bearer, map[string]any{"preferences": map[string]bool{"kickoff_reminder": false}})
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT = %d %s, want 200", rec.Code, rec.Body.String())
	}
	saved := preferenceBody(t, rec)
	if saved["kickoff_reminder"] {
		t.Error("kickoff_reminder stayed on after turning it off")
	}
	for _, kind := range services.AllPushTypes {
		if kind != "kickoff_reminder" && !saved[kind] {
			t.Errorf("%s = false after updating a different type", kind)
		}
	}

	rec = sendJSON(router, http.MethodGet, "/players/me/notification-preferences", bearer, nil)
	if rec.Code != http.StatusOK || preferenceBody(t, rec)["kickoff_reminder"] {
		t.Fatalf("GET after PUT = %d %s, want kickoff_reminder false", rec.Code, rec.Body.String())
	}

	rec = sendJSON(router, http.MethodPut, "/players/me/notification-preferences", bearer, map[string]any{"preferences": map[string]bool{"kickoff_reminder": true}})
	if rec.Code != http.StatusOK || !preferenceBody(t, rec)["kickoff_reminder"] {
		t.Fatalf("PUT back on = %d %s, want kickoff_reminder true", rec.Code, rec.Body.String())
	}
}
