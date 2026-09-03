package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"app/internal/handlers"
	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const testMatchRegistrationJWTSecret = "zzz-integration-test-match-registration-secret"

// registrationEnv mirrors main.go's wiring for the five sign-up routes plus
// POST /matches, all bound to the same rolled-back transaction. The middleware
// chain is the real one: what these tests mostly prove is which status code the
// *authorization* layer produces, so substituting a looser chain would prove
// nothing.
type registrationEnv struct {
	groups        *services.GroupService
	memberships   *services.GroupMembershipService
	players       *services.PlayerService
	matches       *services.MatchService
	registrations *services.MatchRegistrationService
	auth          *services.AuthService
	router        *gin.Engine
}

func newRegistrationEnv(t *testing.T, tx *gorm.DB) *registrationEnv {
	t.Helper()

	membershipService := services.NewGroupMembershipService(tx)
	authService := services.NewAuthService(tx, testMatchRegistrationJWTSecret)
	matchService := services.NewMatchService(tx)
	registrationService := services.NewMatchRegistrationService(tx)
	matchHandler := handlers.NewMatchHandler(matchService, membershipService)
	registrationHandler := handlers.NewMatchRegistrationHandler(registrationService)

	authRequired := handlers.AuthMiddleware(authService)
	requireGroupAdmin := handlers.RequireGroupAdmin(membershipService)
	requireGroupMemberByMatchID := handlers.RequireGroupMembershipByMatchPathParam(matchService, membershipService, "id")
	requireGroupAdminByMatchID := handlers.RequireGroupAdminByMatchPathParam(matchService, membershipService, "id")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/matches", authRequired, requireGroupAdmin, matchHandler.CreateMatch)
	router.POST("/matches/:id/registrations", authRequired, requireGroupMemberByMatchID, registrationHandler.Register)
	router.DELETE("/matches/:id/registrations", authRequired, requireGroupMemberByMatchID, registrationHandler.Unregister)
	router.GET("/matches/:id/registrations", authRequired, requireGroupMemberByMatchID, registrationHandler.ListRegistrations)
	router.POST("/matches/:id/registrations/close", authRequired, requireGroupAdminByMatchID, registrationHandler.CloseRegistrations)
	router.POST("/matches/:id/registrations/reopen", authRequired, requireGroupAdminByMatchID, registrationHandler.ReopenRegistrations)

	return &registrationEnv{
		groups:        services.NewGroupService(tx),
		memberships:   membershipService,
		players:       services.NewPlayerService(tx),
		matches:       matchService,
		registrations: registrationService,
		auth:          authService,
		router:        router,
	}
}

func (e *registrationEnv) authenticatedPlayer(t *testing.T, name, email string) (uuid.UUID, string) {
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

func (e *registrationEnv) do(method, path, token string, body any) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			panic(err)
		}
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

// registrationGroup builds a group with one admin and one plain member, both
// authenticated.
type registrationGroup struct {
	id          uuid.UUID
	adminID     uuid.UUID
	adminToken  string
	memberID    uuid.UUID
	memberToken string
}

