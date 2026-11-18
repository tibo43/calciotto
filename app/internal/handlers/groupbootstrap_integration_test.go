package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
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
// invitation routes require administration of the group in the path.
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
	requireGroupAdminByPathID := handlers.RequireGroupAdminByPathParam(membershipService, "id")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/groups", authRequired, groupHandler.CreateGroup)
	router.POST("/groups/join", authRequired, groupHandler.JoinGroup)
	router.GET("/groups", authRequired, groupHandler.GetGroups)
	router.GET("/groups/me", authRequired, groupHandler.GetMyGroups)
	router.GET("/groups/:id", authRequired, requireGroupMemberByPathID, groupHandler.GetGroupByID)
	router.PUT("/groups/:id/details", authRequired, requireGroupAdminByPathID, groupHandler.UpdateGroupDetails)
	router.GET("/groups/:id/invite-code", authRequired, requireGroupAdminByPathID, groupHandler.GetInviteCode)
	router.POST("/groups/:id/invite-code", authRequired, requireGroupAdminByPathID, groupHandler.ChangeInvitation)
	router.DELETE("/groups/:id/invite-code", authRequired, requireGroupAdminByPathID, groupHandler.ChangeInvitation)
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

func TestGroupDetails_Integration_AdminEditsAndMembersRead(t *testing.T) {
	env := newBootstrapEnv(t, testutil.BeginTx(t, testutil.OpenDB(t)))
	admin, adminToken := env.newAuthenticatedPlayer(t, "Details admin", "details-admin@example.test")
	member, memberToken := env.newAuthenticatedPlayer(t, "Details member", "details-member@example.test")
	otherAdmin, otherToken := env.newAuthenticatedPlayer(t, "Other admin", "details-other@example.test")
	group := env.createGroupDirect(t, "Details group", admin)
	env.createGroupDirect(t, "Other group", otherAdmin)
	if err := env.memberships.AddPlayerToGroup(group.ID, member); err != nil {
		t.Fatal(err)
	}
	path := "/groups/" + group.ID.String()
	details := models.GroupDetails{
		Description:  "We play Thursday evenings.",
		WebsiteURL:   "https://example.test/request-to-join",
		WebsiteLabel: "Join our group chat",
	}

	for _, actor := range []struct {
		token  string
		status int
	}{
		{"", http.StatusUnauthorized},
		{memberToken, http.StatusForbidden},
		{otherToken, http.StatusForbidden},
		{adminToken, http.StatusOK},
	} {
		if rec := env.do(http.MethodPut, path+"/details", actor.token, details); rec.Code != actor.status {
			t.Fatalf("update status = %d, want %d", rec.Code, actor.status)
		}
	}

	for _, websiteURL := range []string{"mailto:group@example.test", "//example.test", "https://"} {
		if rec := env.do(http.MethodPut, path+"/details", adminToken, models.GroupDetails{
			WebsiteURL: websiteURL,
		}); rec.Code != http.StatusBadRequest {
			t.Fatalf("invalid URL update status = %d, want 400", rec.Code)
		}
	}

	memberGroups := env.do(http.MethodGet, "/groups/me", memberToken, nil)
	var groups []models.GroupWithRole
	if err := json.Unmarshal(memberGroups.Body.Bytes(), &groups); err != nil ||
		memberGroups.Code != http.StatusOK || len(groups) != 1 ||
		groups[0].GroupDetails != details || groups[0].Role != models.RoleMember {
		t.Fatalf("member did not receive group details: %v (status %d)", err, memberGroups.Code)
	}
	if rec := env.do(http.MethodGet, path, otherToken, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("external group read status = %d, want 403", rec.Code)
	}

	cleared := env.do(http.MethodPut, path+"/details", adminToken, models.GroupDetails{
		Description: details.Description,
	})
	var updated models.Group
	if err := json.Unmarshal(cleared.Body.Bytes(), &updated); err != nil || cleared.Code != http.StatusOK ||
		updated.Description != details.Description || updated.WebsiteURL != "" || updated.WebsiteLabel != "" {
		t.Fatalf("clearing website did not preserve the description: %v (status %d)", err, cleared.Code)
	}
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

func TestGroupInvitation_Integration_AdminLifecycle(t *testing.T) {
	tx := testutil.BeginTx(t, testutil.OpenDB(t))
	env := newBootstrapEnv(t, tx)
	adminID, adminToken := env.newAuthenticatedPlayer(t, "Zzz Invite Lifecycle Admin", "invite-admin@example.test")
	memberID, memberToken := env.newAuthenticatedPlayer(t, "Zzz Invite Lifecycle Member", "invite-member@example.test")
	_, outsiderToken := env.newAuthenticatedPlayer(t, "Zzz Invite Lifecycle Outsider", "invite-outsider@example.test")
	group := env.createGroupDirect(t, "Zzz Invite Lifecycle", adminID)
	if err := env.memberships.AddPlayerToGroup(group.ID, memberID); err != nil {
		t.Fatal(err)
	}
	path := "/groups/" + group.ID.String() + "/invite-code"

	for _, token := range []string{memberToken, outsiderToken} {
		for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodDelete} {
			if rec := env.do(method, path, token, nil); rec.Code != http.StatusForbidden {
				t.Fatalf("%s invitation without admin role: got %d, want 403", method, rec.Code)
			}
		}
	}
	if rec := env.do(http.MethodGet, path, "", nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous invitation read: got %d, want 401", rec.Code)
	}
	active := env.do(http.MethodGet, path, adminToken, nil)
	if active.Code != http.StatusOK || decodeInviteCode(t, active) != group.InviteCode {
		t.Fatal("admin could not read the active invitation")
	}
	rotated := env.do(http.MethodPost, path, adminToken, nil)
	if rotated.Code != http.StatusOK {
		t.Fatalf("rotate invitation: got %d", rotated.Code)
	}
	replacement := decodeInviteCode(t, rotated)
	if replacement == "" || replacement == group.InviteCode {
		t.Fatal("rotation did not issue a fresh code")
	}
	if _, err := env.auth.SignupNewPlayer("Zzz Rotation Fixture", "invite-rotation@example.test", "fixture-password", group.InviteCode); !errors.Is(err, services.ErrInviteCodeNotFound) {
		t.Fatalf("old invitation is still accepted: %v", err)
	}
	if rec := env.do(http.MethodDelete, path, adminToken, nil); rec.Code != http.StatusOK || decodeInviteCode(t, rec) != "" {
		t.Fatal("admin could not revoke the invitation")
	}
	if _, err := env.auth.SignupNewPlayer("Zzz Revoked Fixture", "invite-revoked@example.test", "fixture-password", replacement); !errors.Is(err, services.ErrInviteCodeNotFound) {
		t.Fatalf("revoked invitation is still accepted: %v", err)
	}
}

