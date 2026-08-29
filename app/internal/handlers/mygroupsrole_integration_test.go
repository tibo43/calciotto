package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"app/internal/models"
	"app/internal/testutil"

	"github.com/google/uuid"
)

// decodeGroupsWithRole reads GET /groups/me's response as the role-tagged
// shape the endpoint now returns, keyed by group id.
func decodeGroupsWithRole(t *testing.T, rec *httptest.ResponseRecorder) map[uuid.UUID]string {
	t.Helper()
	var groups []struct {
		ID   uuid.UUID `json:"id"`
		Name string    `json:"name"`
		Role string    `json:"role"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &groups); err != nil {
		t.Fatalf("failed to decode groups from %s: %v", rec.Body.String(), err)
	}
	byID := make(map[uuid.UUID]string, len(groups))
	for _, g := range groups {
		byID[g.ID] = g.Role
	}
	return byID
}

// TestGetMyGroups_Integration_IncludesCallerRole covers what the endpoint
// gained: each group comes back tagged with the *caller's own* role in it, so
// a client can tell where it may act as an admin without one request per
// group. The same group is therefore seen as "admin" by its creator and
// "member" by someone who joined it by invite code.
func TestGetMyGroups_Integration_IncludesCallerRole(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newBootstrapEnv(t, tx)

	_, creatorToken := env.newAuthenticatedPlayer(t,
		"Zzz My Groups Role Creator", "my-groups-role-creator@example.com")
	joinerID, joinerToken := env.newAuthenticatedPlayer(t,
		"Zzz My Groups Role Joiner", "my-groups-role-joiner@example.com")

	createRec := env.do(http.MethodPost, "/groups", creatorToken, createGroupBody("Zzz My Groups Role Group"))
	if createRec.Code != http.StatusOK {
		t.Fatalf("POST /groups returned status %d, body: %s", createRec.Code, createRec.Body.String())
	}
	groupID := decodeGroupID(t, createRec)

	// A second group the joiner creates, so the joiner's list mixes both
	// roles and a wrong query (e.g. one dropping the per-player filter on the
	// membership row) can't accidentally pass.
	joinerOwnRec := env.do(http.MethodPost, "/groups", joinerToken, createGroupBody("Zzz My Groups Role Own Group"))
	if joinerOwnRec.Code != http.StatusOK {
		t.Fatalf("POST /groups returned status %d, body: %s", joinerOwnRec.Code, joinerOwnRec.Body.String())
	}
	joinerOwnID := decodeGroupID(t, joinerOwnRec)

	if err := env.memberships.AddPlayerToGroup(groupID, joinerID); err != nil {
		t.Fatalf("failed to add joiner to the creator's group: %v", err)
	}

	creatorRec := env.do(http.MethodGet, "/groups/me", creatorToken, nil)
	if creatorRec.Code != http.StatusOK {
		t.Fatalf("GET /groups/me returned status %d, want 200, body: %s", creatorRec.Code, creatorRec.Body.String())
	}
	creatorRoles := decodeGroupsWithRole(t, creatorRec)
	if got := creatorRoles[groupID]; got != models.RoleAdmin {
		t.Errorf("creator sees role %q for their own group, want %q (body: %s)", got, models.RoleAdmin, creatorRec.Body.String())
	}
	if _, ok := creatorRoles[joinerOwnID]; ok {
		t.Errorf("GET /groups/me leaked a group the caller doesn't belong to: %s", creatorRec.Body.String())
	}

	joinerRec := env.do(http.MethodGet, "/groups/me", joinerToken, nil)
	if joinerRec.Code != http.StatusOK {
		t.Fatalf("GET /groups/me returned status %d, want 200, body: %s", joinerRec.Code, joinerRec.Body.String())
	}
	joinerRoles := decodeGroupsWithRole(t, joinerRec)
	if got := joinerRoles[groupID]; got != models.RoleMember {
		t.Errorf("joiner sees role %q for the creator's group, want %q (body: %s)", got, models.RoleMember, joinerRec.Body.String())
	}
	if got := joinerRoles[joinerOwnID]; got != models.RoleAdmin {
		t.Errorf("joiner sees role %q for their own group, want %q (body: %s)", got, models.RoleAdmin, joinerRec.Body.String())
	}

	// Adding a role must not have opened a hole in the invite-code secrecy:
	// GroupWithRole embeds Group, which keeps InviteCode behind json:"-".
	if strings.Contains(joinerRec.Body.String(), "invite_code") ||
		strings.Contains(joinerRec.Body.String(), "InviteCode") {
		t.Errorf("GET /groups/me exposes the invite code: %s", joinerRec.Body.String())
	}
}