func (e *registrationEnv) newGroup(t *testing.T, label string) registrationGroup {
	t.Helper()

	group, err := e.groups.CreateGroup("Zzz Reg "+label, services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	adminID, adminToken := e.authenticatedPlayer(t,
		"Zzz Reg "+label+" Admin", "reg-"+label+"-admin@example.com")
	if err := e.memberships.AddPlayerToGroupWithRole(group.ID, adminID, models.RoleAdmin); err != nil {
		t.Fatalf("failed to add admin: %v", err)
	}

	memberID, memberToken := e.authenticatedPlayer(t,
		"Zzz Reg "+label+" Member", "reg-"+label+"-member@example.com")
	if err := e.memberships.AddPlayerToGroup(group.ID, memberID); err != nil {
		t.Fatalf("failed to add member: %v", err)
	}

	return registrationGroup{
		id:          group.ID,
		adminID:     adminID,
		adminToken:  adminToken,
		memberID:    memberID,
		memberToken: memberToken,
	}
}

// scheduledMatch creates a match open to sign-ups right now: registrations
// opened an hour ago, kick-off is in a day.
func (e *registrationEnv) scheduledMatch(t *testing.T, groupID uuid.UUID, maxPlayers int) uuid.UUID {
	t.Helper()
	kickoff := time.Now().Add(24 * time.Hour)
	opens := time.Now().Add(-time.Hour)
	return e.createMatch(t, groupID, services.MatchSpec{
		ScheduledAt:         &kickoff,
		RegistrationOpensAt: &opens,
		MaxPlayers:          &maxPlayers,
	})
}

func (e *registrationEnv) createMatch(t *testing.T, groupID uuid.UUID, spec services.MatchSpec) uuid.UUID {
	t.Helper()
	id, err := e.matches.CreateMatch(spec, groupID)
	if err != nil {
		t.Fatalf("failed to create match: %v", err)
	}
	return id
}

func decodeEntry(t *testing.T, rec *httptest.ResponseRecorder) models.MatchRegistrationEntry {
	t.Helper()
	var entry models.MatchRegistrationEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &entry); err != nil {
		t.Fatalf("failed to unmarshal registration entry from %s: %v", rec.Body.String(), err)
	}
	return entry
}

