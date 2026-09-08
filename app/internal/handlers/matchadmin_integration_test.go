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
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const testMatchAdminJWTSecret = "zzz-integration-test-match-admin-secret"

// matchAdminEnv mirrors main.go's wiring for the match routes that matter
// here: creating, updating and deleting are admin-only, reading details stays
// open to any member (requireGroupMember). POST and DELETE carry the group id
// in the body/query, hence the body/query-resolving requireGroupAdmin; PUT
// names the match in its path, so it uses the *match-scoped*
// requireGroupAdminByMatchID exactly as main.go does — the group must come
// from the match being edited, not from anything the caller supplies.
type matchAdminEnv struct {
	memberships *services.GroupMembershipService
	players     *services.PlayerService
	teams       *services.TeamService
	matches     *services.MatchService
	auth        *services.AuthService
	router      *gin.Engine
}

func newMatchAdminEnv(t *testing.T, tx *gorm.DB) *matchAdminEnv {
	t.Helper()

	membershipService := services.NewGroupMembershipService(tx)
	authService := services.NewAuthService(tx, testMatchAdminJWTSecret)
	matchService := services.NewMatchService(tx)
	matchHandler := handlers.NewMatchHandler(matchService, membershipService)

	authRequired := handlers.AuthMiddleware(authService)
	requireGroupMember := handlers.RequireGroupMembership(membershipService)
	requireGroupAdmin := handlers.RequireGroupAdmin(membershipService)
	requireGroupAdminByMatchID := handlers.RequireGroupAdminByMatchPathParam(matchService, membershipService, "id")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/matches", authRequired, requireGroupAdmin, matchHandler.CreateMatch)
	router.PUT("/matches/:id", authRequired, requireGroupAdminByMatchID, matchHandler.UpdateMatch)
	router.DELETE("/matches/:id", authRequired, requireGroupAdmin, matchHandler.DeleteMatch)
	router.GET("/matches/details", authRequired, requireGroupMember, matchHandler.GetMatchesDetails)
	router.GET("/matches/:id/details", authRequired, requireGroupMember, matchHandler.GetMatchDetailsByID)

	return &matchAdminEnv{
		memberships: membershipService,
		players:     services.NewPlayerService(tx),
		teams:       services.NewTeamService(tx),
		matches:     matchService,
		auth:        authService,
		router:      router,
	}
}

