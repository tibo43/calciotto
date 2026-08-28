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

	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
)

// signupAndLoginRole is a small helper shared by the tests below: it signs
// up playerID with a throwaway email/password and returns a JWT for it, the
// same two-step dance every other _integration_test.go in this package
// repeats inline. Named distinctly from any helper in
// groupmembership_integration_test.go since both live in package
// handlers_test.
func signupAndLoginRole(t *testing.T, authService *services.AuthService, playerID uuid.UUID, email string) string {
	t.Helper()
	if err := authService.Signup(playerID, email, "s3cret-pass"); err != nil {
		t.Fatalf("failed to sign up %s: %v", email, err)
	}
	token, err := authService.Login(email, "s3cret-pass")
	if err != nil {
		t.Fatalf("failed to log in %s: %v", email, err)
	}
	return token
}

// TestCreateGroup_Integration_CreatorBecomesOwner exercises the role side of
// GroupHandler.CreateGroup: the authenticated caller who creates the group
// must come out of it as RoleOwner, not the RoleMember every other join path
// assigns.
func TestCreateGroup_Integration_CreatorBecomesOwner(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	membershipService := services.NewGroupMembershipService(tx)
	playerService := services.NewPlayerService(tx)
	authService := services.NewAuthService(tx, testGroupMembershipJWTSecret)
	groupHandler := handlers.NewGroupHandler(groupService, membershipService)

	creatorID, err := playerService.CreatePlayer("Zzz Role Creator")
	if err != nil {
		t.Fatalf("failed to create player: %v", err)
	}
	token := signupAndLoginRole(t, authService, creatorID, "role-creator@example.com")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/groups", handlers.AuthMiddleware(authService), groupHandler.CreateGroup)

	body := []byte(`{"name":"Zzz Role Owner Group"}`)
	req := httptest.NewRequest(http.MethodPost, "/groups", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create group returned status %d, want 200, body: %s", rec.Code, rec.Body.String())
	}

	var created models.Group
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to unmarshal created group: %v", err)
	}

	role, err := membershipService.GetRole(created.ID, creatorID)
	if err != nil {
		t.Fatalf("failed to get role: %v", err)
	}
	if role != models.RoleOwner {
		t.Errorf("creator role = %q, want %q", role, models.RoleOwner)
	}
}

// TestJoinGroup_Integration_JoinerBecomesMember covers POST /groups/join:
// joining by invite code must assign RoleMember, never RoleOwner.
func TestJoinGroup_Integration_JoinerBecomesMember(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	membershipService := services.NewGroupMembershipService(tx)
	playerService := services.NewPlayerService(tx)
	authService := services.NewAuthService(tx, testGroupMembershipJWTSecret)
	groupHandler := handlers.NewGroupHandler(groupService, membershipService)

	group, err := groupService.CreateGroup("Zzz Role Join Group")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	ownerID, err := playerService.CreatePlayer("Zzz Role Join Owner")
	if err != nil {
		t.Fatalf("failed to create owner player: %v", err)
	}
	if err := membershipService.AddPlayerToGroupWithRole(group.ID, ownerID, models.RoleOwner); err != nil {
		t.Fatalf("failed to add owner to group: %v", err)
	}

	joinerID, err := playerService.CreatePlayer("Zzz Role Joiner")
	if err != nil {
		t.Fatalf("failed to create joiner player: %v", err)
	}
	token := signupAndLoginRole(t, authService, joinerID, "role-joiner@example.com")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/groups/join", handlers.AuthMiddleware(authService), groupHandler.JoinGroup)

	body := []byte(`{"invite_code":"` + group.InviteCode + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/groups/join", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("join group returned status %d, want 200, body: %s", rec.Code, rec.Body.String())
	}

	role, err := membershipService.GetRole(group.ID, joinerID)
	if err != nil {
		t.Fatalf("failed to get role: %v", err)
	}
	if role != models.RoleMember {
		t.Errorf("joiner role = %q, want %q", role, models.RoleMember)
	}
}

