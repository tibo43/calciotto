package handlers_test

import (
	"bytes"
	"log"
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

const testInviteJWTSecret = "zzz-integration-test-invite-player-secret"

// inviteEnv bundles what a POST /groups/:id/members/:playerId/invite test
// needs, all bound to the same rolled-back transaction, with main.go's real
// middleware chain in front of the handler. It also mounts
// /auth/reset-password and /auth/login so a test can follow an invite all the
// way to a working account — the claim path is the whole point of the route,
// and a 200 alone proves nothing about it.
type inviteEnv struct {
	memberships *services.GroupMembershipService
	players     *services.PlayerService
	auth        *services.AuthService
	router      *gin.Engine
}

func newInviteEnv(t *testing.T, tx *gorm.DB) *inviteEnv {
	t.Helper()

	groupService := services.NewGroupService(tx)
	membershipService := services.NewGroupMembershipService(tx)
	authService := services.NewAuthService(tx, testInviteJWTSecret)
	groupHandler := handlers.NewGroupHandler(groupService, membershipService, authService)
	authHandler := handlers.NewAuthHandler(authService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/groups/:id/members/:playerId/invite",
		handlers.AuthMiddleware(authService),
		handlers.RequireGroupAdminByPathParam(membershipService, "id"),
		groupHandler.InvitePlayer)
	router.POST("/auth/reset-password", authHandler.ResetPassword)
	router.POST("/auth/login", authHandler.Login)

	return &inviteEnv{
		memberships: membershipService,
		players:     services.NewPlayerService(tx),
		auth:        authService,
		router:      router,
	}
}

func (e *inviteEnv) authenticatedPlayer(t *testing.T, name, email string) (uuid.UUID, string) {
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

// ghostMember creates a credential-less player and puts them on the group's
// roster — exactly what POST /players produces, and the only shape this route
// accepts.
func (e *inviteEnv) ghostMember(t *testing.T, groupID uuid.UUID, name string) uuid.UUID {
	t.Helper()
	id, err := e.players.CreatePlayer(name)
	if err != nil {
		t.Fatalf("failed to create ghost player %q: %v", name, err)
	}
	if err := e.memberships.AddPlayerToGroup(groupID, id); err != nil {
		t.Fatalf("failed to add ghost player %q to the group: %v", name, err)
	}
	return id
}

func (e *inviteEnv) newInviteGroup(t *testing.T, tx *gorm.DB, name string, adminID uuid.UUID) uuid.UUID {
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

func (e *inviteEnv) postInvite(groupID, targetID uuid.UUID, token, email string) *httptest.ResponseRecorder {
	body := []byte(`{"email":"` + email + `"}`)
	req := httptest.NewRequest(http.MethodPost,
		"/groups/"+groupID.String()+"/members/"+targetID.String()+"/invite",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// postInviteAndCaptureToken is postInvite plus the log-scraping the current
// no-mail-transport design forces: the claim link only ever exists in the
// server log (see AuthService.sendPasswordResetLink).
func (e *inviteEnv) postInviteAndCaptureToken(groupID, targetID uuid.UUID, token, email string) (*httptest.ResponseRecorder, string) {
	var logs bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&logs)
	rec := e.postInvite(groupID, targetID, token, email)
	log.SetOutput(previous)

	_, after, found := strings.Cut(logs.String(), "?token=")
	if !found {
		return rec, ""
	}
	claimToken, _, _ := strings.Cut(after, "\n")
	return rec, strings.TrimSpace(claimToken)
}

// TestInvitePlayer_Integration_AdminInvitesGhostMember follows the whole chain
// an admin triggers: invite → claim link → password set through the existing
// /auth/reset-password → login. A 200 on the invite alone would not prove the
// reuse of the password-reset machinery actually works end to end.
func TestInvitePlayer_Integration_AdminInvitesGhostMember(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newInviteEnv(t, tx)

	adminID, adminToken := env.authenticatedPlayer(t, "Zzz Invite HTTP Admin", "invite-http-admin@example.com")
	groupID := env.newInviteGroup(t, tx, "Zzz Invite HTTP Group", adminID)
	ghostID := env.ghostMember(t, groupID, "Zzz Invite HTTP Ghost")

	rec, claimToken := env.postInviteAndCaptureToken(groupID, ghostID, adminToken, "invite-http-ghost@example.com")
	if rec.Code != http.StatusOK {
		t.Fatalf("invite returned status %d, want 200, body: %s", rec.Code, rec.Body.String())
	}
	if claimToken == "" {
		t.Fatal("a successful invite logged no claim link")
	}

	resetRec := postJSON(t, env.router, "/auth/reset-password", map[string]string{
		"token":        claimToken,
		"new_password": "ghost-chosen-pass",
	})
	if resetRec.Code != http.StatusOK {
		t.Fatalf("reset-password with the invite token returned status %d, body: %s", resetRec.Code, resetRec.Body.String())
	}

	loginRec := postJSON(t, env.router, "/auth/login", map[string]string{
		"email":    "invite-http-ghost@example.com",
		"password": "ghost-chosen-pass",
	})
	if loginRec.Code != http.StatusOK {
		t.Errorf("login as the newly claimed player returned status %d, want 200, body: %s", loginRec.Code, loginRec.Body.String())
	}
}

// TestInvitePlayer_Integration_NoToken covers the unauthenticated case:
// AuthMiddleware rejects before the admin middleware or the handler run.
func TestInvitePlayer_Integration_NoToken(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newInviteEnv(t, tx)

	adminID, _ := env.authenticatedPlayer(t, "Zzz Invite HTTP NoToken Admin", "invite-http-notoken-admin@example.com")
	groupID := env.newInviteGroup(t, tx, "Zzz Invite HTTP NoToken Group", adminID)
	ghostID := env.ghostMember(t, groupID, "Zzz Invite HTTP NoToken Ghost")

	rec := env.postInvite(groupID, ghostID, "", "invite-http-notoken-ghost@example.com")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("no-token invite returned status %d, want 401, body: %s", rec.Code, rec.Body.String())
	}
}

// TestInvitePlayer_Integration_NonAdminForbidden covers a plain member of the
// group trying to hand an account to someone: giving out credentials is an
// admin capability, so RequireGroupAdminByPathParam must reject with 403
// before the handler runs.
func TestInvitePlayer_Integration_NonAdminForbidden(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newInviteEnv(t, tx)

	adminID, _ := env.authenticatedPlayer(t, "Zzz Invite HTTP Forbidden Admin", "invite-http-forbidden-admin@example.com")
	groupID := env.newInviteGroup(t, tx, "Zzz Invite HTTP Forbidden Group", adminID)
	memberID, memberToken := env.authenticatedPlayer(t, "Zzz Invite HTTP Forbidden Member", "invite-http-forbidden-member@example.com")
	if err := env.memberships.AddPlayerToGroup(groupID, memberID); err != nil {
		t.Fatalf("failed to add member: %v", err)
	}
	ghostID := env.ghostMember(t, groupID, "Zzz Invite HTTP Forbidden Ghost")

	rec, claimToken := env.postInviteAndCaptureToken(groupID, ghostID, memberToken, "invite-http-forbidden-ghost@example.com")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-admin invite returned status %d, want 403, body: %s", rec.Code, rec.Body.String())
	}
	if claimToken != "" {
		t.Error("a forbidden invite still issued a claim link")
	}

	var ghost models.Player
	if err := tx.First(&ghost, "id = ?", ghostID).Error; err != nil {
		t.Fatalf("failed to reload the ghost player: %v", err)
	}
	if ghost.Email != nil {
		t.Errorf("ghost email = %q after a forbidden invite, want nil", *ghost.Email)
	}
}

// TestInvitePlayer_Integration_TargetNotMember is the case the admin
// middleware cannot cover: the caller really is an admin of :id, but :playerId
// belongs to another group entirely. Without the handler's own IsMember check
// this would hand that outsider an account.
func TestInvitePlayer_Integration_TargetNotMember(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newInviteEnv(t, tx)

	adminID, adminToken := env.authenticatedPlayer(t, "Zzz Invite HTTP Outsider Admin", "invite-http-outsider-admin@example.com")
	groupID := env.newInviteGroup(t, tx, "Zzz Invite HTTP Outsider Group A", adminID)

	otherAdminID, _ := env.authenticatedPlayer(t, "Zzz Invite HTTP Outsider OtherAdmin", "invite-http-outsider-other@example.com")
	otherGroupID := env.newInviteGroup(t, tx, "Zzz Invite HTTP Outsider Group B", otherAdminID)
	outsiderID := env.ghostMember(t, otherGroupID, "Zzz Invite HTTP Outsider Ghost")

	rec, claimToken := env.postInviteAndCaptureToken(groupID, outsiderID, adminToken, "invite-http-outsider-ghost@example.com")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("invite of a non-member returned status %d, want 404, body: %s", rec.Code, rec.Body.String())
	}
	if claimToken != "" {
		t.Error("an invite of a non-member still issued a claim link")
	}

	var outsider models.Player
	if err := tx.First(&outsider, "id = ?", outsiderID).Error; err != nil {
		t.Fatalf("failed to reload the outsider: %v", err)
	}
	if outsider.Email != nil {
		t.Errorf("outsider email = %q after a refused invite, want nil", *outsider.Email)
	}
}

// TestInvitePlayer_Integration_AlreadyClaimedReturns400 covers
// ErrPlayerAlreadyClaimed surfacing as a 400: a member who already has an
// account can't have their email rewritten — or be re-invited — through this
// route.
func TestInvitePlayer_Integration_AlreadyClaimedReturns400(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newInviteEnv(t, tx)

	adminID, adminToken := env.authenticatedPlayer(t, "Zzz Invite HTTP Claimed Admin", "invite-http-claimed-admin@example.com")
	groupID := env.newInviteGroup(t, tx, "Zzz Invite HTTP Claimed Group", adminID)
	targetID, _ := env.authenticatedPlayer(t, "Zzz Invite HTTP Claimed Target", "invite-http-claimed-target@example.com")
	if err := env.memberships.AddPlayerToGroup(groupID, targetID); err != nil {
		t.Fatalf("failed to add target member: %v", err)
	}

	rec, claimToken := env.postInviteAndCaptureToken(groupID, targetID, adminToken, "invite-http-claimed-hijack@example.com")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invite of an already claimed player returned status %d, want 400, body: %s", rec.Code, rec.Body.String())
	}
	if claimToken != "" {
		t.Error("an invite of an already claimed player still issued a claim link")
	}

	var target models.Player
	if err := tx.First(&target, "id = ?", targetID).Error; err != nil {
		t.Fatalf("failed to reload the target: %v", err)
	}
	if target.Email == nil || *target.Email != "invite-http-claimed-target@example.com" {
		t.Errorf("target email = %v after a refused invite, want unchanged", target.Email)
	}
}