func (e *matchAdminEnv) authenticatedPlayer(t *testing.T, name, email string) (uuid.UUID, string) {
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

func (e *matchAdminEnv) do(method, path, token string, body any) *httptest.ResponseRecorder {
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

// matchAdminGroup builds a group with one admin and one plain member, both
// authenticated, and returns the group id plus the two tokens.
func (e *matchAdminEnv) matchAdminGroup(t *testing.T, tx *gorm.DB, name, adminEmail, memberEmail string) (uuid.UUID, string, string) {
	t.Helper()

	group, err := services.NewGroupService(tx).CreateGroup(name, services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	adminID, adminToken := e.authenticatedPlayer(t, name+" Admin", adminEmail)
	if err := e.memberships.AddPlayerToGroupWithRole(group.ID, adminID, models.RoleAdmin); err != nil {
		t.Fatalf("failed to add admin: %v", err)
	}

	memberID, memberToken := e.authenticatedPlayer(t, name+" Member", memberEmail)
	if err := e.memberships.AddPlayerToGroup(group.ID, memberID); err != nil {
		t.Fatalf("failed to add member: %v", err)
	}

	return group.ID, adminToken, memberToken
}

// TestCreateMatch_Integration_AdminOnly covers the authorization change on
// POST /matches: a plain member of the group is now rejected with 403 while an
// admin of the same group still gets 200 — and the rejected request really
// creates nothing.
func TestCreateMatch_Integration_AdminOnly(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newMatchAdminEnv(t, tx)

	groupID, adminToken, memberToken := env.matchAdminGroup(t, tx,
		"Zzz Match Create", "match-create-admin@example.com", "match-create-member@example.com")

	payload := map[string]string{"date": "2026-01-04", "group_id": groupID.String()}

	memberRec := env.do(http.MethodPost, "/matches", memberToken, payload)
	if memberRec.Code != http.StatusForbidden {
		t.Fatalf("plain member POST /matches returned status %d, want 403, body: %s", memberRec.Code, memberRec.Body.String())
	}

	before, err := env.matches.GetMatchesDetails(groupID, "")
	if err != nil {
		t.Fatalf("GetMatchesDetails returned error: %v", err)
	}
	if len(before) != 0 {
		t.Fatalf("a forbidden POST /matches still created %d match(es)", len(before))
	}

	adminRec := env.do(http.MethodPost, "/matches", adminToken, payload)
	if adminRec.Code != http.StatusOK {
		t.Fatalf("admin POST /matches returned status %d, want 200, body: %s", adminRec.Code, adminRec.Body.String())
	}

	after, err := env.matches.GetMatchesDetails(groupID, "")
	if err != nil {
		t.Fatalf("GetMatchesDetails returned error: %v", err)
	}
	if len(after) != 1 {
		t.Errorf("after an admin POST /matches the group has %d match(es), want 1", len(after))
	}
}

// TestUpdateMatch_Integration_AdminOnly covers the same change on
// PUT /matches/:id — editing scores is admin-only — while reading the very
// same match through GET /matches/details stays open to a plain member.
func TestUpdateMatch_Integration_AdminOnly(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newMatchAdminEnv(t, tx)

	groupID, adminToken, memberToken := env.matchAdminGroup(t, tx,
		"Zzz Match Update", "match-update-admin@example.com", "match-update-member@example.com")

	scorerID, err := env.players.CreatePlayer("Zzz Match Update Scorer")
	if err != nil {
		t.Fatalf("failed to create scorer player: %v", err)
	}
	if err := env.memberships.AddPlayerToGroup(groupID, scorerID); err != nil {
		t.Fatalf("failed to add scorer to group: %v", err)
	}

	matchID, err := env.matches.CreateMatch(services.MatchSpec{Date: models.Date{}}, groupID)
	if err != nil {
		t.Fatalf("failed to create match: %v", err)
	}
	teams, err := env.teams.GetTeamsByGroupID(groupID)
	if err != nil {
		t.Fatalf("failed to load teams: %v", err)
	}
	if len(teams) == 0 {
		t.Fatal("group has no teams to assign players to")
	}

	payload := models.MatchWithDetails{
		ID:      matchID,
		GroupID: groupID,
		Teams: []models.TeamWithPlayers{{
			ID:      teams[0].ID,
			Colour:  teams[0].Colour,
			Players: []models.PlayerCustom{{ID: scorerID, Name: "Zzz Match Update Scorer", GoalsScored: 3}},
		}},
	}

	memberRec := env.do(http.MethodPut, "/matches/"+matchID.String(), memberToken, payload)
	if memberRec.Code != http.StatusForbidden {
		t.Fatalf("plain member PUT /matches/:id returned status %d, want 403, body: %s", memberRec.Code, memberRec.Body.String())
	}

	var rows int64
	if err := tx.Model(&models.MatchPlayer{}).Where("match_id = ?", matchID).Count(&rows).Error; err != nil {
		t.Fatalf("counting match_players returned error: %v", err)
	}
	if rows != 0 {
		t.Fatalf("a forbidden PUT /matches/:id still wrote %d match_player row(s)", rows)
	}

	// The same member can still *read* the match: only writing became
	// admin-only.
	readRec := env.do(http.MethodGet, "/matches/details?group_id="+groupID.String(), memberToken, nil)
	if readRec.Code != http.StatusOK {
		t.Errorf("plain member GET /matches/details returned status %d, want 200, body: %s", readRec.Code, readRec.Body.String())
	}

	adminRec := env.do(http.MethodPut, "/matches/"+matchID.String(), adminToken, payload)
	if adminRec.Code != http.StatusOK {
		t.Fatalf("admin PUT /matches/:id returned status %d, want 200, body: %s", adminRec.Code, adminRec.Body.String())
	}

	var stored models.MatchPlayer
	if err := tx.Where("match_id = ? AND player_id = ?", matchID, scorerID).First(&stored).Error; err != nil {
		t.Fatalf("expected a match_player row after the admin update: %v", err)
	}
	if stored.GoalsScored != 3 {
		t.Errorf("stored goals = %d, want 3", stored.GoalsScored)
	}
}

// TestCreateMatch_Integration_OutsiderStillForbidden guards the case the admin
// gating must not weaken: someone who isn't in the group at all is rejected
// too, and with the same 403 as a plain member — the middleware doesn't
// distinguish "member but not admin" from "not a member", so it can't be used
// to probe group membership.
func TestCreateMatch_Integration_OutsiderStillForbidden(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newMatchAdminEnv(t, tx)

	groupID, _, _ := env.matchAdminGroup(t, tx,
		"Zzz Match Outsider", "match-outsider-admin@example.com", "match-outsider-member@example.com")

	_, outsiderToken := env.authenticatedPlayer(t, "Zzz Match Outsider Player", "match-outsider@example.com")

	rec := env.do(http.MethodPost, "/matches", outsiderToken,
		map[string]string{"date": "2026-01-04", "group_id": groupID.String()})
	if rec.Code != http.StatusForbidden {
		t.Errorf("outsider POST /matches returned status %d, want 403, body: %s", rec.Code, rec.Body.String())
	}
}

// TestDeleteMatch_Integration_AdminOnly exercises the full authorization
// matrix for DELETE /matches/:id: no token -> 401, a plain member of the
// group -> 403, a match belonging to a different group even for an admin of
// *some* group -> 404, and finally an admin of the right group -> 200,
// followed by a GET .../details on the same id confirming the delete really
// happened rather than just returning a happy status code.
func TestDeleteMatch_Integration_AdminOnly(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newMatchAdminEnv(t, tx)

	groupID, adminToken, memberToken := env.matchAdminGroup(t, tx,
		"Zzz Match Delete", "match-delete-admin@example.com", "match-delete-member@example.com")

	matchID, err := env.matches.CreateMatch(services.MatchSpec{Date: models.Date{}}, groupID)
	if err != nil {
		t.Fatalf("failed to create match: %v", err)
	}

	// No token at all.
	noAuthRec := env.do(http.MethodDelete, "/matches/"+matchID.String(), "", nil)
	if noAuthRec.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated DELETE /matches/:id returned status %d, want 401, body: %s", noAuthRec.Code, noAuthRec.Body.String())
	}

	// A plain member of the group cannot delete.
	memberRec := env.do(http.MethodDelete, "/matches/"+matchID.String()+"?group_id="+groupID.String(), memberToken, nil)
	if memberRec.Code != http.StatusForbidden {
		t.Fatalf("plain member DELETE /matches/:id returned status %d, want 403, body: %s", memberRec.Code, memberRec.Body.String())
	}

	// An admin of a *different* group must not be able to reach this match:
	// the match id is scoped to groupID, not the admin's own group, so this
	// is a 404 rather than a 403 — mirroring GetMatchDetailsByID's behaviour.
	otherGroupID, otherAdminToken, _ := env.matchAdminGroup(t, tx,
		"Zzz Match Delete Other", "match-delete-other-admin@example.com", "match-delete-other-member@example.com")
	wrongGroupRec := env.do(http.MethodDelete, "/matches/"+matchID.String()+"?group_id="+otherGroupID.String(), otherAdminToken, nil)
	if wrongGroupRec.Code != http.StatusNotFound {
		t.Fatalf("admin of a different group DELETE /matches/:id returned status %d, want 404, body: %s", wrongGroupRec.Code, wrongGroupRec.Body.String())
	}

	// The admin of the right group can delete it.
	adminRec := env.do(http.MethodDelete, "/matches/"+matchID.String()+"?group_id="+groupID.String(), adminToken, nil)
	if adminRec.Code != http.StatusOK {
		t.Fatalf("admin DELETE /matches/:id returned status %d, want 200, body: %s", adminRec.Code, adminRec.Body.String())
	}

	var body map[string]bool
	if err := json.Unmarshal(adminRec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal delete response: %v", err)
	}
	if !body["deleted"] {
		t.Errorf("delete response = %v, want {\"deleted\": true}", body)
	}

	// Proof the delete actually happened, not just that the endpoint returned 200.
	getRec := env.do(http.MethodGet, "/matches/"+matchID.String()+"/details?group_id="+groupID.String(), adminToken, nil)
	if getRec.Code != http.StatusNotFound {
		t.Errorf("GET .../details after delete returned status %d, want 404, body: %s", getRec.Code, getRec.Body.String())
	}
}

// TestUpdateMatch_Integration_CannotForgeAnotherGroupsMatch is the regression
// test for the cross-tenant hole PUT /matches/:id used to have.
//
// The route used to be gated by requireGroupAdmin, which resolves the group
// from the request *body* — so an admin of group A could send GroupID = A (a
// group they legitimately administer, satisfying the admin check) together
// with ID = a match belonging to group B, and MatchService.UpdateMatch, which
// read the match id off that same body, rewrote group B's roster and scores.
// Group B's real members then saw the forged result.
//
// The body still carries both fields here — a real attacker's would — so what
// this pins is that they are now ignored entirely: the group comes from the
// match named in the path, group A's admin is not a member of that group, and
// the answer is the same 404 an unknown match id gets (never 403, which would
// confirm the match exists — see matchscope.go).
func TestUpdateMatch_Integration_CannotForgeAnotherGroupsMatch(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newMatchAdminEnv(t, tx)

	groupAID, adminAToken, _ := env.matchAdminGroup(t, tx,
		"Zzz Match Forge A", "match-forge-a-admin@example.com", "match-forge-a-member@example.com")
	groupBID, _, _ := env.matchAdminGroup(t, tx,
		"Zzz Match Forge B", "match-forge-b-admin@example.com", "match-forge-b-member@example.com")

	teamsA, err := env.teams.GetTeamsByGroupID(groupAID)
	if err != nil {
		t.Fatalf("failed to load group A's teams: %v", err)
	}
	teamsB, err := env.teams.GetTeamsByGroupID(groupBID)
	if err != nil {
		t.Fatalf("failed to load group B's teams: %v", err)
	}

	// Group B's own match, with a roster already recorded: the forged request
	// must leave it byte-for-byte as it is.
	victimID, err := env.players.CreatePlayer("Zzz Match Forge Victim")
	if err != nil {
		t.Fatalf("failed to create group B's player: %v", err)
	}
	if err := env.memberships.AddPlayerToGroup(groupBID, victimID); err != nil {
		t.Fatalf("failed to add group B's player: %v", err)
	}
	matchBID, err := env.matches.CreateMatch(services.MatchSpec{Date: models.Date{}}, groupBID)
	if err != nil {
		t.Fatalf("failed to create group B's match: %v", err)
	}
	if err := env.matches.UpdateMatch(matchBID, groupBID, []models.TeamWithPlayers{{
		ID:      teamsB[0].ID,
		Players: []models.PlayerCustom{{ID: victimID, GoalsScored: 1}},
	}}); err != nil {
		t.Fatalf("failed to compose group B's roster: %v", err)
	}

	// The forged payload: group A's own group id and one of group A's own
	// teams, aimed at group B's match.
	intruderID, err := env.players.CreatePlayer("Zzz Match Forge Intruder")
	if err != nil {
		t.Fatalf("failed to create group A's player: %v", err)
	}
	if err := env.memberships.AddPlayerToGroup(groupAID, intruderID); err != nil {
		t.Fatalf("failed to add group A's player: %v", err)
	}
	forged := models.MatchWithDetails{
		ID:      matchBID,
		GroupID: groupAID,
		Teams: []models.TeamWithPlayers{{
			ID:      teamsA[0].ID,
			Players: []models.PlayerCustom{{ID: intruderID, GoalsScored: 99}},
		}},
	}

	rec := env.do(http.MethodPut, "/matches/"+matchBID.String(), adminAToken, forged)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("admin of group A PUT on group B's match returned status %d, want 404, body: %s", rec.Code, rec.Body.String())
	}

	// Nothing of group B's match was touched: still exactly the one row it
	// had, still one goal, and no row for group A's team or player.
	var rows []models.MatchPlayer
	if err := tx.Where("match_id = ?", matchBID).Find(&rows).Error; err != nil {
		t.Fatalf("loading group B's match_players returned error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("group B's match has %d match_player row(s) after the forged PUT, want 1: %+v", len(rows), rows)
	}
	if rows[0].PlayerID != victimID || rows[0].TeamID != teamsB[0].ID || rows[0].GoalsScored != 1 {
		t.Errorf("group B's match_player row = %+v, want player %s on team %s with 1 goal", rows[0], victimID, teamsB[0].ID)
	}
}
