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
	"github.com/google/uuid"
)

const testMatchDetailsNotFoundJWTSecret = "zzz-integration-test-match-details-not-found-secret"

// TestGetMatchDetailsByID_Integration_CrossGroupReturns404 pins the fix for
// MatchHandler.GetMatchDetailsByID mapping every service error to 500: a
// match that exists but belongs to a different group than the one the
// caller is scoped to must come back as 404 (services.ErrMatchNotFound), the
// same "not found" a client sees for a match ID that doesn't exist at all —
// not the generic 500 a real server-side failure would produce.
func TestGetMatchDetailsByID_Integration_CrossGroupReturns404(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	membershipService := services.NewGroupMembershipService(tx)
	teamService := services.NewTeamService(tx)
	playerService := services.NewPlayerService(tx)
	matchService := services.NewMatchService(tx)
	authService := services.NewAuthService(tx, testMatchDetailsNotFoundJWTSecret)
	matchHandler := handlers.NewMatchHandler(matchService, membershipService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/matches/:id/details",
		handlers.AuthMiddleware(authService),
		handlers.RequireGroupMembership(membershipService),
		matchHandler.GetMatchDetailsByID)

	// Group A has the match under test; group B is a second group. Bob is a
	// member of both, so RequireGroupMembership (which only checks
	// membership of the group_id in the query string) lets his request for
	// group_id=groupB through — the point under test is what the
	// service/handler do once inside, not the membership gate itself.
	groupA, err := groupService.CreateGroup("Zzz Match Details 404 Group A", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group A: %v", err)
	}
	groupB, err := groupService.CreateGroup("Zzz Match Details 404 Group B", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group B: %v", err)
	}

	teamsA, err := teamService.GetTeamsByGroupID(groupA.ID)
	if err != nil || len(teamsA) != 2 {
		t.Fatalf("failed to load group A teams: err=%v teams=%+v", err, teamsA)
	}

	aliceID, err := playerService.CreatePlayer("Zzz Match Details 404 Alice")
	if err != nil {
		t.Fatalf("failed to create player alice: %v", err)
	}

	matchAID, err := matchService.CreateMatch(services.MatchSpec{Date: models.Date{}}, groupA.ID)
	if err != nil {
		t.Fatalf("failed to create match in group A: %v", err)
	}
	if err := matchService.UpdateMatch(models.MatchWithDetails{
		ID: matchAID,
		Teams: []models.TeamWithPlayers{
			{ID: teamsA[0].ID, Players: []models.PlayerCustom{{ID: aliceID, GoalsScored: 1}}},
		},
	}); err != nil {
		t.Fatalf("UpdateMatch (group A) returned error: %v", err)
	}

	bobID, err := playerService.CreatePlayer("Zzz Match Details 404 Bob")
	if err != nil {
		t.Fatalf("failed to create player bob: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(groupA.ID, bobID); err != nil {
		t.Fatalf("failed to add bob to group A: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(groupB.ID, bobID); err != nil {
		t.Fatalf("failed to add bob to group B: %v", err)
	}
	if err := authService.Signup(bobID, "match-details-404-bob@example.com", "s3cret-pass"); err != nil {
		t.Fatalf("failed to sign up bob: %v", err)
	}
	bobToken, err := authService.Login("match-details-404-bob@example.com", "s3cret-pass")
	if err != nil {
		t.Fatalf("failed to log in bob: %v", err)
	}

	// Requesting group A's match while scoped to group B (the "wrong group"
	// case the bug report describes) must be a 404, not a 500.
	wrongGroupReq := httptest.NewRequest(http.MethodGet,
		"/matches/"+matchAID.String()+"/details?group_id="+groupB.ID.String(), nil)
	wrongGroupReq.Header.Set("Authorization", "Bearer "+bobToken)
	wrongGroupRec := httptest.NewRecorder()
	router.ServeHTTP(wrongGroupRec, wrongGroupReq)
	if wrongGroupRec.Code != http.StatusNotFound {
		t.Fatalf("GET /matches/:id/details for a match outside the scoped group returned status %d, want 404, body: %s",
			wrongGroupRec.Code, wrongGroupRec.Body.String())
	}

	// Sanity check: the same match, correctly scoped to its own group, still
	// resolves — this isn't a case of the match having gone missing.
	okReq := httptest.NewRequest(http.MethodGet,
		"/matches/"+matchAID.String()+"/details?group_id="+groupA.ID.String(), nil)
	okReq.Header.Set("Authorization", "Bearer "+bobToken)
	okRec := httptest.NewRecorder()
	router.ServeHTTP(okRec, okReq)
	if okRec.Code != http.StatusOK {
		t.Fatalf("GET /matches/:id/details correctly scoped returned status %d, want 200, body: %s",
			okRec.Code, okRec.Body.String())
	}

	// An outright nonexistent match ID must also be a 404 (this already
	// worked before the fix, but pins that it keeps working).
	missingReq := httptest.NewRequest(http.MethodGet,
		"/matches/"+uuid.New().String()+"/details?group_id="+groupA.ID.String(), nil)
	missingReq.Header.Set("Authorization", "Bearer "+bobToken)
	missingRec := httptest.NewRecorder()
	router.ServeHTTP(missingRec, missingReq)
	if missingRec.Code != http.StatusNotFound {
		t.Fatalf("GET /matches/:id/details for a nonexistent match returned status %d, want 404, body: %s",
			missingRec.Code, missingRec.Body.String())
	}
}
