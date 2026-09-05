package handlers_test

import (
	"net/http"
	"strings"
	"testing"

	"app/internal/testutil"
)

// TestGetMatchesDetails_Integration_EmptyGroupReturnsEmptyArray pins
// MatchHandler.GetMatchesDetails to return a JSON array, never the literal
// `null`, for a group with zero matches. MatchService.GetMatchesDetails
// declares `var matches []models.MatchWithDetails` and only ever appends to
// it, so an empty result stays a nil slice; encoding/json serializes a nil
// slice as `null`, not `[]`. The frontend's MatchesAll.vue does a strict
// Array.isArray(matches) check and throws on anything else, so a brand-new
// group with no matches yet would crash the matches list instead of showing
// its empty state.
//
// Asserting on the raw response body (not just unmarshaling it into a Go
// slice) is what actually catches the regression: json.Unmarshal("null",
// &[]T{}) also produces a zero-length Go slice, so a test that only checked
// len(result) == 0 would pass against the buggy handler too.
func TestGetMatchesDetails_Integration_EmptyGroupReturnsEmptyArray(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newBootstrapEnv(t, tx)

	playerID, token := env.newAuthenticatedPlayer(t,
		"Zzz Matches Details Empty Player", "matches-details-empty@example.com")

	groupID := env.createGroupDirect(t, "Zzz Matches Details Empty Group", playerID).ID

	rec := env.do(http.MethodGet, "/matches/details?group_id="+groupID.String(), token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /matches/details returned status %d, want 200, body: %s", rec.Code, rec.Body.String())
	}

	body := strings.TrimSpace(rec.Body.String())
	if body == "null" {
		t.Fatalf("GET /matches/details for a group with no matches returned the JSON literal null instead of an empty array")
	}
	if body != "[]" {
		t.Fatalf("GET /matches/details for a group with no matches returned body %q, want \"[]\"", body)
	}
}