func TestGroupReads_Integration_RequireMembership(t *testing.T) {
	tx := testutil.BeginTx(t, testutil.OpenDB(t))
	env := newBootstrapEnv(t, tx)
	aID, aToken := env.newAuthenticatedPlayer(t, "Zzz Group Read A", "group-read-a@example.test")
	bID, _ := env.newAuthenticatedPlayer(t, "Zzz Group Read B", "group-read-b@example.test")
	a := env.createGroupDirect(t, "Zzz Visible Group", aID)
	b := env.createGroupDirect(t, "Zzz Other Group", bID)

	all := env.do(http.MethodGet, "/groups", aToken, nil)
	var groups []models.Group
	if all.Code != http.StatusOK || json.Unmarshal(all.Body.Bytes(), &groups) != nil ||
		len(groups) != 1 || groups[0].ID != a.ID {
		t.Fatalf("list should return only the caller's group, got %d groups and status %d", len(groups), all.Code)
	}
	if rec := env.do(http.MethodGet, "/groups/"+b.ID.String(), aToken, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("other group read: got %d, want 403", rec.Code)
	}
}

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
		{"GET /groups as member", "/groups", memberToken},
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

// TestGetGroups_Integration_RequiresAuthentication and its :id sibling pin the
// fix for the enumeration gap these two routes used to have: unauthenticated,
// anyone on the internet could list the name of every group in the system in
// one request. They still return every group regardless of the caller's own
// membership (see TestGetMyGroups_Integration_ReturnsOnlyTheCallersGroups for
// the route that actually scopes to the caller) — authRequired only raises
// the bar from "anonymous, at scale" to "any registered player," the same
// trust boundary every other read in this app already assumes.
func TestGetGroups_Integration_RequiresAuthentication(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newBootstrapEnv(t, tx)

	rec := env.do(http.MethodGet, "/groups", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("GET /groups anonymous returned status %d, want 401, body: %s", rec.Code, rec.Body.String())
	}
}

func TestGetGroupByID_Integration_RequiresAuthentication(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newBootstrapEnv(t, tx)

	memberID, _ := env.newAuthenticatedPlayer(t,
		"Zzz Bootstrap Auth Gate Member", "bootstrap-auth-gate-member@example.com")
	stored := env.createGroupDirect(t, "Zzz Bootstrap Auth Gate Group", memberID)

	rec := env.do(http.MethodGet, "/groups/"+stored.ID.String(), "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("GET /groups/:id anonymous returned status %d, want 401, body: %s", rec.Code, rec.Body.String())
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
