package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"app/internal/handlers"
	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/gin-gonic/gin"
)

const testLeaveGroupJWTSecret = "zzz-integration-test-leave-group-secret"

func newLeaveGroupTestRouter(authService *services.AuthService, membershipService *services.GroupMembershipService, groupHandler *handlers.GroupHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	requireGroupMemberByPathID := handlers.RequireGroupMembershipByPathParam(membershipService, "id")
	router.DELETE("/groups/:id/members/me",
		handlers.AuthMiddleware(authService),
		requireGroupMemberByPathID,
		groupHandler.LeaveGroup)

	// Mirrors GET /groups/:id/teams-style routes wired in main.go, used here
	// only to prove that after leaving, the same middleware rejects the
	// player with 403 on a subsequent request against the same group.
	router.GET("/groups/:id/probe",
		handlers.AuthMiddleware(authService),
		requireGroupMemberByPathID,
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

	return router
}

// TestLeaveGroup_Integration_HappyPath exercises DELETE /groups/:id/members/me
// end to end: a member with a valid token leaves successfully, and the
// membership is really gone afterward — both via IsMember and by re-hitting a
// RequireGroupMembershipByPathParam-protected route, which must now 403.
func TestLeaveGroup_Integration_HappyPath(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	membershipService := services.NewGroupMembershipService(tx)
	playerService := services.NewPlayerService(tx)
	authService := services.NewAuthService(tx, testLeaveGroupJWTSecret)
	groupHandler := handlers.NewGroupHandler(groupService, membershipService, authService)

	group, err := groupService.CreateGroup("Zzz Leave HTTP Group", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	adminID, err := playerService.CreatePlayer("Zzz Leave HTTP Admin")
	if err != nil {
		t.Fatalf("failed to create admin player: %v", err)
	}
	if err := membershipService.AddPlayerToGroupWithRole(group.ID, adminID, models.RoleAdmin); err != nil {
		t.Fatalf("failed to add admin: %v", err)
	}

	leavingID, err := playerService.CreatePlayer("Zzz Leave HTTP Leaving")
	if err != nil {
		t.Fatalf("failed to create leaving player: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(group.ID, leavingID); err != nil {
		t.Fatalf("failed to add leaving player: %v", err)
	}
	if err := authService.Signup(leavingID, "leave-http@example.com", "s3cret-pass"); err != nil {
		t.Fatalf("failed to sign up leaving player: %v", err)
	}
	leavingToken, err := authService.Login("leave-http@example.com", "s3cret-pass")
	if err != nil {
		t.Fatalf("failed to log in leaving player: %v", err)
	}

	router := newLeaveGroupTestRouter(authService, membershipService, groupHandler)

	req := httptest.NewRequest(http.MethodDelete, "/groups/"+group.ID.String()+"/members/me", nil)
	req.Header.Set("Authorization", "Bearer "+leavingToken)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("leave group returned status %d, want 200, body: %s", rec.Code, rec.Body.String())
	}

	isMember, err := membershipService.IsMember(group.ID, leavingID)
	if err != nil {
		t.Fatalf("IsMember returned error: %v", err)
	}
	if isMember {
		t.Error("player is still a member after a successful leave")
	}

	probeReq := httptest.NewRequest(http.MethodGet, "/groups/"+group.ID.String()+"/probe", nil)
	probeReq.Header.Set("Authorization", "Bearer "+leavingToken)
	probeRec := httptest.NewRecorder()
	router.ServeHTTP(probeRec, probeReq)
	if probeRec.Code != http.StatusForbidden {
		t.Errorf("group-scoped route after leaving returned status %d, want 403, body: %s", probeRec.Code, probeRec.Body.String())
	}
}

// TestLeaveGroup_Integration_NoToken covers the unauthenticated case: no
// Authorization header at all must be rejected by AuthMiddleware before the
// handler or the membership middleware ever runs.
func TestLeaveGroup_Integration_NoToken(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	membershipService := services.NewGroupMembershipService(tx)
	authService := services.NewAuthService(tx, testLeaveGroupJWTSecret)
	groupHandler := handlers.NewGroupHandler(groupService, membershipService, authService)

	group, err := groupService.CreateGroup("Zzz Leave HTTP No Token Group", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	router := newLeaveGroupTestRouter(authService, membershipService, groupHandler)

	req := httptest.NewRequest(http.MethodDelete, "/groups/"+group.ID.String()+"/members/me", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("no-token leave request returned status %d, want 401, body: %s", rec.Code, rec.Body.String())
	}
}

// TestLeaveGroup_Integration_NonMemberForbidden covers a valid token for a
// player who simply isn't a member of the target group: the whole chain
// (RequireGroupMembershipByPathParam, ahead of GroupHandler.LeaveGroup) must
// reject with 403 without ever reaching the leave logic.
func TestLeaveGroup_Integration_NonMemberForbidden(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	membershipService := services.NewGroupMembershipService(tx)
	playerService := services.NewPlayerService(tx)
	authService := services.NewAuthService(tx, testLeaveGroupJWTSecret)
	groupHandler := handlers.NewGroupHandler(groupService, membershipService, authService)

	group, err := groupService.CreateGroup("Zzz Leave HTTP Outsider Group", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	outsiderID, err := playerService.CreatePlayer("Zzz Leave HTTP Outsider")
	if err != nil {
		t.Fatalf("failed to create outsider player: %v", err)
	}
	if err := authService.Signup(outsiderID, "leave-http-outsider@example.com", "s3cret-pass"); err != nil {
		t.Fatalf("failed to sign up outsider: %v", err)
	}
	outsiderToken, err := authService.Login("leave-http-outsider@example.com", "s3cret-pass")
	if err != nil {
		t.Fatalf("failed to log in outsider: %v", err)
	}

	router := newLeaveGroupTestRouter(authService, membershipService, groupHandler)

	req := httptest.NewRequest(http.MethodDelete, "/groups/"+group.ID.String()+"/members/me", nil)
	req.Header.Set("Authorization", "Bearer "+outsiderToken)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("outsider leave request returned status %d, want 403, body: %s", rec.Code, rec.Body.String())
	}
}
