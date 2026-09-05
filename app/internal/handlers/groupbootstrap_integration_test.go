package handlers_test

import (
	"bytes"
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
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const testGroupBootstrapJWTSecret = "zzz-integration-test-group-bootstrap-secret"

// bootstrapEnv bundles the services a group-bootstrapping test needs, all
// bound to the same rolled-back transaction.
type bootstrapEnv struct {
	groups      *services.GroupService
	memberships *services.GroupMembershipService
	players     *services.PlayerService
	auth        *services.AuthService
	router      *gin.Engine
}

// newBootstrapEnv mirrors main.go's wiring for the routes involved in group
// creation and joining, so the middleware chain under test is the real one:
// POST /groups and POST /groups/join take authRequired alone, while
// /groups/:id/invite-code also requires membership of the group in the path.
func newBootstrapEnv(t *testing.T, tx *gorm.DB) *bootstrapEnv {
	t.Helper()

	groupService := services.NewGroupService(tx)
	membershipService := services.NewGroupMembershipService(tx)
	authService := services.NewAuthService(tx, testGroupBootstrapJWTSecret)
	groupHandler := handlers.NewGroupHandler(groupService, membershipService)
	matchHandler := handlers.NewMatchHandler(services.NewMatchService(tx), membershipService)

	authRequired := handlers.AuthMiddleware(authService)
	requireGroupMember := handlers.RequireGroupMembership(membershipService)
	requireGroupMemberByPathID := handlers.RequireGroupMembershipByPathParam(membershipService, "id")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/groups", authRequired, groupHandler.CreateGroup)
	router.POST("/groups/join", authRequired, groupHandler.JoinGroup)
	router.GET("/groups", groupHandler.GetGroups)
	router.GET("/groups/me", authRequired, groupHandler.GetMyGroups)
	router.GET("/groups/:id", groupHandler.GetGroupByID)
	router.GET("/groups/:id/invite-code", authRequired, requireGroupMemberByPathID, groupHandler.GetInviteCode)
	router.GET("/matches/details", authRequired, requireGroupMember, matchHandler.GetMatchesDetails)

	return &bootstrapEnv{
		groups:      groupService,
		memberships: membershipService,
		players:     services.NewPlayerService(tx),
		auth:        authService,
		router:      router,
	}
}

// newAuthenticatedPlayer creates a player, claims it with credentials and logs
// in, returning the player id and a usable bearer token.
func (e *bootstrapEnv) newAuthenticatedPlayer(t *testing.T, name, email string) (uuid.UUID, string) {
	t.Helper()

	id, err := e.players.CreatePlayer(name)
	if err != nil {
		t.Fatalf("failed to create player %q: %v", name, err)
	}
	if err := e.auth.Signup(id, email, "s3cret-pass"); err != nil {
		t.Fatalf("failed to sign up %q: %v", name, err)
	}
	token, err := e.auth.Login(email, "s3cret-pass")
	if err != nil {
		t.Fatalf("failed to log in %q: %v", name, err)
	}
	return id, token
}

// createGroupDirect creates a group and makes creatorID its first admin,
// mirroring exactly what the now-disabled GroupHandler.CreateGroup used to do
// (see CLAUDE.md — POST /groups is a deliberate, reversible 403 today). Tests
// that only need *a* group to exist as setup — rather than testing the
// disabled route itself — use this instead of the old env.do(POST /groups).
func (e *bootstrapEnv) createGroupDirect(t *testing.T, name string, creatorID uuid.UUID) *models.Group {
	t.Helper()
	group, err := e.groups.CreateGroup(name, services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group %q: %v", name, err)
	}
	if err := e.memberships.AddPlayerToGroupWithRole(group.ID, creatorID, models.RoleAdmin); err != nil {
		t.Fatalf("failed to add creator as admin of group %q: %v", name, err)
	}
	return group
}

// do issues a request against the test router. An empty token means "send no
// Authorization header at all", which is how the 401 cases are exercised.
func (e *bootstrapEnv) do(method, path, token string, body any) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// createGroupBody builds a POST /groups request body with the given group
// name and two throwaway team specs (name/colour don't matter to any of
// these tests, only that exactly 2 are present) — CreateGroup now requires
// both on every group, mirroring services.DefaultTeamSpecs.
func createGroupBody(name string) map[string]any {
	return map[string]any{
		"name": name,
		"teams": []map[string]string{
			{"name": "Black", "colour": "black"},
			{"name": "White", "colour": "white"},
		},
	}
}

func decodeInviteCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var payload struct {
		InviteCode string `json:"invite_code"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode invite code from response %s: %v", rec.Body.String(), err)
	}
	return payload.InviteCode
}

// TestCreateGroup_Integration_RequiresAuth pins the one thing that still runs
// before the disabled-feature response: an anonymous caller gets 401 from
// authRequired, never reaching GroupHandler.CreateGroup's 403 at all.
func TestCreateGroup_Integration_RequiresAuth(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newBootstrapEnv(t, tx)

	rec := env.do(http.MethodPost, "/groups", "", createGroupBody("Zzz Bootstrap Anonymous Group"))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("anonymous POST /groups returned status %d, want 401, body: %s", rec.Code, rec.Body.String())
	}
}

// TestCreateGroup_Integration_Disabled pins the deliberate, reversible
// business decision that self-service group creation is off: an authenticated
// caller with an otherwise-valid body still gets 403, not 200. See
// GroupHandler.CreateGroup and CLAUDE.md.
func TestCreateGroup_Integration_Disabled(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newBootstrapEnv(t, tx)

	_, token := env.newAuthenticatedPlayer(t,
		"Zzz Bootstrap Disabled Create", "bootstrap-disabled-create@example.com")

	rec := env.do(http.MethodPost, "/groups", token, createGroupBody("Zzz Bootstrap Disabled Group"))
	if rec.Code != http.StatusForbidden {
		t.Errorf("POST /groups returned status %d, want 403 (disabled), body: %s", rec.Code, rec.Body.String())
	}
}

// TestJoinGroup_Integration_Disabled mirrors TestCreateGroup_Integration_Disabled
// for the other half of the disabled bootstrapping flow: a valid invite code
// still gets 403, never a successful join. See GroupHandler.JoinGroup.
func TestJoinGroup_Integration_Disabled(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newBootstrapEnv(t, tx)

	creatorID, _ := env.newAuthenticatedPlayer(t,
		"Zzz Bootstrap Disabled Join Creator", "bootstrap-disabled-join-creator@example.com")
	_, joinerToken := env.newAuthenticatedPlayer(t,
		"Zzz Bootstrap Disabled Joiner", "bootstrap-disabled-joiner@example.com")

	group := env.createGroupDirect(t, "Zzz Bootstrap Disabled Join Group", creatorID)

	rec := env.do(http.MethodPost, "/groups/join", joinerToken, map[string]string{"invite_code": group.InviteCode})
	if rec.Code != http.StatusForbidden {
		t.Errorf("POST /groups/join returned status %d, want 403 (disabled), body: %s", rec.Code, rec.Body.String())
	}
}

// TestJoinGroup_Integration_RequiresAuth mirrors
// TestCreateGroup_Integration_RequiresAuth for the join route.
func TestJoinGroup_Integration_RequiresAuth(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newBootstrapEnv(t, tx)

	rec := env.do(http.MethodPost, "/groups/join", "", map[string]string{"invite_code": "ZZZZZZZZ"})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("anonymous POST /groups/join returned status %d, want 401, body: %s", rec.Code, rec.Body.String())
	}
}

// TestGetInviteCode_Integration_MembersOnly checks that the one endpoint
// exposing the code is gated on current membership — an invite code is a
// shared secret, so anyone able to read it can hand out access to the group.
func TestGetInviteCode_Integration_MembersOnly(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newBootstrapEnv(t, tx)

	memberID, memberToken := env.newAuthenticatedPlayer(t,
		"Zzz Bootstrap Code Member", "bootstrap-code-member@example.com")
	_, outsiderToken := env.newAuthenticatedPlayer(t,
		"Zzz Bootstrap Code Outsider", "bootstrap-code-outsider@example.com")

	stored := env.createGroupDirect(t, "Zzz Bootstrap Code Group", memberID)
	groupID := stored.ID

	memberRec := env.do(http.MethodGet, "/groups/"+groupID.String()+"/invite-code", memberToken, nil)
	if memberRec.Code != http.StatusOK {
		t.Fatalf("member GET invite-code returned status %d, want 200, body: %s", memberRec.Code, memberRec.Body.String())
	}
	if got := decodeInviteCode(t, memberRec); got != stored.InviteCode {
		t.Errorf("member got invite code %q, want %q", got, stored.InviteCode)
	}

	outsiderRec := env.do(http.MethodGet, "/groups/"+groupID.String()+"/invite-code", outsiderToken, nil)
	if outsiderRec.Code != http.StatusForbidden {
		t.Errorf("non-member GET invite-code returned status %d, want 403, body: %s", outsiderRec.Code, outsiderRec.Body.String())
	}

	anonRec := env.do(http.MethodGet, "/groups/"+groupID.String()+"/invite-code", "", nil)
	if anonRec.Code != http.StatusUnauthorized {
		t.Errorf("anonymous GET invite-code returned status %d, want 401, body: %s", anonRec.Code, anonRec.Body.String())
	}
}

// TestGroupJSON_Integration_NeverExposesInviteCode is the counterpart of the
// SearchPlayer email-leak test: GET /groups and GET /groups/:id are public, so
// the invite code must never ride along in their JSON — not even for a member,
// who has GET /groups/:id/invite-code for that.
func TestGroupJSON_Integration_NeverExposesInviteCode(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newBootstrapEnv(t, tx)

	memberID, memberToken := env.newAuthenticatedPlayer(t,
		"Zzz Bootstrap Leak Member", "bootstrap-leak-member@example.com")

	stored := env.createGroupDirect(t, "Zzz Bootstrap Leak Group", memberID)
	groupID := stored.ID
	if stored.InviteCode == "" {
		t.Fatal("stored invite code is empty — nothing to check for leaks")
	}

	cases := []struct {
		name  string
		path  string
		token string
	}{
		{"GET /groups anonymous", "/groups", ""},
		{"GET /groups as member", "/groups", memberToken},
		{"GET /groups/:id anonymous", "/groups/" + groupID.String(), ""},
		{"GET /groups/:id as member", "/groups/" + groupID.String(), memberToken},
	}
	for _, tc := range cases {
		rec := env.do(http.MethodGet, tc.path, tc.token, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s returned status %d, body: %s", tc.name, rec.Code, rec.Body.String())
		}
		body := rec.Body.String()
		if strings.Contains(body, "invite_code") {
			t.Errorf("%s exposes an invite_code field: %s", tc.name, body)
		}
		if strings.Contains(body, stored.InviteCode) {
			t.Errorf("%s leaks the invite code %q: %s", tc.name, stored.InviteCode, body)
		}
	}
}

// decodeGroups reads a group list response into just the public fields, so a
// leaked invite_code would show up as a body assertion failure rather than
// being silently deserialized.
func decodeGroups(t *testing.T, rec *httptest.ResponseRecorder) []struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
} {
	t.Helper()
	var groups []struct {
		ID   uuid.UUID `json:"id"`
		Name string    `json:"name"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &groups); err != nil {
		t.Fatalf("failed to decode groups from response %s: %v", rec.Body.String(), err)
	}
	return groups
}