// TestRegister_Integration_ReportsPositionAndWaiting is the core of the
// endpoint's contract: signing up answers with the caller's *resulting* entry,
// so the client learns its position (and whether it is on the bench) without a
// second request.
func TestRegister_Integration_ReportsPositionAndWaiting(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newRegistrationEnv(t, tx)

	group := env.newGroup(t, "position")
	matchID := env.scheduledMatch(t, group.id, 16)
	path := "/matches/" + matchID.String() + "/registrations"

	firstRec := env.do(http.MethodPost, path, group.memberToken, nil)
	if firstRec.Code != http.StatusOK {
		t.Fatalf("member POST registrations returned status %d, want 200, body: %s", firstRec.Code, firstRec.Body.String())
	}
	first := decodeEntry(t, firstRec)
	if first.PlayerID != group.memberID {
		t.Errorf("response PlayerID = %s, want the caller's own id %s", first.PlayerID, group.memberID)
	}
	if first.Position != 1 {
		t.Errorf("first sign-up Position = %d, want 1", first.Position)
	}
	if first.IsWaiting {
		t.Error("first sign-up on a max-16 match came back IsWaiting = true")
	}
	if first.Name == "" {
		t.Error("response Name is empty, want the player's display name")
	}
	if first.RegisteredAt.IsZero() {
		t.Error("response RegisteredAt is zero, want the sign-up timestamp")
	}

	// A second, distinct member gets position 2 — proof the position is read
	// from the real list rather than hardcoded.
	secondRec := env.do(http.MethodPost, path, group.adminToken, nil)
	if secondRec.Code != http.StatusOK {
		t.Fatalf("admin POST registrations returned status %d, want 200, body: %s", secondRec.Code, secondRec.Body.String())
	}
	if second := decodeEntry(t, secondRec); second.Position != 2 {
		t.Errorf("second sign-up Position = %d, want 2", second.Position)
	}

	// And a repeat sign-up is a conflict, not a silent duplicate.
	repeatRec := env.do(http.MethodPost, path, group.memberToken, nil)
	if repeatRec.Code != http.StatusConflict {
		t.Errorf("repeat POST registrations returned status %d, want 409, body: %s", repeatRec.Code, repeatRec.Body.String())
	}

	// GET returns both, in order.
	listRec := env.do(http.MethodGet, path, group.memberToken, nil)
	if listRec.Code != http.StatusOK {
		t.Fatalf("GET registrations returned status %d, want 200, body: %s", listRec.Code, listRec.Body.String())
	}
	var entries []models.MatchRegistrationEntry
	if err := json.Unmarshal(listRec.Body.Bytes(), &entries); err != nil {
		t.Fatalf("failed to unmarshal registration list: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("GET registrations returned %d entries, want 2, body: %s", len(entries), listRec.Body.String())
	}
	if entries[0].PlayerID != group.memberID || entries[1].PlayerID != group.adminID {
		t.Errorf("registration list is out of sign-up order: %s then %s", entries[0].PlayerID, entries[1].PlayerID)
	}
}

// TestRegister_Integration_SurplusSignUpWaits covers the design's central
// non-error: the 17th sign-up on a max-16 match still succeeds, and the
// response is what tells the caller they are on the waiting list.
func TestRegister_Integration_SurplusSignUpWaits(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newRegistrationEnv(t, tx)

	group := env.newGroup(t, "waiting")
	matchID := env.scheduledMatch(t, group.id, 16)

	// The first 16 sign-ups go in through the service directly: this test is
	// about the 17th one's HTTP response, and 16 more bcrypt-backed logins
	// would only make it slow.
	for i := 0; i < 16; i++ {
		filler, err := env.players.CreatePlayer(fmt.Sprintf("Zzz Reg Waiting Filler %d", i))
		if err != nil {
			t.Fatalf("failed to create filler player %d: %v", i, err)
		}
		if err := env.memberships.AddPlayerToGroup(group.id, filler); err != nil {
			t.Fatalf("failed to add filler player %d to the group: %v", i, err)
		}
		if err := env.registrations.Register(matchID, filler); err != nil {
			t.Fatalf("failed to register filler player %d: %v", i, err)
		}
	}

	rec := env.do(http.MethodPost, "/matches/"+matchID.String()+"/registrations", group.memberToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("17th POST registrations returned status %d, want 200 (capacity is not an error), body: %s", rec.Code, rec.Body.String())
	}
	entry := decodeEntry(t, rec)
	if entry.Position != 17 {
		t.Errorf("17th sign-up Position = %d, want 17", entry.Position)
	}
	if !entry.IsWaiting {
		t.Error("17th sign-up on a max-16 match came back IsWaiting = false")
	}
}

// TestMatchRegistrations_Integration_OtherGroupGets404 is the security test of
// this slice. A player who belongs to a *different* group — and is an admin of
// it, so no privilege is missing on their own side — must be told the match
// does not exist on every one of the five routes. A 403 anywhere here would
// confirm the match's existence and make other groups' match ids enumerable.
func TestMatchRegistrations_Integration_OtherGroupGets404(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newRegistrationEnv(t, tx)

	owner := env.newGroup(t, "owner")
	outsider := env.newGroup(t, "outsider")
	matchID := env.scheduledMatch(t, owner.id, 16)

	base := "/matches/" + matchID.String() + "/registrations"
	cases := []struct {
		method string
		path   string
	}{
		{http.MethodPost, base},
		{http.MethodDelete, base},
		{http.MethodGet, base},
		{http.MethodPost, base + "/close"},
		{http.MethodPost, base + "/reopen"},
	}

	for _, tc := range cases {
		rec := env.do(tc.method, tc.path, outsider.adminToken, nil)
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s %s as an admin of another group returned status %d, want 404, body: %s",
				tc.method, tc.path, rec.Code, rec.Body.String())
		}
	}

	// And nothing was written: the outsider's POST must not have created a
	// sign-up before the authorization check ran.
	entries, err := env.registrations.ListRegistrations(matchID)
	if err != nil {
		t.Fatalf("ListRegistrations returned error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("an outsider's rejected requests still created %d sign-up(s)", len(entries))
	}
}

