package handlers_test

import (
	"bytes"
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

const testUpdateRoleJWTSecret = "zzz-integration-test-update-member-role-secret"

// roleEnv bundles what a PATCH /groups/:id/members/:playerId/role test needs,
// all bound to the same rolled-back transaction, with main.go's real
// middleware chain in front of the handler.
type roleEnv struct {
	memberships *services.GroupMembershipService
	players     *services.PlayerService
	auth        *services.AuthService
	router      *gin.Engine
}

func newRoleEnv(t *testing.T, tx *gorm.DB) *roleEnv {
	t.Helper()

	groupService := services.NewGroupService(tx)
	membershipService := services.NewGroupMembershipService(tx)
	authService := services.NewAuthService(tx, testUpdateRoleJWTSecret)
	groupHandler := handlers.NewGroupHandler(groupService, membershipService, authService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.PATCH("/groups/:id/members/:playerId/role",
		handlers.AuthMiddleware(authService),
		handlers.RequireGroupAdminByPathParam(membershipService, "id"),
		groupHandler.UpdateMemberRole)

	return &roleEnv{
		memberships: membershipService,
		players:     services.NewPlayerService(tx),
		auth:        authService,
		router:      router,
	}
}

func (e *roleEnv) authenticatedPlayer(t *testing.T, name, email string) (uuid.UUID, string) {
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

func (e *roleEnv) patchRole(groupID, targetID uuid.UUID, token, role string) *httptest.ResponseRecorder {
	body := []byte(`{"role":"` + role + `"}`)
	req := httptest.NewRequest(http.MethodPatch,
		"/groups/"+groupID.String()+"/members/"+targetID.String()+"/role",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// newRoleGroup creates a group with the given admin and one plain member.
func (e *roleEnv) newRoleGroup(t *testing.T, tx *gorm.DB, name string, adminID uuid.UUID) uuid.UUID {
	t.Helper()
	group, err := services.NewGroupService(tx).CreateGroup(name, services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	if err := e.memberships.AddPlayerToGroupWithRole(group.ID, adminID, models.RoleAdmin); err != nil {
		t.Fatalf("failed to add admin: %v", err)
	}
	return group.ID
}

// TestUpdateMemberRole_Integration_Promote covers the happy path an admin
// needs for a group to gain a second admin at all: PATCH with role "admin".
func TestUpdateMemberRole_Integration_Promote(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newRoleEnv(t, tx)

	adminID, adminToken := env.authenticatedPlayer(t, "Zzz Role HTTP Promote Admin", "role-http-promote-admin@example.com")
	groupID := env.newRoleGroup(t, tx, "Zzz Role HTTP Promote Group", adminID)
	targetID, _ := env.authenticatedPlayer(t, "Zzz Role HTTP Promote Target", "role-http-promote-target@example.com")
	if err := env.memberships.AddPlayerToGroup(groupID, targetID); err != nil {
		t.Fatalf("failed to add target member: %v", err)
	}

	rec := env.patchRole(groupID, targetID, adminToken, models.RoleAdmin)
	if rec.Code != http.StatusOK {
		t.Fatalf("promote returned status %d, want 200, body: %s", rec.Code, rec.Body.String())
	}

	role, err := env.memberships.GetRole(groupID, targetID)
	if err != nil {
		t.Fatalf("GetRole(target) returned error: %v", err)
	}
	if role != models.RoleAdmin {
		t.Errorf("target role = %q after promotion, want %q", role, models.RoleAdmin)
	}
}

// TestUpdateMemberRole_Integration_Demote covers the reverse: a second admin
// being put back to plain member by the first one.
func TestUpdateMemberRole_Integration_Demote(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newRoleEnv(t, tx)

	adminID, adminToken := env.authenticatedPlayer(t, "Zzz Role HTTP Demote Admin", "role-http-demote-admin@example.com")
	groupID := env.newRoleGroup(t, tx, "Zzz Role HTTP Demote Group", adminID)
	targetID, _ := env.authenticatedPlayer(t, "Zzz Role HTTP Demote Target", "role-http-demote-target@example.com")
	if err := env.memberships.AddPlayerToGroupWithRole(groupID, targetID, models.RoleAdmin); err != nil {
		t.Fatalf("failed to add second admin: %v", err)
	}

	rec := env.patchRole(groupID, targetID, adminToken, models.RoleMember)
	if rec.Code != http.StatusOK {
		t.Fatalf("demote returned status %d, want 200, body: %s", rec.Code, rec.Body.String())
	}

	role, err := env.memberships.GetRole(groupID, targetID)
	if err != nil {
		t.Fatalf("GetRole(target) returned error: %v", err)
	}
	if role != models.RoleMember {
		t.Errorf("target role = %q after demotion, want %q", role, models.RoleMember)
	}
}

// TestUpdateMemberRole_Integration_NonAdminForbidden covers a plain member of
// the group trying to promote someone: RequireGroupAdminByPathParam must
// reject with 403 before the handler ever runs, so a member can't grant
// themselves — or a friend — admin powers.
func TestUpdateMemberRole_Integration_NonAdminForbidden(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newRoleEnv(t, tx)

	adminID, _ := env.authenticatedPlayer(t, "Zzz Role HTTP Forbidden Admin", "role-http-forbidden-admin@example.com")
	groupID := env.newRoleGroup(t, tx, "Zzz Role HTTP Forbidden Group", adminID)

	memberID, memberToken := env.authenticatedPlayer(t, "Zzz Role HTTP Forbidden Member", "role-http-forbidden-member@example.com")
	if err := env.memberships.AddPlayerToGroup(groupID, memberID); err != nil {
		t.Fatalf("failed to add member: %v", err)
	}
	targetID, _ := env.authenticatedPlayer(t, "Zzz Role HTTP Forbidden Target", "role-http-forbidden-target@example.com")
	if err := env.memberships.AddPlayerToGroup(groupID, targetID); err != nil {
		t.Fatalf("failed to add target member: %v", err)
	}

	rec := env.patchRole(groupID, targetID, memberToken, models.RoleAdmin)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-admin promote returned status %d, want 403, body: %s", rec.Code, rec.Body.String())
	}

	role, err := env.memberships.GetRole(groupID, targetID)
	if err != nil {
		t.Fatalf("GetRole(target) returned error: %v", err)
	}
	if role != models.RoleMember {
		t.Errorf("target role = %q after a forbidden request, want unchanged %q", role, models.RoleMember)
	}
}

// TestUpdateMemberRole_Integration_SelfTargetRejected covers
// ErrCannotChangeOwnRole surfacing as a 400: there is no self-service step
// down through this route.
func TestUpdateMemberRole_Integration_SelfTargetRejected(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newRoleEnv(t, tx)

	adminID, adminToken := env.authenticatedPlayer(t, "Zzz Role HTTP Self Admin", "role-http-self-admin@example.com")
	groupID := env.newRoleGroup(t, tx, "Zzz Role HTTP Self Group", adminID)

	rec := env.patchRole(groupID, adminID, adminToken, models.RoleMember)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("self-targeting returned status %d, want 400, body: %s", rec.Code, rec.Body.String())
	}

	role, err := env.memberships.GetRole(groupID, adminID)
	if err != nil {
		t.Fatalf("GetRole(admin) returned error: %v", err)
	}
	if role != models.RoleAdmin {
		t.Errorf("admin role = %q after a refused self-update, want unchanged %q", role, models.RoleAdmin)
	}
}

// TestUpdateMemberRole_Integration_TargetNotMember covers a target who isn't
// in the group: gorm.ErrRecordNotFound must surface as a 404, not a 500.
func TestUpdateMemberRole_Integration_TargetNotMember(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newRoleEnv(t, tx)

	adminID, adminToken := env.authenticatedPlayer(t, "Zzz Role HTTP NotMember Admin", "role-http-notmember-admin@example.com")
	groupID := env.newRoleGroup(t, tx, "Zzz Role HTTP NotMember Group", adminID)
	outsiderID, _ := env.authenticatedPlayer(t, "Zzz Role HTTP NotMember Outsider", "role-http-notmember-outsider@example.com")

	rec := env.patchRole(groupID, outsiderID, adminToken, models.RoleAdmin)
	if rec.Code != http.StatusNotFound {
		t.Errorf("non-member target returned status %d, want 404, body: %s", rec.Code, rec.Body.String())
	}
}

// TestUpdateMemberRole_Integration_InvalidRole covers a role value the model
// doesn't allow: ErrInvalidRole must surface as a 400.
func TestUpdateMemberRole_Integration_InvalidRole(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newRoleEnv(t, tx)

	adminID, adminToken := env.authenticatedPlayer(t, "Zzz Role HTTP Invalid Admin", "role-http-invalid-admin@example.com")
	groupID := env.newRoleGroup(t, tx, "Zzz Role HTTP Invalid Group", adminID)
	targetID, _ := env.authenticatedPlayer(t, "Zzz Role HTTP Invalid Target", "role-http-invalid-target@example.com")
	if err := env.memberships.AddPlayerToGroup(groupID, targetID); err != nil {
		t.Fatalf("failed to add target member: %v", err)
	}

	rec := env.patchRole(groupID, targetID, adminToken, "superadmin")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid role returned status %d, want 400, body: %s", rec.Code, rec.Body.String())
	}

	role, err := env.memberships.GetRole(groupID, targetID)
	if err != nil {
		t.Fatalf("GetRole(target) returned error: %v", err)
	}
	if role != models.RoleMember {
		t.Errorf("target role = %q after a refused update, want unchanged %q", role, models.RoleMember)
	}
}

// TestUpdateMemberRole_Integration_NoToken covers the unauthenticated case:
// AuthMiddleware rejects before the admin middleware or the handler run.
func TestUpdateMemberRole_Integration_NoToken(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newRoleEnv(t, tx)

	adminID, _ := env.authenticatedPlayer(t, "Zzz Role HTTP NoToken Admin", "role-http-notoken-admin@example.com")
	groupID := env.newRoleGroup(t, tx, "Zzz Role HTTP NoToken Group", adminID)
	targetID, _ := env.authenticatedPlayer(t, "Zzz Role HTTP NoToken Target", "role-http-notoken-target@example.com")
	if err := env.memberships.AddPlayerToGroup(groupID, targetID); err != nil {
		t.Fatalf("failed to add target member: %v", err)
	}

	rec := env.patchRole(groupID, targetID, "", models.RoleAdmin)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("no-token request returned status %d, want 401, body: %s", rec.Code, rec.Body.String())
	}
}