// TestGetMyGroups_Integration_ReturnsOnlyTheCallersGroups covers what GET
// /groups can't do: it lists every group in the system, so a client has no way
// to ask "which ones are mine". The caller here belongs to two of the three
// groups that exist, and must get exactly those two.
func TestGetMyGroups_Integration_ReturnsOnlyTheCallersGroups(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newBootstrapEnv(t, tx)

	playerID, token := env.newAuthenticatedPlayer(t,
		"Zzz My Groups Member", "my-groups-member@example.com")
	otherID, _ := env.newAuthenticatedPlayer(t,
		"Zzz My Groups Outsider", "my-groups-outsider@example.com")

	firstID := env.createGroupDirect(t, "Zzz My Groups First", playerID).ID
	secondID := env.createGroupDirect(t, "Zzz My Groups Second", playerID).ID

	// A third group the caller has nothing to do with — it must not show up.
	foreignID := env.createGroupDirect(t, "Zzz My Groups Foreign", otherID).ID

	rec := env.do(http.MethodGet, "/groups/me", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /groups/me returned status %d, want 200, body: %s", rec.Code, rec.Body.String())
	}

	groups := decodeGroups(t, rec)
	if len(groups) != 2 {
		t.Fatalf("GET /groups/me returned %d groups, want 2: %s", len(groups), rec.Body.String())
	}
	seen := map[uuid.UUID]string{}
	for _, group := range groups {
		seen[group.ID] = group.Name
	}
	for _, want := range []uuid.UUID{firstID, secondID} {
		if _, ok := seen[want]; !ok {
			t.Errorf("group %s missing from GET /groups/me: %s", want, rec.Body.String())
		}
	}
	if _, ok := seen[foreignID]; ok {
		t.Errorf("GET /groups/me leaked a group the caller doesn't belong to: %s", rec.Body.String())
	}
	if name := seen[firstID]; name != "Zzz My Groups First" {
		t.Errorf("group %s came back named %q, want %q", firstID, name, "Zzz My Groups First")
	}

	// Same json:"-" guarantee as everywhere else: the code stays exclusive to
	// GET /groups/:id/invite-code.
	if strings.Contains(rec.Body.String(), "invite_code") {
		t.Errorf("GET /groups/me exposes an invite_code field: %s", rec.Body.String())
	}
}