// TestMatchRegistrations_Integration_MemberCannotCloseOrReopen covers the other
// half of the status-code split: a member of the right group already knows the
// match exists (they can read its sign-up list), so being refused an admin
// action is an honest 403 rather than a 404.
func TestMatchRegistrations_Integration_MemberCannotCloseOrReopen(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newRegistrationEnv(t, tx)

	group := env.newGroup(t, "adminonly")
	matchID := env.scheduledMatch(t, group.id, 16)
	base := "/matches/" + matchID.String() + "/registrations"

	for _, action := range []string{"/close", "/reopen"} {
		rec := env.do(http.MethodPost, base+action, group.memberToken, nil)
		if rec.Code != http.StatusForbidden {
			t.Errorf("plain member POST %s returned status %d, want 403, body: %s", base+action, rec.Code, rec.Body.String())
		}
	}

	// The same member can still read the list.
	if rec := env.do(http.MethodGet, base, group.memberToken, nil); rec.Code != http.StatusOK {
		t.Errorf("plain member GET registrations returned status %d, want 200, body: %s", rec.Code, rec.Body.String())
	}

	// And the admin can close, then reopen.
	closeRec := env.do(http.MethodPost, base+"/close", group.adminToken, nil)
	if closeRec.Code != http.StatusOK {
		t.Fatalf("admin POST close returned status %d, want 200, body: %s", closeRec.Code, closeRec.Body.String())
	}
	var closeBody map[string]bool
	if err := json.Unmarshal(closeRec.Body.Bytes(), &closeBody); err != nil {
		t.Fatalf("failed to unmarshal close response: %v", err)
	}
	if !closeBody["closed"] {
		t.Errorf("close response = %v, want {\"closed\": true}", closeBody)
	}

	reopenRec := env.do(http.MethodPost, base+"/reopen", group.adminToken, nil)
	if reopenRec.Code != http.StatusOK {
		t.Fatalf("admin POST reopen returned status %d, want 200, body: %s", reopenRec.Code, reopenRec.Body.String())
	}
	var reopenBody map[string]bool
	if err := json.Unmarshal(reopenRec.Body.Bytes(), &reopenBody); err != nil {
		t.Fatalf("failed to unmarshal reopen response: %v", err)
	}
	if !reopenBody["reopened"] {
		t.Errorf("reopen response = %v, want {\"reopened\": true}", reopenBody)
	}

	// Reopening really restored the window, rather than just answering 200.
	if rec := env.do(http.MethodPost, base, group.memberToken, nil); rec.Code != http.StatusOK {
		t.Errorf("POST registrations after reopen returned status %d, want 200, body: %s", rec.Code, rec.Body.String())
	}
}

// TestMatchRegistrations_Integration_UnknownAndMalformedID checks the two
// shapes of bad path param: a well-formed uuid naming no match is a 404 (same
// answer as another group's match, by design), and a non-uuid is a 400 rejected
// before any lookup.
func TestMatchRegistrations_Integration_UnknownAndMalformedID(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newRegistrationEnv(t, tx)

	group := env.newGroup(t, "badid")

	unknown := "/matches/" + uuid.New().String() + "/registrations"
	malformed := "/matches/not-a-uuid/registrations"

	for _, tc := range []struct {
		method string
		path   string
		want   int
	}{
		{http.MethodPost, unknown, http.StatusNotFound},
		{http.MethodDelete, unknown, http.StatusNotFound},
		{http.MethodGet, unknown, http.StatusNotFound},
		{http.MethodPost, unknown + "/close", http.StatusNotFound},
		{http.MethodPost, unknown + "/reopen", http.StatusNotFound},
		{http.MethodPost, malformed, http.StatusBadRequest},
		{http.MethodDelete, malformed, http.StatusBadRequest},
		{http.MethodGet, malformed, http.StatusBadRequest},
		{http.MethodPost, malformed + "/close", http.StatusBadRequest},
		{http.MethodPost, malformed + "/reopen", http.StatusBadRequest},
	} {
		rec := env.do(tc.method, tc.path, group.adminToken, nil)
		if rec.Code != tc.want {
			t.Errorf("%s %s returned status %d, want %d, body: %s", tc.method, tc.path, rec.Code, tc.want, rec.Body.String())
		}
	}
}

