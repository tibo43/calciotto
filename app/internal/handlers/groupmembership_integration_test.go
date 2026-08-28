package handlers_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"app/internal/handlers"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/gin-gonic/gin"
)

const testGroupMembershipJWTSecret = "zzz-integration-test-group-membership-secret"

func newGroupMembershipTestRouter(authService *services.AuthService, membershipService *services.GroupMembershipService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/public", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	protected := router.Group("/protected")
	protected.Use(handlers.AuthMiddleware(authService), handlers.RequireGroupMembership(membershipService))
	protected.GET("", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	protected.POST("", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	router.GET("/protected-by-path/:id",
		handlers.AuthMiddleware(authService),
		handlers.RequireGroupMembershipByPathParam(membershipService, "id"),
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

	return router
}

// TestRequireGroupMembership_Integration exercises the query-param path of
// RequireGroupMembership: a member gets 200, a non-member gets 403, and a
// request with no token at all gets 401 — while an unrelated public route
// stays reachable without a token.
func TestRequireGroupMembership_Integration(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	membershipService := services.NewGroupMembershipService(tx)
	playerService := services.NewPlayerService(tx)
	authService := services.NewAuthService(tx, testGroupMembershipJWTSecret)

	group, err := groupService.CreateGroup("Zzz Integration Membership Group")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	memberID, err := playerService.CreatePlayer("Zzz Integration Membership Member")
	if err != nil {
		t.Fatalf("failed to create member player: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(group.ID, memberID); err != nil {
		t.Fatalf("failed to add member to group: %v", err)
	}
	if err := authService.Signup(memberID, "member@example.com", "s3cret-pass"); err != nil {
		t.Fatalf("failed to sign up member: %v", err)
	}
	memberToken, err := authService.Login("member@example.com", "s3cret-pass")
	if err != nil {
		t.Fatalf("failed to log in member: %v", err)
	}

	outsiderID, err := playerService.CreatePlayer("Zzz Integration Membership Outsider")
	if err != nil {
		t.Fatalf("failed to create outsider player: %v", err)
	}
	if err := authService.Signup(outsiderID, "outsider@example.com", "s3cret-pass"); err != nil {
		t.Fatalf("failed to sign up outsider: %v", err)
	}
	outsiderToken, err := authService.Login("outsider@example.com", "s3cret-pass")
	if err != nil {
		t.Fatalf("failed to log in outsider: %v", err)
	}

	router := newGroupMembershipTestRouter(authService, membershipService)

	memberReq := httptest.NewRequest(http.MethodGet, "/protected?group_id="+group.ID.String(), nil)
	memberReq.Header.Set("Authorization", "Bearer "+memberToken)
	memberRec := httptest.NewRecorder()
	router.ServeHTTP(memberRec, memberReq)
	if memberRec.Code != http.StatusOK {
		t.Errorf("member request returned status %d, want 200, body: %s", memberRec.Code, memberRec.Body.String())
	}

	outsiderReq := httptest.NewRequest(http.MethodGet, "/protected?group_id="+group.ID.String(), nil)
	outsiderReq.Header.Set("Authorization", "Bearer "+outsiderToken)
	outsiderRec := httptest.NewRecorder()
	router.ServeHTTP(outsiderRec, outsiderReq)
	if outsiderRec.Code != http.StatusForbidden {
		t.Errorf("non-member request returned status %d, want 403, body: %s", outsiderRec.Code, outsiderRec.Body.String())
	}

	noTokenReq := httptest.NewRequest(http.MethodGet, "/protected?group_id="+group.ID.String(), nil)
	noTokenRec := httptest.NewRecorder()
	router.ServeHTTP(noTokenRec, noTokenReq)
	if noTokenRec.Code != http.StatusUnauthorized {
		t.Errorf("no-token request returned status %d, want 401, body: %s", noTokenRec.Code, noTokenRec.Body.String())
	}

	publicReq := httptest.NewRequest(http.MethodGet, "/public", nil)
	publicRec := httptest.NewRecorder()
	router.ServeHTTP(publicRec, publicReq)
	if publicRec.Code != http.StatusOK {
		t.Errorf("public route without token returned status %d, want 200, body: %s", publicRec.Code, publicRec.Body.String())
	}
}

// TestRequireGroupMembership_Integration_BodyGroupID exercises the body-peek
// fallback used by POST /matches (group_id) and PUT /matches/:id (GroupID) —
// the group id must be readable from either JSON key without consuming the
// body the downstream handler still needs to bind.
func TestRequireGroupMembership_Integration_BodyGroupID(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	membershipService := services.NewGroupMembershipService(tx)
	playerService := services.NewPlayerService(tx)
	authService := services.NewAuthService(tx, testGroupMembershipJWTSecret)

	group, err := groupService.CreateGroup("Zzz Integration Membership Body Group")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	memberID, err := playerService.CreatePlayer("Zzz Integration Membership Body Member")
	if err != nil {
		t.Fatalf("failed to create member player: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(group.ID, memberID); err != nil {
		t.Fatalf("failed to add member to group: %v", err)
	}
	if err := authService.Signup(memberID, "body-member@example.com", "s3cret-pass"); err != nil {
		t.Fatalf("failed to sign up member: %v", err)
	}
	memberToken, err := authService.Login("body-member@example.com", "s3cret-pass")
	if err != nil {
		t.Fatalf("failed to log in member: %v", err)
	}

	router := newGroupMembershipTestRouter(authService, membershipService)

	for _, key := range []string{"group_id", "GroupID"} {
		body := []byte(`{"` + key + `":"` + group.ID.String() + `"}`)
		req := httptest.NewRequest(http.MethodPost, "/protected", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+memberToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("body key %q: returned status %d, want 200, body: %s", key, rec.Code, rec.Body.String())
		}
	}
}

// TestRequireGroupMembershipByPathParam_Integration exercises the path-param
// variant used by GET/POST /groups/:id/* routes.
func TestRequireGroupMembershipByPathParam_Integration(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	membershipService := services.NewGroupMembershipService(tx)
	playerService := services.NewPlayerService(tx)
	authService := services.NewAuthService(tx, testGroupMembershipJWTSecret)

	group, err := groupService.CreateGroup("Zzz Integration Membership Path Group")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	memberID, err := playerService.CreatePlayer("Zzz Integration Membership Path Member")
	if err != nil {
		t.Fatalf("failed to create member player: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(group.ID, memberID); err != nil {
		t.Fatalf("failed to add member to group: %v", err)
	}
	if err := authService.Signup(memberID, "path-member@example.com", "s3cret-pass"); err != nil {
		t.Fatalf("failed to sign up member: %v", err)
	}
	memberToken, err := authService.Login("path-member@example.com", "s3cret-pass")
	if err != nil {
		t.Fatalf("failed to log in member: %v", err)
	}

	outsiderID, err := playerService.CreatePlayer("Zzz Integration Membership Path Outsider")
	if err != nil {
		t.Fatalf("failed to create outsider player: %v", err)
	}
	if err := authService.Signup(outsiderID, "path-outsider@example.com", "s3cret-pass"); err != nil {
		t.Fatalf("failed to sign up outsider: %v", err)
	}
	outsiderToken, err := authService.Login("path-outsider@example.com", "s3cret-pass")
	if err != nil {
		t.Fatalf("failed to log in outsider: %v", err)
	}

	router := newGroupMembershipTestRouter(authService, membershipService)

	memberReq := httptest.NewRequest(http.MethodGet, "/protected-by-path/"+group.ID.String(), nil)
	memberReq.Header.Set("Authorization", "Bearer "+memberToken)
	memberRec := httptest.NewRecorder()
	router.ServeHTTP(memberRec, memberReq)
	if memberRec.Code != http.StatusOK {
		t.Errorf("member request returned status %d, want 200, body: %s", memberRec.Code, memberRec.Body.String())
	}

	outsiderReq := httptest.NewRequest(http.MethodGet, "/protected-by-path/"+group.ID.String(), nil)
	outsiderReq.Header.Set("Authorization", "Bearer "+outsiderToken)
	outsiderRec := httptest.NewRecorder()
	router.ServeHTTP(outsiderRec, outsiderReq)
	if outsiderRec.Code != http.StatusForbidden {
		t.Errorf("non-member request returned status %d, want 403, body: %s", outsiderRec.Code, outsiderRec.Body.String())
	}

	noTokenReq := httptest.NewRequest(http.MethodGet, "/protected-by-path/"+group.ID.String(), nil)
	noTokenRec := httptest.NewRecorder()
	router.ServeHTTP(noTokenRec, noTokenReq)
	if noTokenRec.Code != http.StatusUnauthorized {
		t.Errorf("no-token request returned status %d, want 401, body: %s", noTokenRec.Code, noTokenRec.Body.String())
	}
}
