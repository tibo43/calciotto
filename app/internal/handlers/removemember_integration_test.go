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

const testRemoveMemberJWTSecret = "zzz-integration-test-remove-member-secret"

func newRemoveMemberTestRouter(authService *services.AuthService, membershipService *services.GroupMembershipService, groupHandler *handlers.GroupHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	requireGroupOwnerByPathID := handlers.RequireGroupOwnerByPathParam(membershipService, "id")
	router.DELETE("/groups/:id/members/:playerId",
		handlers.AuthMiddleware(authService),
		requireGroupOwnerByPathID,
		groupHandler.RemoveMember)

	// Mirrors a RequireGroupMembershipByPathParam-protected route wired in
	// main.go, used here only to prove that after removal, the target is
	// rejected with 403 on a subsequent request against the same group.
	requireGroupMemberByPathID := handlers.RequireGroupMembershipByPathParam(membershipService, "id")
	router.GET("/groups/:id/probe",
		handlers.AuthMiddleware(authService),
		requireGroupMemberByPathID,
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

	return router
}

// TestRemoveMember_Integration_HappyPath exercises
// DELETE /groups/:id/members/:playerId end to end: the owner removes another
// member successfully, and afterward the removed player loses access to a
// membership-protected route.
func TestRemoveMember_Integration_HappyPath(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	membershipService := services.NewGroupMembershipService(tx)
	playerService := services.NewPlayerService(tx)
	authService := services.NewAuthService(tx, testRemoveMemberJWTSecret)
	groupHandler := handlers.NewGroupHandler(groupService, membershipService)

	group, err := groupService.CreateGroup("Zzz Remove HTTP Group")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	ownerID, err := playerService.CreatePlayer("Zzz Remove HTTP Owner")
	if err != nil {
		t.Fatalf("failed to create owner player: %v", err)
	}
	if err := membershipService.AddPlayerToGroupWithRole(group.ID, ownerID, models.RoleOwner); err != nil {
		t.Fatalf("failed to add owner: %v", err)
	}
	if err := authService.Signup(ownerID, "remove-http-owner@example.com", "s3cret-pass"); err != nil {
		t.Fatalf("failed to sign up owner: %v", err)
	}
	ownerToken, err := authService.Login("remove-http-owner@example.com", "s3cret-pass")
	if err != nil {
		t.Fatalf("failed to log in owner: %v", err)
	}

	targetID, err := playerService.CreatePlayer("Zzz Remove HTTP Target")
	if err != nil {
		t.Fatalf("failed to create target player: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(group.ID, targetID); err != nil {
		t.Fatalf("failed to add target player: %v", err)
	}
	if err := authService.Signup(targetID, "remove-http-target@example.com", "s3cret-pass"); err != nil {
		t.Fatalf("failed to sign up target: %v", err)
	}
	targetToken, err := authService.Login("remove-http-target@example.com", "s3cret-pass")
	if err != nil {
		t.Fatalf("failed to log in target: %v", err)
	}

	router := newRemoveMemberTestRouter(authService, membershipService, groupHandler)

	req := httptest.NewRequest(http.MethodDelete, "/groups/"+group.ID.String()+"/members/"+targetID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("remove member returned status %d, want 200, body: %s", rec.Code, rec.Body.String())
	}

	isMember, err := membershipService.IsMember(group.ID, targetID)
	if err != nil {
		t.Fatalf("IsMember returned error: %v", err)
	}
	if isMember {
		t.Error("target is still a member after a successful removal")
	}

	probeReq := httptest.NewRequest(http.MethodGet, "/groups/"+group.ID.String()+"/probe", nil)
	probeReq.Header.Set("Authorization", "Bearer "+targetToken)
	probeRec := httptest.NewRecorder()
	router.ServeHTTP(probeRec, probeReq)
	if probeRec.Code != http.StatusForbidden {
		t.Errorf("group-scoped route after removal returned status %d, want 403, body: %s", probeRec.Code, probeRec.Body.String())
	}
}

// TestRemoveMember_Integration_NoToken covers the unauthenticated case: no
// Authorization header at all must be rejected by AuthMiddleware before the
// handler or the owner middleware ever runs.
func TestRemoveMember_Integration_NoToken(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	membershipService := services.NewGroupMembershipService(tx)
	playerService := services.NewPlayerService(tx)
	authService := services.NewAuthService(tx, testRemoveMemberJWTSecret)
	groupHandler := handlers.NewGroupHandler(groupService, membershipService)

	group, err := groupService.CreateGroup("Zzz Remove HTTP No Token Group")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	targetID, err := playerService.CreatePlayer("Zzz Remove HTTP No Token Target")
	if err != nil {
		t.Fatalf("failed to create target player: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(group.ID, targetID); err != nil {
		t.Fatalf("failed to add target player: %v", err)
	}

	router := newRemoveMemberTestRouter(authService, membershipService, groupHandler)

	req := httptest.NewRequest(http.MethodDelete, "/groups/"+group.ID.String()+"/members/"+targetID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("no-token remove request returned status %d, want 401, body: %s", rec.Code, rec.Body.String())
	}
}

// TestRemoveMember_Integration_NonOwnerForbidden covers a valid token for a
// plain member (not the owner) of the target group: RequireGroupOwnerByPathParam
// must reject with 403 without ever reaching the removal logic.
func TestRemoveMember_Integration_NonOwnerForbidden(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	membershipService := services.NewGroupMembershipService(tx)
	playerService := services.NewPlayerService(tx)
	authService := services.NewAuthService(tx, testRemoveMemberJWTSecret)
	groupHandler := handlers.NewGroupHandler(groupService, membershipService)

	group, err := groupService.CreateGroup("Zzz Remove HTTP NonOwner Group")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	ownerID, err := playerService.CreatePlayer("Zzz Remove HTTP NonOwner Owner")
	if err != nil {
		t.Fatalf("failed to create owner player: %v", err)
	}
	if err := membershipService.AddPlayerToGroupWithRole(group.ID, ownerID, models.RoleOwner); err != nil {
		t.Fatalf("failed to add owner: %v", err)
	}

	memberID, err := playerService.CreatePlayer("Zzz Remove HTTP NonOwner Member")
	if err != nil {
		t.Fatalf("failed to create member player: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(group.ID, memberID); err != nil {
		t.Fatalf("failed to add member: %v", err)
	}
	if err := authService.Signup(memberID, "remove-http-nonowner@example.com", "s3cret-pass"); err != nil {
		t.Fatalf("failed to sign up member: %v", err)
	}
	memberToken, err := authService.Login("remove-http-nonowner@example.com", "s3cret-pass")
	if err != nil {
		t.Fatalf("failed to log in member: %v", err)
	}

	targetID, err := playerService.CreatePlayer("Zzz Remove HTTP NonOwner Target")
	if err != nil {
		t.Fatalf("failed to create target player: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(group.ID, targetID); err != nil {
		t.Fatalf("failed to add target player: %v", err)
	}

	router := newRemoveMemberTestRouter(authService, membershipService, groupHandler)

	req := httptest.NewRequest(http.MethodDelete, "/groups/"+group.ID.String()+"/members/"+targetID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+memberToken)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("non-owner remove request returned status %d, want 403, body: %s", rec.Code, rec.Body.String())
	}

	isMember, err := membershipService.IsMember(group.ID, targetID)
	if err != nil {
		t.Fatalf("IsMember returned error: %v", err)
	}
	if !isMember {
		t.Error("target lost membership even though the remove request was forbidden")
	}
}