// TestMatchRegistrations_Integration_RegisterWithdrawRoundTrip walks the whole
// player-facing loop, including the two 409s a withdrawal has: withdrawing
// twice, and the position a re-registration lands on.
func TestMatchRegistrations_Integration_RegisterWithdrawRoundTrip(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newRegistrationEnv(t, tx)

	group := env.newGroup(t, "roundtrip")
	matchID := env.scheduledMatch(t, group.id, 16)
	path := "/matches/" + matchID.String() + "/registrations"

	// Withdrawing without a sign-up is a conflict.
	if rec := env.do(http.MethodDelete, path, group.memberToken, nil); rec.Code != http.StatusConflict {
		t.Errorf("DELETE registrations with no sign-up returned status %d, want 409, body: %s", rec.Code, rec.Body.String())
	}

	if rec := env.do(http.MethodPost, path, group.memberToken, nil); rec.Code != http.StatusOK {
		t.Fatalf("POST registrations returned status %d, want 200, body: %s", rec.Code, rec.Body.String())
	}

	deleteRec := env.do(http.MethodDelete, path, group.memberToken, nil)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("DELETE registrations returned status %d, want 200, body: %s", deleteRec.Code, deleteRec.Body.String())
	}
	var deleteBody map[string]bool
	if err := json.Unmarshal(deleteRec.Body.Bytes(), &deleteBody); err != nil {
		t.Fatalf("failed to unmarshal withdraw response: %v", err)
	}
	if !deleteBody["unregistered"] {
		t.Errorf("withdraw response = %v, want {\"unregistered\": true}", deleteBody)
	}

	// Withdrawing again is a conflict, and the list is empty.
	if rec := env.do(http.MethodDelete, path, group.memberToken, nil); rec.Code != http.StatusConflict {
		t.Errorf("second DELETE registrations returned status %d, want 409, body: %s", rec.Code, rec.Body.String())
	}
	listRec := env.do(http.MethodGet, path, group.memberToken, nil)
	var entries []models.MatchRegistrationEntry
	if err := json.Unmarshal(listRec.Body.Bytes(), &entries); err != nil {
		t.Fatalf("failed to unmarshal registration list: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("after withdrawing, the list still has %d entry/entries", len(entries))
	}

	// Registering again works, and starts over at position 1 — the withdrawal
	// really deleted the row rather than flagging it.
	againRec := env.do(http.MethodPost, path, group.memberToken, nil)
	if againRec.Code != http.StatusOK {
		t.Fatalf("re-POST registrations returned status %d, want 200, body: %s", againRec.Code, againRec.Body.String())
	}
	if again := decodeEntry(t, againRec); again.Position != 1 {
		t.Errorf("re-registration Position = %d, want 1", again.Position)
	}
}