// TestAddPlayerToGroup_Integration_AddedPlayerBecomesMember covers POST
// /groups/:id/players: an existing member adding another player must assign
// that player RoleMember, never RoleOwner.
func TestAddPlayerToGroup_Integration_AddedPlayerBecomesMember(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	membershipService := services.NewGroupMembershipService(tx)
	playerService := services.NewPlayerService(tx)
	authService := services.NewAuthService(tx, testGroupMembershipJWTSecret)
	groupHandler := handlers.NewGroupHandler(groupService, membershipService)

	group, err := groupService.CreateGroup("Zzz Role Add Group")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	actorID, err := playerService.CreatePlayer("Zzz Role Add Actor")
	if err != nil {
		t.Fatalf("failed to create actor player: %v", err)
	}
	if err := membershipService.AddPlayerToGroupWithRole(group.ID, actorID, models.RoleOwner); err != nil {
		t.Fatalf("failed to add actor to group: %v", err)
	}
	token := signupAndLoginRole(t, authService, actorID, "role-add-member@example.com")

	addedID, err := playerService.CreatePlayer("Zzz Role Added Player")
	if err != nil {
		t.Fatalf("failed to create added player: %v", err)
	}

	requireGroupMemberByPathID := handlers.RequireGroupMembershipByPathParam(membershipService, "id")
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/groups/:id/players", handlers.AuthMiddleware(authService), requireGroupMemberByPathID, groupHandler.AddPlayerToGroup)

	body := []byte(`{"player_id":"` + addedID.String() + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/groups/"+group.ID.String()+"/players", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("add player returned status %d, want 200, body: %s", rec.Code, rec.Body.String())
	}

	role, err := membershipService.GetRole(group.ID, addedID)
	if err != nil {
		t.Fatalf("failed to get role: %v", err)
	}
	if role != models.RoleMember {
		t.Errorf("added player role = %q, want %q", role, models.RoleMember)
	}
}

