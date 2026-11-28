package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"app/internal/handlers"
	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const testLocaleJWTSecret = "zzz-integration-test-locale-secret"

// newLocaleEnv mounts the two locale routes with main.go's real auth
// middleware.
func newLocaleEnv(t *testing.T, tx *gorm.DB) *registrationEnv {
	t.Helper()
	env := newRegistrationEnv(t, tx)
	env.auth = services.NewAuthService(tx, testLocaleJWTSecret)
	authRequired := handlers.AuthMiddleware(env.auth)
	playerHandler := handlers.NewPlayerHandler(env.players)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/players/me/locale", authRequired, playerHandler.GetMyLocale)
	router.PUT("/players/me/locale", authRequired, playerHandler.UpdateMyLocale)
	env.router = router
	return env
}

// decodeLocale returns the "locale" of a response, and whether the key was
// there at all (null must be an explicit null, not a missing key).
func decodeLocale(t *testing.T, rec *httptest.ResponseRecorder) (*string, bool) {
	t.Helper()
	var body map[string]*string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode %s: %v", rec.Body.String(), err)
	}
	value, present := body["locale"]
	return value, present
}

// A new player follows the phone (null), a saved language comes back as it was
// saved, and null goes back to following the phone.
func TestMyLocale_Integration_RoundTrip(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newLocaleEnv(t, tx)
	playerID, token := env.authenticatedPlayer(t, "Zzz Locale Roundtrip", "locale-roundtrip@example.com")

	rec := env.do(http.MethodGet, "/players/me/locale", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET returned %d: %s", rec.Code, rec.Body.String())
	}
	if got, present := decodeLocale(t, rec); !present || got != nil {
		t.Errorf("new player locale = %v (present %v), want an explicit null", got, present)
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", rec.Header().Get("Cache-Control"))
	}

	rec = env.do(http.MethodPut, "/players/me/locale", token, map[string]any{"locale": " DE "})
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT returned %d: %s", rec.Code, rec.Body.String())
	}
	if got, _ := decodeLocale(t, rec); got == nil || *got != "de" {
		t.Errorf("PUT answered %v, want de", got)
	}
	var stored models.Player
	if err := tx.First(&stored, "id = ?", playerID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Locale == nil || *stored.Locale != "de" {
		t.Errorf("stored %v, want de", stored.Locale)
	}
	if got, _ := decodeLocale(t, env.do(http.MethodGet, "/players/me/locale", token, nil)); got == nil || *got != "de" {
		t.Errorf("GET after PUT = %v, want de", got)
	}

	for _, body := range []map[string]any{{"locale": nil}, {"locale": ""}} {
		if rec = env.do(http.MethodPut, "/players/me/locale", token, map[string]any{"locale": "it"}); rec.Code != http.StatusOK {
			t.Fatalf("PUT it returned %d", rec.Code)
		}
		rec = env.do(http.MethodPut, "/players/me/locale", token, body)
		if rec.Code != http.StatusOK {
			t.Fatalf("PUT %v returned %d: %s", body, rec.Code, rec.Body.String())
		}
		if got, present := decodeLocale(t, rec); !present || got != nil {
			t.Errorf("PUT %v answered %v (present %v), want an explicit null", body, got, present)
		}
		if err := tx.First(&stored, "id = ?", playerID).Error; err != nil {
			t.Fatal(err)
		}
		if stored.Locale != nil {
			t.Errorf("stored %q after %v, want NULL", *stored.Locale, body)
		}
	}
}

// Unknown languages, a missing key and a wrong type are 400s that leave the
// saved language untouched; no token is a 401.
func TestMyLocale_Integration_Validation(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newLocaleEnv(t, tx)
	_, token := env.authenticatedPlayer(t, "Zzz Locale Validation", "locale-validation@example.com")

	if rec := env.do(http.MethodPut, "/players/me/locale", token, map[string]any{"locale": "fr"}); rec.Code != http.StatusOK {
		t.Fatalf("seeding PUT returned %d: %s", rec.Code, rec.Body.String())
	}
	for _, body := range []any{
		map[string]any{"locale": "pt"},
		map[string]any{"locale": "fr-CA"},
		map[string]any{"locale": "français"},
		map[string]any{},
		map[string]any{"language": "de"},
		map[string]any{"locale": 7},
		map[string]any{"locale": []string{"de"}},
		nil,
	} {
		rec := env.do(http.MethodPut, "/players/me/locale", token, body)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("PUT %v returned %d, want 400: %s", body, rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"error"`) {
			t.Errorf("PUT %v answered %s, want an error reason", body, rec.Body.String())
		}
	}
	if got, _ := decodeLocale(t, env.do(http.MethodGet, "/players/me/locale", token, nil)); got == nil || *got != "fr" {
		t.Errorf("locale after rejected PUTs = %v, want unchanged fr", got)
	}
	if rec := env.do(http.MethodPut, "/players/me/locale", "", map[string]any{"locale": "de"}); rec.Code != http.StatusUnauthorized {
		t.Errorf("no-token PUT returned %d, want 401", rec.Code)
	}
	if rec := env.do(http.MethodGet, "/players/me/locale", "", nil); rec.Code != http.StatusUnauthorized {
		t.Errorf("no-token GET returned %d, want 401", rec.Code)
	}
}

// The language is the player's own: it is never part of the Player JSON, so
// no other route can leak it.
func TestPlayerLocale_NeverInPlayerJSON(t *testing.T) {
	code := "fr"
	raw, err := json.Marshal(models.Player{Name: "Zzz", Locale: &code})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "fr") || strings.Contains(strings.ToLower(string(raw)), "locale") {
		t.Errorf("Player JSON %s carries the locale", raw)
	}
}
