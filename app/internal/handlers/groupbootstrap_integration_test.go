package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"app/internal/handlers"
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

func decodeGroupID(t *testing.T, rec *httptest.ResponseRecorder) uuid.UUID {
	t.Helper()
	var created struct {
		ID uuid.UUID `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode group from response %s: %v", rec.Body.String(), err)
	}
	return created.ID
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

// TestCreateGroup_Integration_CreatorBecomesFirstMember covers the actual
// bootstrapping hole: POST /groups/:id/players already requires membership of
// the target group, so unless creating a group also creates the creator's
// membership, a new group can never gain its first member.
func TestCreateGroup_Integration_CreatorBecomesFirstMember(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newBootstrapEnv(t, tx)

	creatorID, creatorToken := env.newAuthenticatedPlayer(t,
		"Zzz Bootstrap Creator", "bootstrap-creator@example.com")

	rec := env.do(http.MethodPost, "/groups", creatorToken, map[string]string{"name": "Zzz Bootstrap Group"})
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /groups returned status %d, want 200, body: %s", rec.Code, rec.Body.String())
	}
	groupID := decodeGroupID(t, rec)
	if groupID == uuid.Nil {
		t.Fatal("POST /groups returned a nil group id")
	}

	isMember, err := env.memberships.IsMember(groupID, creatorID)
	if err != nil {
		t.Fatalf("IsMember returned error: %v", err)
	}
	if !isMember {
		t.Error("creator is not a member of the group they just created")
	}

	groups, err := env.memberships.GetGroupsByPlayerID(creatorID)
	if err != nil {
		t.Fatalf("GetGroupsByPlayerID returned error: %v", err)
	}
	found := false
	for _, group := range groups {
		if group.ID == groupID {
			found = true
		}
	}
	if !found {
		t.Errorf("created group %s missing from the creator's groups: %+v", groupID, groups)
	}
}

// TestCreateGroup_Integration_RequiresAuth pins the route change: POST /groups
// used to be public, but it now has to know who to make the first member.
func TestCreateGroup_Integration_RequiresAuth(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newBootstrapEnv(t, tx)

	rec := env.do(http.MethodPost, "/groups", "", map[string]string{"name": "Zzz Bootstrap Anonymous Group"})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("anonymous POST /groups returned status %d, want 401, body: %s", rec.Code, rec.Body.String())
	}
}

// TestJoinGroup_Integration_GrantsAccessToGroupData is the end-to-end shape of
// the target workflow: a creator shares the invite code, an outsider joins
// with it, and only then do the group's own routes stop answering 403.
func TestJoinGroup_Integration_GrantsAccessToGroupData(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newBootstrapEnv(t, tx)

	_, creatorToken := env.newAuthenticatedPlayer(t,
		"Zzz Bootstrap Join Creator", "bootstrap-join-creator@example.com")
	joinerID, joinerToken := env.newAuthenticatedPlayer(t,
		"Zzz Bootstrap Joiner", "bootstrap-joiner@example.com")

	createRec := env.do(http.MethodPost, "/groups", creatorToken, map[string]string{"name": "Zzz Bootstrap Join Group"})
	if createRec.Code != http.StatusOK {
		t.Fatalf("POST /groups returned status %d, body: %s", createRec.Code, createRec.Body.String())
	}
	groupID := decodeGroupID(t, createRec)

	codeRec := env.do(http.MethodGet, "/groups/"+groupID.String()+"/invite-code", creatorToken, nil)
	if codeRec.Code != http.StatusOK {
		t.Fatalf("creator GET invite-code returned status %d, body: %s", codeRec.Code, codeRec.Body.String())
	}
	inviteCode := decodeInviteCode(t, codeRec)
	if inviteCode == "" {
		t.Fatal("invite code is empty — CreateGroup must generate one")
	}

	// Before joining, the group's data is off limits.
	beforeRec := env.do(http.MethodGet, "/matches/details?group_id="+groupID.String(), joinerToken, nil)
	if beforeRec.Code != http.StatusForbidden {
		t.Fatalf("non-member GET /matches/details returned status %d, want 403, body: %s", beforeRec.Code, beforeRec.Body.String())
	}

	// The code is matched case-insensitively, so a hand-typed lower-case
	// version has to work too.
	joinRec := env.do(http.MethodPost, "/groups/join", joinerToken,
		map[string]string{"invite_code": strings.ToLower(inviteCode)})
	if joinRec.Code != http.StatusOK {
		t.Fatalf("POST /groups/join returned status %d, want 200, body: %s", joinRec.Code, joinRec.Body.String())
	}
	if joined := decodeGroupID(t, joinRec); joined != groupID {
		t.Errorf("POST /groups/join returned group %s, want %s", joined, groupID)
	}

	isMember, err := env.memberships.IsMember(groupID, joinerID)
	if err != nil {
		t.Fatalf("IsMember returned error: %v", err)
	}
	if !isMember {
		t.Error("joiner is not a member of the group after a successful join")
	}

	// After joining, the very same request goes through.
	afterRec := env.do(http.MethodGet, "/matches/details?group_id="+groupID.String(), joinerToken, nil)
	if afterRec.Code != http.StatusOK {
		t.Errorf("member GET /matches/details returned status %d, want 200, body: %s", afterRec.Code, afterRec.Body.String())
	}

	// Joining twice is a client mistake, not a server error.
	againRec := env.do(http.MethodPost, "/groups/join", joinerToken, map[string]string{"invite_code": inviteCode})
	if againRec.Code != http.StatusBadRequest {
		t.Errorf("re-joining returned status %d, want 400 (ErrAlreadyMember), body: %s", againRec.Code, againRec.Body.String())
	}
}

// TestJoinGroup_Integration_RejectsBadInput covers the failure modes of the
// only route that lets a non-member in: an unknown or empty code must not
// match anything, and an anonymous caller has no player to enroll.
func TestJoinGroup_Integration_RejectsBadInput(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newBootstrapEnv(t, tx)

	_, token := env.newAuthenticatedPlayer(t,
		"Zzz Bootstrap Bad Join", "bootstrap-bad-join@example.com")

	unknownRec := env.do(http.MethodPost, "/groups/join", token, map[string]string{"invite_code": "ZZZZZZZZ"})
	if unknownRec.Code != http.StatusNotFound {
		t.Errorf("unknown invite code returned status %d, want 404, body: %s", unknownRec.Code, unknownRec.Body.String())
	}

	// An empty code must not silently match groups predating the invite-code
	// column (whose invite_code is NULL/empty).
	emptyRec := env.do(http.MethodPost, "/groups/join", token, map[string]string{"invite_code": "   "})
	if emptyRec.Code != http.StatusNotFound {
		t.Errorf("empty invite code returned status %d, want 404, body: %s", emptyRec.Code, emptyRec.Body.String())
	}

	anonRec := env.do(http.MethodPost, "/groups/join", "", map[string]string{"invite_code": "ZZZZZZZZ"})
	if anonRec.Code != http.StatusUnauthorized {
		t.Errorf("anonymous POST /groups/join returned status %d, want 401, body: %s", anonRec.Code, anonRec.Body.String())
	}
}

// TestGetInviteCode_Integration_MembersOnly checks that the one endpoint
// exposing the code is gated on current membership — an invite code is a
// shared secret, so anyone able to read it can hand out access to the group.
func TestGetInviteCode_Integration_MembersOnly(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newBootstrapEnv(t, tx)

	_, memberToken := env.newAuthenticatedPlayer(t,
		"Zzz Bootstrap Code Member", "bootstrap-code-member@example.com")
	_, outsiderToken := env.newAuthenticatedPlayer(t,
		"Zzz Bootstrap Code Outsider", "bootstrap-code-outsider@example.com")

	createRec := env.do(http.MethodPost, "/groups", memberToken, map[string]string{"name": "Zzz Bootstrap Code Group"})
	if createRec.Code != http.StatusOK {
		t.Fatalf("POST /groups returned status %d, body: %s", createRec.Code, createRec.Body.String())
	}
	groupID := decodeGroupID(t, createRec)

	stored, err := env.groups.GetGroupByID(groupID)
	if err != nil {
		t.Fatalf("GetGroupByID returned error: %v", err)
	}

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

	_, memberToken := env.newAuthenticatedPlayer(t,
		"Zzz Bootstrap Leak Member", "bootstrap-leak-member@example.com")

	createRec := env.do(http.MethodPost, "/groups", memberToken, map[string]string{"name": "Zzz Bootstrap Leak Group"})
	if createRec.Code != http.StatusOK {
		t.Fatalf("POST /groups returned status %d, body: %s", createRec.Code, createRec.Body.String())
	}
	groupID := decodeGroupID(t, createRec)

	stored, err := env.groups.GetGroupByID(groupID)
	if err != nil {
		t.Fatalf("GetGroupByID returned error: %v", err)
	}
	if stored.InviteCode == "" {
		t.Fatal("stored invite code is empty — nothing to check for leaks")
	}

	cases := []struct {
		name  string
		path  string
		token string
	}{
		{"POST /groups response", "", memberToken},
		{"GET /groups anonymous", "/groups", ""},
		{"GET /groups as member", "/groups", memberToken},
		{"GET /groups/:id anonymous", "/groups/" + groupID.String(), ""},
		{"GET /groups/:id as member", "/groups/" + groupID.String(), memberToken},
	}
	for _, tc := range cases {
		body := createRec.Body.String()
		if tc.path != "" {
			rec := env.do(http.MethodGet, tc.path, tc.token, nil)
			if rec.Code != http.StatusOK {
				t.Fatalf("%s returned status %d, body: %s", tc.name, rec.Code, rec.Body.String())
			}
			body = rec.Body.String()
		}
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

	_, token := env.newAuthenticatedPlayer(t,
		"Zzz My Groups Member", "my-groups-member@example.com")
	_, otherToken := env.newAuthenticatedPlayer(t,
		"Zzz My Groups Outsider", "my-groups-outsider@example.com")

	firstRec := env.do(http.MethodPost, "/groups", token, map[string]string{"name": "Zzz My Groups First"})
	if firstRec.Code != http.StatusOK {
		t.Fatalf("POST /groups returned status %d, body: %s", firstRec.Code, firstRec.Body.String())
	}
	firstID := decodeGroupID(t, firstRec)

	secondRec := env.do(http.MethodPost, "/groups", token, map[string]string{"name": "Zzz My Groups Second"})
	if secondRec.Code != http.StatusOK {
		t.Fatalf("POST /groups returned status %d, body: %s", secondRec.Code, secondRec.Body.String())
	}
	secondID := decodeGroupID(t, secondRec)

	// A third group the caller has nothing to do with — it must not show up.
	foreignRec := env.do(http.MethodPost, "/groups", otherToken, map[string]string{"name": "Zzz My Groups Foreign"})
	if foreignRec.Code != http.StatusOK {
		t.Fatalf("POST /groups returned status %d, body: %s", foreignRec.Code, foreignRec.Body.String())
	}
	foreignID := decodeGroupID(t, foreignRec)

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
