package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"app/internal/handlers"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/gin-gonic/gin"
)

// TestMatchesDetails_Integration_SurvivesNewGroupWithLowerUUID reproduces the
// incident found testing auth in real conditions: GroupService.GetDefaultGroup
// picked "the default group" by sorting on a group's random UUID, which has
// no relation to which group a caller actually belongs to. As long as only
// one group existed this never showed up, but once RequireGroupMembership
// became a real authorization gate, anyone creating a second group (POST
// /groups is public) whose UUID happened to sort before the existing group's
// could flip "the default group" out from under every user who never passes
// group_id explicitly — which is every request the current frontend makes —
// and lock them out with 403 on requests that worked a moment earlier.
//
// The fix replaces that fallback, for authenticated requests, with
// GroupMembershipService.GetFirstGroupForPlayer — a group the caller is
// actually a member of — so this scenario must no longer break them.
func TestMatchesDetails_Integration_SurvivesNewGroupWithLowerUUID(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	membershipService := services.NewGroupMembershipService(tx)
	playerService := services.NewPlayerService(tx)
	authService := services.NewAuthService(tx, testGroupMembershipJWTSecret)
	matchHandler := handlers.NewMatchHandler(services.NewMatchService(tx), membershipService)

	groupA, err := groupService.CreateGroup("Zzz Bug Repro Group A", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group A: %v", err)
	}

	playerID, err := playerService.CreatePlayer("Zzz Bug Repro Player")
	if err != nil {
		t.Fatalf("failed to create player: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(groupA.ID, playerID); err != nil {
		t.Fatalf("failed to add player to group A: %v", err)
	}
	if err := authService.Signup(playerID, "bug-repro@example.com", "s3cret-pass"); err != nil {
		t.Fatalf("failed to sign up player: %v", err)
	}
	token, err := authService.Login("bug-repro@example.com", "s3cret-pass")
	if err != nil {
		t.Fatalf("failed to log in player: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/matches/details",
		handlers.AuthMiddleware(authService),
		handlers.RequireGroupMembership(membershipService),
		matchHandler.GetMatchesDetails)

	doRequest := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/matches/details", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	// Baseline: the player can list matches for their group without passing
	// group_id, same as the real frontend does.
	baseline := doRequest()
	if baseline.Code != http.StatusOK {
		t.Fatalf("baseline request returned status %d, want 200, body: %s", baseline.Code, baseline.Body.String())
	}

	// Simulate "n'importe qui" creating a second group whose UUID sorts
	// before group A's. GORM's BeforeCreate hook always overwrites any ID we
	// set on the model, so this bypasses the service layer with a raw insert
	// to pin an ID that is guaranteed (short of astronomical odds) to sort
	// before any randomly generated v4 UUID.
	const lowUUIDGroupID = "00000000-0000-0000-0000-000000000001"
	if err := tx.Exec("INSERT INTO groups (id, name) VALUES (?, ?)", lowUUIDGroupID, "Zzz Bug Repro Group B (lower uuid)").Error; err != nil {
		t.Fatalf("failed to insert low-UUID group B: %v", err)
	}

	// The exact regression: the same request, unchanged, must still succeed
	// — the player was never touched by group B's creation.
	afterNewGroup := doRequest()
	if afterNewGroup.Code != http.StatusOK {
		t.Fatalf("request after a lower-UUID group was created returned status %d, want 200, body: %s",
			afterNewGroup.Code, afterNewGroup.Body.String())
	}
}
