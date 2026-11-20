package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"
)

// TestEmptyGroup_Integration_StaysAddressableAndDoesNotLeakOtherGroup
// reproduces the "I deleted every match in Test and the app showed me the
// other group" report at the API layer: a group with zero matches must still
// be listed, its season and match lists must be empty arrays (not null, not
// another group's data), and only a request that omits group_id may fall
// back to the player's first-joined group.
func TestEmptyGroup_Integration_StaysAddressableAndDoesNotLeakOtherGroup(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	env := newBootstrapEnv(t, tx)
	standingsRouter := newStandingsTestRouter(tx, env.auth, env.memberships)

	playerID, token := env.newAuthenticatedPlayer(t,
		"Zzz Empty Group Player", "empty-group-fallback@example.com")

	firstGroup := env.createGroupDirect(t, "Giovedi Calciotto", playerID)
	emptyGroup := env.createGroupDirect(t, "Test", playerID)

	matchService := services.NewMatchService(tx)
	_, err := matchService.CreateMatch(services.MatchSpec{
		Date: models.Date(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)),
	}, firstGroup.ID)
	if err != nil {
		t.Fatalf("failed to create a match in the first-joined group: %v", err)
	}

	// 1. GET /groups/me still lists the empty group — membership is not
	// tied to having matches or a derived season.
	rec := env.do(http.MethodGet, "/groups/me", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /groups/me returned status %d, want 200, body: %s", rec.Code, rec.Body.String())
	}
	var groups []models.GroupWithRole
	if err := json.Unmarshal(rec.Body.Bytes(), &groups); err != nil {
		t.Fatalf("failed to decode GET /groups/me: %v, body: %s", err, rec.Body.String())
	}
	ids := map[string]bool{}
	for _, group := range groups {
		ids[group.ID.String()] = true
	}
	if !ids[firstGroup.ID.String()] || !ids[emptyGroup.ID.String()] {
		t.Fatalf("GET /groups/me = %+v, want both %s and %s", groups, firstGroup.ID, emptyGroup.ID)
	}

	// 2. Scoped reads on the empty group succeed and return empty arrays.
	matchesRec := env.do(http.MethodGet, "/matches/details?group_id="+emptyGroup.ID.String(), token, nil)
	if matchesRec.Code != http.StatusOK {
		t.Fatalf("GET /matches/details?group_id=empty returned status %d, want 200, body: %s", matchesRec.Code, matchesRec.Body.String())
	}
	if body := strings.TrimSpace(matchesRec.Body.String()); body != "[]" {
		t.Fatalf("GET /matches/details?group_id=empty returned %q, want []", body)
	}

	seasonsReq := env.doSeasons(standingsRouter, token, emptyGroup.ID.String())
	if seasonsReq.Code != http.StatusOK {
		t.Fatalf("GET /standings/seasons?group_id=empty returned status %d, want 200, body: %s", seasonsReq.Code, seasonsReq.Body.String())
	}
	if body := strings.TrimSpace(seasonsReq.Body.String()); body != "[]" {
		t.Fatalf("GET /standings/seasons?group_id=empty returned %q, want []", body)
	}

	pointsRec := env.doStandings(standingsRouter, "/standings/points?group_id="+emptyGroup.ID.String(), token)
	if pointsRec.Code != http.StatusOK {
		t.Fatalf("GET /standings/points?group_id=empty returned status %d, want 200, body: %s", pointsRec.Code, pointsRec.Body.String())
	}

	// 3. Omitting group_id falls back to the first-joined group (Giovedi),
	// which is how an unscoped HomeV2 fetch can show the other group's
	// matches while the empty group is the one the user selected.
	unscoped := env.do(http.MethodGet, "/matches/details", token, nil)
	if unscoped.Code != http.StatusOK {
		t.Fatalf("GET /matches/details (no group_id) returned status %d, want 200, body: %s", unscoped.Code, unscoped.Body.String())
	}
	var unscopedMatches []models.MatchWithDetails
	if err := json.Unmarshal(unscoped.Body.Bytes(), &unscopedMatches); err != nil {
		t.Fatalf("failed to decode unscoped matches: %v, body: %s", err, unscoped.Body.String())
	}
	if len(unscopedMatches) != 1 {
		t.Fatalf("GET /matches/details (no group_id) returned %d matches, want the 1 match from the first-joined group; body: %s", len(unscopedMatches), unscoped.Body.String())
	}
	if unscopedMatches[0].GroupID != firstGroup.ID {
		t.Fatalf("unscoped matches belonged to %s, want first-joined group %s", unscopedMatches[0].GroupID, firstGroup.ID)
	}

	unscopedSeasons := env.doSeasons(standingsRouter, token, "")
	if unscopedSeasons.Code != http.StatusOK {
		t.Fatalf("GET /standings/seasons (no group_id) returned status %d, want 200, body: %s", unscopedSeasons.Code, unscopedSeasons.Body.String())
	}
	var seasons []string
	if err := json.Unmarshal(unscopedSeasons.Body.Bytes(), &seasons); err != nil {
		t.Fatalf("failed to decode unscoped seasons: %v, body: %s", err, unscopedSeasons.Body.String())
	}
	if len(seasons) == 0 {
		t.Fatalf("GET /standings/seasons (no group_id) returned %v, want the first-joined group's seasons, not an empty list", seasons)
	}
}

// doSeasons / doStandings hit the standings router, which is wired separately
// from bootstrapEnv (that router only has /groups and /matches/details).
func (e *bootstrapEnv) doSeasons(router http.Handler, token, groupID string) *httptest.ResponseRecorder {
	path := "/standings/seasons"
	if groupID != "" {
		path += "?group_id=" + groupID
	}
	return e.doStandings(router, path, token)
}

func (e *bootstrapEnv) doStandings(router http.Handler, path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}
