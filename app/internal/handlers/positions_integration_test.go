package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"app/internal/handlers"
	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const testPositionsJWTSecret = "zzz-integration-test-positions-secret"

// newPositionsEnv reuses registrationEnv's group/match/player helpers and adds
// the three positions routes with main.go's real middleware chains, plus the
// two match payloads every member reads, to prove they never carry positions.
func newPositionsEnv(t *testing.T, tx *gorm.DB) *registrationEnv {
	t.Helper()
	env := newRegistrationEnv(t, tx)
	env.auth = services.NewAuthService(tx, testPositionsJWTSecret)

	membershipService := env.memberships
	authRequired := handlers.AuthMiddleware(env.auth)
	requireGroupMember := handlers.RequireGroupMembership(membershipService)
	requireGroupMemberByMatchID := handlers.RequireGroupMembershipByMatchPathParam(env.matches, membershipService, "id")
	requireGroupAdminByMatchID := handlers.RequireGroupAdminByMatchPathParam(env.matches, membershipService, "id")
	playerHandler := handlers.NewPlayerHandler(env.players)
	matchHandler := handlers.NewMatchHandler(env.matches, membershipService)
	registrationHandler := handlers.NewMatchRegistrationHandler(env.registrations)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/players/me/positions", authRequired, playerHandler.GetMyPositions)
	router.PUT("/players/me/positions", authRequired, playerHandler.UpdateMyPositions)
	router.GET("/matches/:id/positions", authRequired, requireGroupAdminByMatchID, playerHandler.GetMatchPositions)
	router.POST("/matches/:id/registrations", authRequired, requireGroupMemberByMatchID, registrationHandler.Register)
	router.GET("/matches/:id/registrations", authRequired, requireGroupMemberByMatchID, registrationHandler.ListRegistrations)
	router.GET("/matches/:id/details", authRequired, requireGroupMember, matchHandler.GetMatchDetailsByID)
	env.router = router
	return env
}

func decodePositions(t *testing.T, rec *httptest.ResponseRecorder) []string {
	t.Helper()
	var body struct {
		PreferredPositions []string `json:"PreferredPositions"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode %s: %v", rec.Body.String(), err)
	}
	return body.PreferredPositions
}

// A new player has no position, the player's own picks are stored
// deduplicated and in canonical order, and an empty list clears them.
func TestMyPositions_Integration_RoundTrip(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newPositionsEnv(t, tx)
	playerID, token := env.authenticatedPlayer(t, "Zzz Pos Roundtrip", "pos-roundtrip@example.com")

	rec := env.do(http.MethodGet, "/players/me/positions", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET returned %d: %s", rec.Code, rec.Body.String())
	}
	if got := decodePositions(t, rec); got == nil || len(got) != 0 {
		t.Errorf("new player positions = %#v, want an empty list (not null)", got)
	}
	if !strings.Contains(rec.Body.String(), `"PreferredPositions":[]`) {
		t.Errorf("body %s, want PreferredPositions as []", rec.Body.String())
	}

	rec = env.do(http.MethodPut, "/players/me/positions", token, map[string]any{"PreferredPositions": []string{"MID", "def", "MID"}})
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT returned %d: %s", rec.Code, rec.Body.String())
	}
	if got := decodePositions(t, rec); !reflect.DeepEqual(got, []string{"DEF", "MID"}) {
		t.Errorf("PUT answered %#v, want [DEF MID]", got)
	}
	var stored models.Player
	if err := tx.First(&stored, "id = ?", playerID).Error; err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual([]string(stored.PreferredPositions), []string{"DEF", "MID"}) {
		t.Errorf("stored %#v, want [DEF MID]", stored.PreferredPositions)
	}
	if got := decodePositions(t, env.do(http.MethodGet, "/players/me/positions", token, nil)); !reflect.DeepEqual(got, []string{"DEF", "MID"}) {
		t.Errorf("GET after PUT = %#v, want [DEF MID]", got)
	}

	rec = env.do(http.MethodPut, "/players/me/positions", token, map[string]any{"PreferredPositions": []string{}})
	if rec.Code != http.StatusOK {
		t.Fatalf("clearing PUT returned %d: %s", rec.Code, rec.Body.String())
	}
	if got := decodePositions(t, env.do(http.MethodGet, "/players/me/positions", token, nil)); len(got) != 0 {
		t.Errorf("positions after clearing = %#v, want none", got)
	}
}

// Unknown values, a missing key and a wrong type are 400s that leave the
// stored positions untouched; no token is a 401.
func TestMyPositions_Integration_Validation(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newPositionsEnv(t, tx)
	_, token := env.authenticatedPlayer(t, "Zzz Pos Validation", "pos-validation@example.com")

	if rec := env.do(http.MethodPut, "/players/me/positions", token, map[string]any{"PreferredPositions": []string{"GK"}}); rec.Code != http.StatusOK {
		t.Fatalf("seeding PUT returned %d: %s", rec.Code, rec.Body.String())
	}
	for _, body := range []any{
		map[string]any{"PreferredPositions": []string{"GK", "FORW"}},
		map[string]any{"PreferredPositions": []string{"ST"}},
		map[string]any{},
		map[string]any{"preferred_positions": []string{"DEF"}},
		map[string]any{"PreferredPositions": "GK"},
		map[string]any{"PreferredPositions": nil},
	} {
		rec := env.do(http.MethodPut, "/players/me/positions", token, body)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("PUT %v returned %d, want 400: %s", body, rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"error"`) {
			t.Errorf("PUT %v answered %s, want an error reason", body, rec.Body.String())
		}
	}
	if got := decodePositions(t, env.do(http.MethodGet, "/players/me/positions", token, nil)); !reflect.DeepEqual(got, []string{"GK"}) {
		t.Errorf("positions after rejected PUTs = %#v, want unchanged [GK]", got)
	}
	if rec := env.do(http.MethodPut, "/players/me/positions", "", map[string]any{"PreferredPositions": []string{}}); rec.Code != http.StatusUnauthorized {
		t.Errorf("no-token PUT returned %d, want 401", rec.Code)
	}
	if rec := env.do(http.MethodGet, "/players/me/positions", "", nil); rec.Code != http.StatusUnauthorized {
		t.Errorf("no-token GET returned %d, want 401", rec.Code)
	}
}