// TestMatchRegistrations_Integration_WindowConflicts covers the three ways
// sign-ups are refused on a properly scheduled match — too early, closed by an
// admin, and kick-off passed with the admin never having closed anything. All
// three are 409: the request is legitimate, the match just isn't in the state
// for it.
func TestMatchRegistrations_Integration_WindowConflicts(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newRegistrationEnv(t, tx)

	group := env.newGroup(t, "window")

	// Not open yet: registrations open tomorrow, kick-off the day after.
	notOpenKickoff := time.Now().Add(48 * time.Hour)
	notOpenOpens := time.Now().Add(24 * time.Hour)
	notOpenID := env.createMatch(t, group.id, services.MatchSpec{
		ScheduledAt:         &notOpenKickoff,
		RegistrationOpensAt: &notOpenOpens,
		MaxPlayers:          intPtr(16),
	})
	if rec := env.do(http.MethodPost, "/matches/"+notOpenID.String()+"/registrations", group.memberToken, nil); rec.Code != http.StatusConflict {
		t.Errorf("POST registrations before the window opens returned status %d, want 409, body: %s", rec.Code, rec.Body.String())
	}

	// Closed by an admin, through the real route.
	closedID := env.scheduledMatch(t, group.id, 16)
	closedBase := "/matches/" + closedID.String() + "/registrations"
	if rec := env.do(http.MethodPost, closedBase+"/close", group.adminToken, nil); rec.Code != http.StatusOK {
		t.Fatalf("admin POST close returned status %d, want 200, body: %s", rec.Code, rec.Body.String())
	}
	if rec := env.do(http.MethodPost, closedBase, group.memberToken, nil); rec.Code != http.StatusConflict {
		t.Errorf("POST registrations after the admin closed returned status %d, want 409, body: %s", rec.Code, rec.Body.String())
	}
	// Withdrawing is gated on the same window, so it is refused too.
	if rec := env.do(http.MethodDelete, closedBase, group.memberToken, nil); rec.Code != http.StatusConflict {
		t.Errorf("DELETE registrations after the admin closed returned status %d, want 409, body: %s", rec.Code, rec.Body.String())
	}

	// Kick-off passed, with the admin never closing anything: the backstop.
	pastKickoff := time.Now().Add(-time.Hour)
	pastOpens := time.Now().Add(-48 * time.Hour)
	pastID := env.createMatch(t, group.id, services.MatchSpec{
		ScheduledAt:         &pastKickoff,
		RegistrationOpensAt: &pastOpens,
		MaxPlayers:          intPtr(16),
	})
	if rec := env.do(http.MethodPost, "/matches/"+pastID.String()+"/registrations", group.memberToken, nil); rec.Code != http.StatusConflict {
		t.Errorf("POST registrations after kick-off returned status %d, want 409, body: %s", rec.Code, rec.Body.String())
	}
}

// TestMatchRegistrations_Integration_UnscheduledMatch checks the one non-409
// refusal: a match with no schedule has no sign-up list at all, so asking it to
// take one is a malformed request (400), not a state conflict. Reading the
// (empty) list of one is still fine — a client rendering a match shouldn't have
// to know about scheduling first.
func TestMatchRegistrations_Integration_UnscheduledMatch(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newRegistrationEnv(t, tx)

	group := env.newGroup(t, "unscheduled")
	matchID := env.createMatch(t, group.id, services.MatchSpec{Date: models.Date{}})
	base := "/matches/" + matchID.String() + "/registrations"

	for _, tc := range []struct {
		method string
		path   string
		want   int
	}{
		{http.MethodPost, base, http.StatusBadRequest},
		{http.MethodDelete, base, http.StatusBadRequest},
		{http.MethodGet, base, http.StatusOK},
	} {
		token := group.memberToken
		rec := env.do(tc.method, tc.path, token, nil)
		if rec.Code != tc.want {
			t.Errorf("%s %s on an unscheduled match returned status %d, want %d, body: %s",
				tc.method, tc.path, rec.Code, tc.want, rec.Body.String())
		}
	}

	for _, action := range []string{"/close", "/reopen"} {
		if rec := env.do(http.MethodPost, base+action, group.adminToken, nil); rec.Code != http.StatusBadRequest {
			t.Errorf("admin POST %s on an unscheduled match returned status %d, want 400, body: %s", base+action, rec.Code, rec.Body.String())
		}
	}
}

