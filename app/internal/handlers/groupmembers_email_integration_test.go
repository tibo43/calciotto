package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"app/internal/handlers"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/gin-gonic/gin"
)

const testGroupMembersJWTSecret = "zzz-integration-test-group-members-secret"

// TestGetGroupMembers_Integration_NeverExposesEmails pins that GET
// /groups/:id/players, which any member can call, never carries an email
// address — neither the caller's nor anyone else's — and that has_account
// tells a player with an account from a ghost player without one.
func TestGetGroupMembers_Integration_NeverExposesEmails(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	playerService := services.NewPlayerService(tx)
	membershipService := services.NewGroupMembershipService(tx)
	authService := services.NewAuthService(tx, testGroupMembersJWTSecret)

	group, err := groupService.CreateGroup("Zzz Integration Members Email Group", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	const (
		callerEmail = "members-email-caller@example.com"
		otherEmail  = "members-email-other@example.com"
	)
	callerID, err := playerService.CreatePlayer("Zzz Integration Members Email Caller")
	if err != nil {
		t.Fatalf("failed to create the caller: %v", err)
	}
	otherID, err := playerService.CreatePlayer("Zzz Integration Members Email Other")
	if err != nil {
		t.Fatalf("failed to create the other player: %v", err)
	}
	ghostID, err := playerService.CreatePlayer("Zzz Integration Members Email Ghost")
	if err != nil {
		t.Fatalf("failed to create the ghost player: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(group.ID, callerID); err != nil {
		t.Fatalf("failed to add the caller: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(group.ID, otherID); err != nil {
		t.Fatalf("failed to add the other player: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(group.ID, ghostID); err != nil {
		t.Fatalf("failed to add the ghost player: %v", err)
	}
	if err := authService.Signup(callerID, callerEmail, "s3cret-pass"); err != nil {
		t.Fatalf("failed to sign up the caller: %v", err)
	}
	if err := authService.Signup(otherID, otherEmail, "s3cret-pass"); err != nil {
		t.Fatalf("failed to sign up the other player: %v", err)
	}
	token, err := authService.Login(callerEmail, "s3cret-pass")
	if err != nil {
		t.Fatalf("failed to log in the caller: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	groupHandler := handlers.NewGroupHandler(groupService, membershipService)
	router.GET("/groups/:id/players",
		handlers.AuthMiddleware(authService),
		handlers.RequireGroupMembershipByPathParam(membershipService, "id"),
		groupHandler.GetGroupMembers)

	req := httptest.NewRequest(http.MethodGet, "/groups/"+group.ID.String()+"/players", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /groups/:id/players returned status %d, want 200, body: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	for _, email := range []string{callerEmail, otherEmail} {
		if strings.Contains(body, email) {
			t.Errorf("response leaks %q: %s", email, body)
		}
	}
	if strings.Contains(body, "@") {
		t.Errorf("response contains an @, an address may be leaking: %s", body)
	}

	var members []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &members); err != nil {
		t.Fatalf("failed to decode the response: %v, body: %s", err, body)
	}
	if len(members) != 3 {
		t.Fatalf("got %d members, want 3: %s", len(members), body)
	}
	want := map[string]bool{
		callerID.String(): true,
		otherID.String():  true,
		ghostID.String():  false,
	}
	for _, member := range members {
		for key := range member {
			if strings.Contains(strings.ToLower(key), "email") {
				t.Errorf("member has an email field %q: %v", key, member)
			}
		}
		id, _ := member["id"].(string)
		wantHasAccount, known := want[id]
		if !known {
			t.Errorf("unexpected member %v in %s", member, body)
			continue
		}
		hasAccount, ok := member["has_account"].(bool)
		if !ok {
			t.Errorf("member %s has no boolean has_account: %v", id, member)
			continue
		}
		if hasAccount != wantHasAccount {
			t.Errorf("member %s has_account = %v, want %v", id, hasAccount, wantHasAccount)
		}
	}
}