func newGroupOwnerTestRouter(authService *services.AuthService, membershipService *services.GroupMembershipService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	protected := router.Group("/owner-only")
	protected.Use(handlers.AuthMiddleware(authService), handlers.RequireGroupOwner(membershipService))
	protected.GET("", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	router.GET("/owner-only-by-path/:id",
		handlers.AuthMiddleware(authService),
		handlers.RequireGroupOwnerByPathParam(membershipService, "id"),
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

	return router
}

// TestRequireGroupOwner_Integration exercises the query-param path of
// RequireGroupOwner: the owner gets 200, a plain member gets 403, an
// outsider gets 403, and a request with no token gets 401.
func TestRequireGroupOwner_Integration(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	membershipService := services.NewGroupMembershipService(tx)
	playerService := services.NewPlayerService(tx)
	authService := services.NewAuthService(tx, testGroupMembershipJWTSecret)

	group, err := groupService.CreateGroup("Zzz Role Owner Middleware Group")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	ownerID, err := playerService.CreatePlayer("Zzz Role Owner Middleware Owner")
	if err != nil {
		t.Fatalf("failed to create owner player: %v", err)
	}
	if err := membershipService.AddPlayerToGroupWithRole(group.ID, ownerID, models.RoleOwner); err != nil {
		t.Fatalf("failed to add owner to group: %v", err)
	}
	ownerToken := signupAndLoginRole(t, authService, ownerID, "role-mw-owner@example.com")

	memberID, err := playerService.CreatePlayer("Zzz Role Owner Middleware Member")
	if err != nil {
		t.Fatalf("failed to create member player: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(group.ID, memberID); err != nil {
		t.Fatalf("failed to add member to group: %v", err)
	}
	memberToken := signupAndLoginRole(t, authService, memberID, "role-mw-member@example.com")

	outsiderID, err := playerService.CreatePlayer("Zzz Role Owner Middleware Outsider")
	if err != nil {
		t.Fatalf("failed to create outsider player: %v", err)
	}
	outsiderToken := signupAndLoginRole(t, authService, outsiderID, "role-mw-outsider@example.com")

	router := newGroupOwnerTestRouter(authService, membershipService)

	ownerReq := httptest.NewRequest(http.MethodGet, "/owner-only?group_id="+group.ID.String(), nil)
	ownerReq.Header.Set("Authorization", "Bearer "+ownerToken)
	ownerRec := httptest.NewRecorder()
	router.ServeHTTP(ownerRec, ownerReq)
	if ownerRec.Code != http.StatusOK {
		t.Errorf("owner request returned status %d, want 200, body: %s", ownerRec.Code, ownerRec.Body.String())
	}

	memberReq := httptest.NewRequest(http.MethodGet, "/owner-only?group_id="+group.ID.String(), nil)
	memberReq.Header.Set("Authorization", "Bearer "+memberToken)
	memberRec := httptest.NewRecorder()
	router.ServeHTTP(memberRec, memberReq)
	if memberRec.Code != http.StatusForbidden {
		t.Errorf("member request returned status %d, want 403, body: %s", memberRec.Code, memberRec.Body.String())
	}

	outsiderReq := httptest.NewRequest(http.MethodGet, "/owner-only?group_id="+group.ID.String(), nil)
	outsiderReq.Header.Set("Authorization", "Bearer "+outsiderToken)
	outsiderRec := httptest.NewRecorder()
	router.ServeHTTP(outsiderRec, outsiderReq)
	if outsiderRec.Code != http.StatusForbidden {
		t.Errorf("outsider request returned status %d, want 403, body: %s", outsiderRec.Code, outsiderRec.Body.String())
	}

	noTokenReq := httptest.NewRequest(http.MethodGet, "/owner-only?group_id="+group.ID.String(), nil)
	noTokenRec := httptest.NewRecorder()
	router.ServeHTTP(noTokenRec, noTokenReq)
	if noTokenRec.Code != http.StatusUnauthorized {
		t.Errorf("no-token request returned status %d, want 401, body: %s", noTokenRec.Code, noTokenRec.Body.String())
	}
}

// TestRequireGroupOwnerByPathParam_Integration exercises the path-param
// variant, same shape as TestRequireGroupOwner_Integration.
func TestRequireGroupOwnerByPathParam_Integration(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	membershipService := services.NewGroupMembershipService(tx)
	playerService := services.NewPlayerService(tx)
	authService := services.NewAuthService(tx, testGroupMembershipJWTSecret)

	group, err := groupService.CreateGroup("Zzz Role Owner Path Group")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	ownerID, err := playerService.CreatePlayer("Zzz Role Owner Path Owner")
	if err != nil {
		t.Fatalf("failed to create owner player: %v", err)
	}
	if err := membershipService.AddPlayerToGroupWithRole(group.ID, ownerID, models.RoleOwner); err != nil {
		t.Fatalf("failed to add owner to group: %v", err)
	}
	ownerToken := signupAndLoginRole(t, authService, ownerID, "role-mw-path-owner@example.com")

	memberID, err := playerService.CreatePlayer("Zzz Role Owner Path Member")
	if err != nil {
		t.Fatalf("failed to create member player: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(group.ID, memberID); err != nil {
		t.Fatalf("failed to add member to group: %v", err)
	}
	memberToken := signupAndLoginRole(t, authService, memberID, "role-mw-path-member@example.com")

	outsiderID, err := playerService.CreatePlayer("Zzz Role Owner Path Outsider")
	if err != nil {
		t.Fatalf("failed to create outsider player: %v", err)
	}
	outsiderToken := signupAndLoginRole(t, authService, outsiderID, "role-mw-path-outsider@example.com")

	router := newGroupOwnerTestRouter(authService, membershipService)

	ownerReq := httptest.NewRequest(http.MethodGet, "/owner-only-by-path/"+group.ID.String(), nil)
	ownerReq.Header.Set("Authorization", "Bearer "+ownerToken)
	ownerRec := httptest.NewRecorder()
	router.ServeHTTP(ownerRec, ownerReq)
	if ownerRec.Code != http.StatusOK {
		t.Errorf("owner request returned status %d, want 200, body: %s", ownerRec.Code, ownerRec.Body.String())
	}

	memberReq := httptest.NewRequest(http.MethodGet, "/owner-only-by-path/"+group.ID.String(), nil)
	memberReq.Header.Set("Authorization", "Bearer "+memberToken)
	memberRec := httptest.NewRecorder()
	router.ServeHTTP(memberRec, memberReq)
	if memberRec.Code != http.StatusForbidden {
		t.Errorf("member request returned status %d, want 403, body: %s", memberRec.Code, memberRec.Body.String())
	}

	outsiderReq := httptest.NewRequest(http.MethodGet, "/owner-only-by-path/"+group.ID.String(), nil)
	outsiderReq.Header.Set("Authorization", "Bearer "+outsiderToken)
	outsiderRec := httptest.NewRecorder()
	router.ServeHTTP(outsiderRec, outsiderReq)
	if outsiderRec.Code != http.StatusForbidden {
		t.Errorf("outsider request returned status %d, want 403, body: %s", outsiderRec.Code, outsiderRec.Body.String())
	}

	noTokenReq := httptest.NewRequest(http.MethodGet, "/owner-only-by-path/"+group.ID.String(), nil)
	noTokenRec := httptest.NewRecorder()
	router.ServeHTTP(noTokenRec, noTokenReq)
	if noTokenRec.Code != http.StatusUnauthorized {
		t.Errorf("no-token request returned status %d, want 401, body: %s", noTokenRec.Code, noTokenRec.Body.String())
	}
}