// TestCreateMatch_Integration_Scheduled covers POST /matches' new scheduling
// fields end to end: a full, valid schedule is stored (and the calendar Date is
// derived from the kick-off in the client's *own* offset, not the server's), a
// partial one is rejected, and so is a non-positive roster size.
func TestCreateMatch_Integration_Scheduled(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newRegistrationEnv(t, tx)

	group := env.newGroup(t, "create")

	// A 23:30 kick-off at +02:00 (Paris in summer) is 21:30 UTC the same day,
	// but 2026-09-07 would be the day *before* in some server zones — so the
	// offset the client sent is what has to decide the derived Date. Fixed
	// dates are used here on purpose: this half of the test is about the
	// derivation, not about the registration window, and CreateMatch accepts a
	// past kick-off.
	rec := env.do(http.MethodPost, "/matches", group.adminToken, map[string]any{
		"group_id":              group.id.String(),
		"scheduled_at":          "2026-09-06T23:30:00+02:00",
		"registration_opens_at": "2026-09-01T12:00:00+02:00",
		"max_players":           16,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /matches with a full schedule returned status %d, want 200, body: %s", rec.Code, rec.Body.String())
	}
	var matchID uuid.UUID
	if err := json.Unmarshal(rec.Body.Bytes(), &matchID); err != nil {
		t.Fatalf("failed to unmarshal created match id from %s: %v", rec.Body.String(), err)
	}

	var stored models.Match
	if err := tx.Where("id = ?", matchID).First(&stored).Error; err != nil {
		t.Fatalf("failed to load the created match: %v", err)
	}
	if !stored.IsScheduled() {
		t.Fatal("the created match is not scheduled: ScheduledAt is nil")
	}
	if stored.RegistrationOpensAt == nil {
		t.Error("the created match has no RegistrationOpensAt")
	}
	if stored.MaxPlayers == nil || *stored.MaxPlayers != 16 {
		t.Errorf("stored MaxPlayers = %v, want 16", stored.MaxPlayers)
	}
	if stored.RegistrationsClosedAt != nil {
		t.Error("a freshly created match already has RegistrationsClosedAt set")
	}
	if got := time.Time(stored.Date).Format("2006-01-02"); got != "2026-09-06" {
		t.Errorf("derived Date = %s, want 2026-09-06 (the kick-off's day in its own offset)", got)
	}
	// Proof a schedule created over HTTP is actually usable: a member can sign
	// up for it. Its window has to be relative to now (opened an hour ago,
	// kick-off tomorrow) rather than fixed, since that is exactly what
	// RegistrationWindowError compares against — and it is still expressed in a
	// non-UTC offset, the shape a browser sends.
	paris := time.FixedZone("UTC+2", 2*60*60)
	openRec := env.do(http.MethodPost, "/matches", group.adminToken, map[string]any{
		"group_id":              group.id.String(),
		"scheduled_at":          time.Now().Add(24 * time.Hour).In(paris).Format(time.RFC3339),
		"registration_opens_at": time.Now().Add(-time.Hour).In(paris).Format(time.RFC3339),
		"max_players":           16,
	})
	if openRec.Code != http.StatusOK {
		t.Fatalf("POST /matches with an open window returned status %d, want 200, body: %s", openRec.Code, openRec.Body.String())
	}
	var openID uuid.UUID
	if err := json.Unmarshal(openRec.Body.Bytes(), &openID); err != nil {
		t.Fatalf("failed to unmarshal created match id: %v", err)
	}
	if regRec := env.do(http.MethodPost, "/matches/"+openID.String()+"/registrations", group.memberToken, nil); regRec.Code != http.StatusOK {
		t.Errorf("POST registrations on the created match returned status %d, want 200, body: %s", regRec.Code, regRec.Body.String())
	}

	// A partial schedule: kick-off with no opening time and no roster size.
	partialRec := env.do(http.MethodPost, "/matches", group.adminToken, map[string]any{
		"group_id":     group.id.String(),
		"scheduled_at": "2026-09-13T21:00:00+02:00",
	})
	if partialRec.Code != http.StatusBadRequest {
		t.Errorf("POST /matches with a partial schedule returned status %d, want 400, body: %s", partialRec.Code, partialRec.Body.String())
	}

	// max_players: 0 would bench every single sign-up.
	zeroRec := env.do(http.MethodPost, "/matches", group.adminToken, map[string]any{
		"group_id":              group.id.String(),
		"scheduled_at":          "2026-09-13T21:00:00+02:00",
		"registration_opens_at": "2026-09-08T12:00:00+02:00",
		"max_players":           0,
	})
	if zeroRec.Code != http.StatusBadRequest {
		t.Errorf("POST /matches with max_players: 0 returned status %d, want 400, body: %s", zeroRec.Code, zeroRec.Body.String())
	}

	// A window that only opens after kick-off can never be used.
	backwardsRec := env.do(http.MethodPost, "/matches", group.adminToken, map[string]any{
		"group_id":              group.id.String(),
		"scheduled_at":          "2026-09-13T21:00:00+02:00",
		"registration_opens_at": "2026-09-14T12:00:00+02:00",
		"max_players":           16,
	})
	if backwardsRec.Code != http.StatusBadRequest {
		t.Errorf("POST /matches with a window opening after kick-off returned status %d, want 400, body: %s", backwardsRec.Code, backwardsRec.Body.String())
	}
}

// TestCreateMatch_Integration_UnscheduledStillWorks is the regression guard for
// the whole "purely additive" premise: the payload the existing frontend sends —
// a date and a group, nothing else — still creates a plain unscheduled match.
func TestCreateMatch_Integration_UnscheduledStillWorks(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newRegistrationEnv(t, tx)

	group := env.newGroup(t, "plain")

	rec := env.do(http.MethodPost, "/matches", group.adminToken, map[string]string{
		"date":     "2026-01-04",
		"group_id": group.id.String(),
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /matches with no scheduling fields returned status %d, want 200, body: %s", rec.Code, rec.Body.String())
	}
	var matchID uuid.UUID
	if err := json.Unmarshal(rec.Body.Bytes(), &matchID); err != nil {
		t.Fatalf("failed to unmarshal created match id from %s: %v", rec.Body.String(), err)
	}

	var stored models.Match
	if err := tx.Where("id = ?", matchID).First(&stored).Error; err != nil {
		t.Fatalf("failed to load the created match: %v", err)
	}
	if stored.IsScheduled() {
		t.Error("a match created with no scheduling fields came out scheduled")
	}
	if stored.RegistrationOpensAt != nil || stored.MaxPlayers != nil || stored.RegistrationsClosedAt != nil {
		t.Errorf("a plain match carries scheduling data: opens=%v max=%v closed=%v",
			stored.RegistrationOpensAt, stored.MaxPlayers, stored.RegistrationsClosedAt)
	}
	if got := time.Time(stored.Date).Format("2006-01-02"); got != "2026-01-04" {
		t.Errorf("stored Date = %s, want the submitted 2026-01-04", got)
	}

	// The old date-only payload must not accidentally become registrable.
	if regRec := env.do(http.MethodPost, "/matches/"+matchID.String()+"/registrations", group.memberToken, nil); regRec.Code != http.StatusBadRequest {
		t.Errorf("POST registrations on a plain match returned status %d, want 400, body: %s", regRec.Code, regRec.Body.String())
	}

	// registrations_closed_at is unrepresentable on this endpoint: the request
	// struct has no such field, so a client sending it is simply ignored rather
	// than creating a match nobody could ever sign up for.
	sneakyRec := env.do(http.MethodPost, "/matches", group.adminToken, map[string]any{
		"group_id":                group.id.String(),
		"scheduled_at":            "2026-09-20T21:00:00+02:00",
		"registration_opens_at":   "2026-09-15T12:00:00+02:00",
		"max_players":             16,
		"registrations_closed_at": "2026-09-16T12:00:00+02:00",
	})
	if sneakyRec.Code != http.StatusOK {
		t.Fatalf("POST /matches carrying registrations_closed_at returned status %d, want 200, body: %s", sneakyRec.Code, sneakyRec.Body.String())
	}
	var sneakyID uuid.UUID
	if err := json.Unmarshal(sneakyRec.Body.Bytes(), &sneakyID); err != nil {
		t.Fatalf("failed to unmarshal created match id: %v", err)
	}
	var sneaky models.Match
	if err := tx.Where("id = ?", sneakyID).First(&sneaky).Error; err != nil {
		t.Fatalf("failed to load the created match: %v", err)
	}
	if sneaky.RegistrationsClosedAt != nil {
		t.Errorf("a client-supplied registrations_closed_at was stored: %v", sneaky.RegistrationsClosedAt)
	}
}

func intPtr(v int) *int { return &v }
