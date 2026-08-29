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

const testUpdateTeamJWTSecret = "zzz-integration-test-update-team-secret"

// updateTeamEnv bundles what a PATCH /groups/:id/teams/:teamId test needs,
// all bound to the same rolled-back transaction, with main.go's real
// middleware chain (RequireGroupAdminByPathParam) in front of the handler.
type updateTeamEnv struct {
	groups      *services.GroupService
	memberships *services.GroupMembershipService
	teams       *services.TeamService
	players     *services.PlayerService
	auth        *services.AuthService
	router      *gin.Engine
}

func newUpdateTeamEnv(t *testing.T, tx *gorm.DB) *updateTeamEnv {
	t.Helper()

	groupService := services.NewGroupService(tx)
	membershipService := services.NewGroupMembershipService(tx)
	teamService := services.NewTeamService(tx)
	authService := services.NewAuthService(tx, testUpdateTeamJWTSecret)
	teamHandler := handlers.NewTeamHandler(teamService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.PATCH("/groups/:id/teams/:teamId",
		handlers.AuthMiddleware(authService),
		handlers.RequireGroupAdminByPathParam(membershipService, "id"),
		teamHandler.UpdateTeam)

	return &updateTeamEnv{
		groups:      groupService,
		memberships: membershipService,
		teams:       teamService,
		players:     services.NewPlayerService(tx),
		auth:        authService,
		router:      router,
	}
}

func (e *updateTeamEnv) authenticatedPlayer(t *testing.T, name, email string) (uuid.UUID, string) {
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

// newGroupWithAdmin creates a group (with the default black/white teams) and
// makes adminID its admin, returning the group id and its two team ids.
func (e *updateTeamEnv) newGroupWithAdmin(t *testing.T, name string, adminID uuid.UUID) (uuid.UUID, []uuid.UUID) {
	t.Helper()
	group, err := e.groups.CreateGroup(name, services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	if err := e.memberships.AddPlayerToGroupWithRole(group.ID, adminID, models.RoleAdmin); err != nil {
		t.Fatalf("failed to add admin: %v", err)
	}
	teams, err := e.teams.GetTeamsByGroupID(group.ID)
	if err != nil || len(teams) != 2 {
		t.Fatalf("failed to load group teams: err=%v teams=%+v", err, teams)
	}
	return group.ID, []uuid.UUID{teams[0].ID, teams[1].ID}
}

func (e *updateTeamEnv) patchTeam(groupID, teamID uuid.UUID, token, name, colour string) *httptest.ResponseRecorder {
	raw, _ := json.Marshal(map[string]string{"name": name, "colour": colour})
	req := httptest.NewRequest(http.MethodPatch,
		"/groups/"+groupID.String()+"/teams/"+teamID.String(),
		bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// TestUpdateTeam_Integration_AdminSuccess covers the happy path: an admin of
// the team's own group renames it and changes its colour, verified both via
// the response body and a follow-up GetTeamsByGroupID (not just the status
// code).
func TestUpdateTeam_Integration_AdminSuccess(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newUpdateTeamEnv(t, tx)

	adminID, adminToken := env.authenticatedPlayer(t, "Zzz Team HTTP Admin", "team-http-admin@example.com")
	groupID, teamIDs := env.newGroupWithAdmin(t, "Zzz Team HTTP Group", adminID)

	rec := env.patchTeam(groupID, teamIDs[0], adminToken, "Les Rouges", "red")
	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH team returned status %d, want 200, body: %s", rec.Code, rec.Body.String())
	}

	var updated models.Team
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("failed to decode updated team: %v", err)
	}
	if updated.Name != "Les Rouges" || updated.Colour != "red" {
		t.Errorf("response team = %+v, want Name=%q Colour=%q", updated, "Les Rouges", "red")
	}

	teams, err := env.teams.GetTeamsByGroupID(groupID)
	if err != nil {
		t.Fatalf("GetTeamsByGroupID returned error: %v", err)
	}
	found := false
	for _, team := range teams {
		if team.ID == teamIDs[0] {
			found = true
			if team.Name != "Les Rouges" || team.Colour != "red" {
				t.Errorf("stored team = %+v, want Name=%q Colour=%q", team, "Les Rouges", "red")
			}
		}
	}
	if !found {
		t.Fatalf("updated team %s missing from group's teams: %+v", teamIDs[0], teams)
	}
}

// TestUpdateTeam_Integration_NoToken covers the unauthenticated case:
// AuthMiddleware rejects before the admin middleware or the handler run.
func TestUpdateTeam_Integration_NoToken(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newUpdateTeamEnv(t, tx)

	adminID, _ := env.authenticatedPlayer(t, "Zzz Team HTTP NoToken Admin", "team-http-notoken-admin@example.com")
	groupID, teamIDs := env.newGroupWithAdmin(t, "Zzz Team HTTP NoToken Group", adminID)

	rec := env.patchTeam(groupID, teamIDs[0], "", "New Name", "red")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("no-token request returned status %d, want 401, body: %s", rec.Code, rec.Body.String())
	}
}

// TestUpdateTeam_Integration_NonAdminForbidden covers a plain member of the
// group trying to rename a team: RequireGroupAdminByPathParam must reject
// with 403 before the handler ever runs.
func TestUpdateTeam_Integration_NonAdminForbidden(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newUpdateTeamEnv(t, tx)

	adminID, _ := env.authenticatedPlayer(t, "Zzz Team HTTP Forbidden Admin", "team-http-forbidden-admin@example.com")
	groupID, teamIDs := env.newGroupWithAdmin(t, "Zzz Team HTTP Forbidden Group", adminID)

	memberID, memberToken := env.authenticatedPlayer(t, "Zzz Team HTTP Forbidden Member", "team-http-forbidden-member@example.com")
	if err := env.memberships.AddPlayerToGroup(groupID, memberID); err != nil {
		t.Fatalf("failed to add member: %v", err)
	}

	rec := env.patchTeam(groupID, teamIDs[0], memberToken, "New Name", "red")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-admin PATCH returned status %d, want 403, body: %s", rec.Code, rec.Body.String())
	}

	teams, err := env.teams.GetTeamsByGroupID(groupID)
	if err != nil {
		t.Fatalf("GetTeamsByGroupID returned error: %v", err)
	}
	for _, team := range teams {
		if team.ID == teamIDs[0] && team.Name == "New Name" {
			t.Errorf("forbidden request still mutated the team: %+v", team)
		}
	}
}

// TestUpdateTeam_Integration_CrossGroupTeamNotFound is the important
// security case: an admin of *some* group (group A) must not be able to
// rename/recolour a team belonging to a different group (group B) just by
// knowing or guessing its UUID. RequireGroupAdminByPathParam only proves the
// caller administers the :id in the path — it says nothing about whether
// :teamId actually belongs to that group — so this has to be enforced by
// TeamService.UpdateTeam's own scoped lookup, surfaced here as a 404.
func TestUpdateTeam_Integration_CrossGroupTeamNotFound(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newUpdateTeamEnv(t, tx)

	adminAID, adminAToken := env.authenticatedPlayer(t, "Zzz Team HTTP CrossGroup Admin A", "team-http-crossgroup-admin-a@example.com")
	groupAID, _ := env.newGroupWithAdmin(t, "Zzz Team HTTP CrossGroup Group A", adminAID)

	adminBID, _ := env.authenticatedPlayer(t, "Zzz Team HTTP CrossGroup Admin B", "team-http-crossgroup-admin-b@example.com")
	_, teamBIDs := env.newGroupWithAdmin(t, "Zzz Team HTTP CrossGroup Group B", adminBID)

	// Admin of group A, targeting a team that belongs to group B via the URL.
	rec := env.patchTeam(groupAID, teamBIDs[0], adminAToken, "Hijacked", "purple")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("cross-group PATCH returned status %d, want 404, body: %s", rec.Code, rec.Body.String())
	}

	teamB, err := env.teams.GetTeamByID(teamBIDs[0])
	if err != nil {
		t.Fatalf("GetTeamByID(teamB) returned error: %v", err)
	}
	if teamB.Name == "Hijacked" || teamB.Colour == "purple" {
		t.Errorf("cross-group PATCH mutated group B's team: %+v", teamB)
	}
}

// TestUpdateTeam_Integration_RequiresNameAndColour covers the validation
// sentinels surfacing as 400s through the handler.
func TestUpdateTeam_Integration_RequiresNameAndColour(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newUpdateTeamEnv(t, tx)

	adminID, adminToken := env.authenticatedPlayer(t, "Zzz Team HTTP Validation Admin", "team-http-validation-admin@example.com")
	groupID, teamIDs := env.newGroupWithAdmin(t, "Zzz Team HTTP Validation Group", adminID)

	if rec := env.patchTeam(groupID, teamIDs[0], adminToken, "", "red"); rec.Code != http.StatusBadRequest {
		t.Errorf("empty name returned status %d, want 400, body: %s", rec.Code, rec.Body.String())
	}
	if rec := env.patchTeam(groupID, teamIDs[0], adminToken, "New Name", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("empty colour returned status %d, want 400, body: %s", rec.Code, rec.Body.String())
	}
}