// GET /matches/:id/positions is the only way to read someone else's
// positions: an admin of the match's group gets them, a plain member 403, a
// player outside the group 404. The sign-up list and the match details, read
// by every member, admin included, never carry them.
func TestMatchPositions_Integration_AdminOnly(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newPositionsEnv(t, tx)
	group := env.newGroup(t, "positions")
	matchID := env.scheduledMatch(t, group.id, 16)
	outsider := env.newGroup(t, "positions-outsider")

	if _, err := env.players.UpdatePreferredPositions(group.memberID, []string{"MID", "GK"}); err != nil {
		t.Fatal(err)
	}
	if _, err := env.players.UpdatePreferredPositions(outsider.memberID, []string{"FWD"}); err != nil {
		t.Fatal(err)
	}

	path := "/matches/" + matchID.String() + "/positions"
	rec := env.do(http.MethodGet, path, group.adminToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin GET positions returned %d: %s", rec.Code, rec.Body.String())
	}
	var entries []models.PlayerPositionsEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &entries); err != nil {
		t.Fatal(err)
	}
	// Only group members with at least one position: the admin has none and
	// the other group's member is not listed.
	if len(entries) != 1 || entries[0].PlayerID != group.memberID ||
		!reflect.DeepEqual([]string(entries[0].PreferredPositions), []string{"GK", "MID"}) {
		t.Errorf("admin GET positions = %+v, want only the member with [GK MID]", entries)
	}

	if rec := env.do(http.MethodGet, path, group.memberToken, nil); rec.Code != http.StatusForbidden {
		t.Errorf("member GET positions returned %d, want 403: %s", rec.Code, rec.Body.String())
	} else if strings.Contains(rec.Body.String(), "MID") {
		t.Errorf("member 403 body leaks positions: %s", rec.Body.String())
	}
	if rec := env.do(http.MethodGet, path, outsider.adminToken, nil); rec.Code != http.StatusNotFound {
		t.Errorf("outsider admin GET positions returned %d, want 404", rec.Code)
	}

	// The shared payloads stay position-free, whoever asks.
	if rec := env.do(http.MethodPost, "/matches/"+matchID.String()+"/registrations", group.memberToken, nil); rec.Code != http.StatusOK {
		t.Fatalf("member sign-up returned %d: %s", rec.Code, rec.Body.String())
	}
	for _, token := range []string{group.adminToken, group.memberToken} {
		for _, p := range []string{
			"/matches/" + matchID.String() + "/registrations",
			"/matches/" + matchID.String() + "/details?group_id=" + group.id.String(),
		} {
			rec := env.do(http.MethodGet, p, token, nil)
			if rec.Code != http.StatusOK {
				t.Fatalf("GET %s returned %d: %s", p, rec.Code, rec.Body.String())
			}
			if strings.Contains(rec.Body.String(), "osition") && strings.Contains(rec.Body.String(), "Preferred") {
				t.Errorf("GET %s leaks positions: %s", p, rec.Body.String())
			}
			if strings.Contains(rec.Body.String(), `"GK"`) || strings.Contains(rec.Body.String(), `"MID"`) {
				t.Errorf("GET %s leaks position values: %s", p, rec.Body.String())
			}
		}
	}
}