// TestGetMyGroups_Integration_RequiresAuth — the route derives its answer from
// the JWT's player, so there is nothing to return without one.
func TestGetMyGroups_Integration_RequiresAuth(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newBootstrapEnv(t, tx)

	rec := env.do(http.MethodGet, "/groups/me", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("anonymous GET /groups/me returned status %d, want 401, body: %s", rec.Code, rec.Body.String())
	}
}

// TestGetMyGroups_Integration_NoGroupsIsEmptyList pins the state a freshly
// signed-up player is in: belonging to no group is normal, so the answer is an
// empty JSON array — not a 404, and not `null`, which would force the frontend
// to special-case a list it should just be able to render.
func TestGetMyGroups_Integration_NoGroupsIsEmptyList(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newBootstrapEnv(t, tx)

	_, token := env.newAuthenticatedPlayer(t,
		"Zzz My Groups Loner", "my-groups-loner@example.com")

	rec := env.do(http.MethodGet, "/groups/me", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /groups/me returned status %d, want 200, body: %s", rec.Code, rec.Body.String())
	}
	if body := strings.TrimSpace(rec.Body.String()); body != "[]" {
		t.Errorf("GET /groups/me for a groupless player returned %s, want []", body)
	}
	if groups := decodeGroups(t, rec); len(groups) != 0 {
		t.Errorf("GET /groups/me returned %d groups for a groupless player", len(groups))
	}
}
