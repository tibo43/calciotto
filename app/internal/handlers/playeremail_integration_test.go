package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"app/internal/handlers"
	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/gin-gonic/gin"
)

const testPlayerEmailJWTSecret = "zzz-integration-test-player-email-secret"

// TestPlayerEmail_Integration_OnlyOnTheCallersProfile pins where a player's
// email address may appear: GET /players/me/stats returns the caller's own
// address (the Account page shows it), while the group roster and the
// points, scorers and MOTM rankings, which list every player of the group,
// never carry an address.
func TestPlayerEmail_Integration_OnlyOnTheCallersProfile(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	teamService := services.NewTeamService(tx)
	playerService := services.NewPlayerService(tx)
	matchService := services.NewMatchService(tx)
	voteService := services.NewMatchVoteService(tx)
	membershipService := services.NewGroupMembershipService(tx)
	authService := services.NewAuthService(tx, testPlayerEmailJWTSecret)

	group, err := groupService.CreateGroup("Zzz Integration Player Email Group", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	teams, err := teamService.GetTeamsByGroupID(group.ID)
	if err != nil || len(teams) != 2 {
		t.Fatalf("failed to load group teams: err=%v teams=%+v", err, teams)
	}

	const (
		aliceEmail = "player-email-alice@example.com"
		bobEmail   = "player-email-bob@example.com"
	)
	aliceID, err := playerService.CreatePlayer("Zzz Integration Player Email Alice")
	if err != nil {
		t.Fatalf("failed to create player alice: %v", err)
	}
	bobID, err := playerService.CreatePlayer("Zzz Integration Player Email Bob")
	if err != nil {
		t.Fatalf("failed to create player bob: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(group.ID, aliceID); err != nil {
		t.Fatalf("failed to add alice to the group: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(group.ID, bobID); err != nil {
		t.Fatalf("failed to add bob to the group: %v", err)
	}
	if err := authService.Signup(aliceID, aliceEmail, "s3cret-pass"); err != nil {
		t.Fatalf("failed to sign up alice: %v", err)
	}
	if err := authService.Signup(bobID, bobEmail, "s3cret-pass"); err != nil {
		t.Fatalf("failed to sign up bob: %v", err)
	}
	token, err := authService.Login(aliceEmail, "s3cret-pass")
	if err != nil {
		t.Fatalf("failed to log in alice: %v", err)
	}

	// One match yesterday (still inside the MOTM voting window, so the vote
	// can be cast), both players scoring, Bob voted MOTM by Alice, then moved
	// past the window so every ranking below has a row for each of them.
	matchID, err := matchService.CreateMatch(services.MatchSpec{Date: models.Date(time.Now().AddDate(0, 0, -1))}, group.ID)
	if err != nil {
		t.Fatalf("failed to create match: %v", err)
	}
	if err := matchService.UpdateMatch(matchID, group.ID, []models.TeamWithPlayers{
		{ID: teams[0].ID, Players: []models.PlayerCustom{{ID: aliceID, GoalsScored: 2}}},
		{ID: teams[1].ID, Players: []models.PlayerCustom{{ID: bobID, GoalsScored: 1}}},
	}); err != nil {
		t.Fatalf("UpdateMatch returned error: %v", err)
	}
	if err := voteService.Vote(matchID, aliceID, bobID); err != nil {
		t.Fatalf("failed to vote: %v", err)
	}
	// The MOTM ranking counts an award only once the vote is closed: move
	// the match back past its voting window now the vote is cast.
	if err := tx.Model(&models.Match{}).Where("id = ?", matchID).
		Update("date", time.Now().AddDate(0, 0, -3)).Error; err != nil {
		t.Fatalf("failed to close the vote: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	standingsHandler := handlers.NewStandingsHandler(services.NewStandingsService(tx, membershipService), membershipService)
	authRequired := handlers.AuthMiddleware(authService)
	requireGroupMember := handlers.RequireGroupMembership(membershipService)
	router.GET("/players/me/stats", authRequired, standingsHandler.GetPlayerProfile)
	router.GET("/groups/:id/players", authRequired,
		handlers.RequireGroupMembershipByPathParam(membershipService, "id"),
		handlers.NewGroupHandler(groupService, membershipService).GetGroupMembers)
	router.GET("/standings/points", authRequired, requireGroupMember, standingsHandler.GetPointsStandings)
	router.GET("/standings/scorers", authRequired, requireGroupMember, standingsHandler.GetScorers)
	router.GET("/standings/motm", authRequired, requireGroupMember, standingsHandler.GetMotmStandings)

	get := func(t *testing.T, path string) string {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s returned status %d, want 200, body: %s", path, rec.Code, rec.Body.String())
		}
		return rec.Body.String()
	}

	meBody := get(t, "/players/me/stats")
	var me models.PlayerProfileStats
	if err := json.Unmarshal([]byte(meBody), &me); err != nil {
		t.Fatalf("failed to decode GET /players/me/stats: %v, body: %s", err, meBody)
	}
	if me.Email != aliceEmail {
		t.Errorf("GET /players/me/stats Email = %q, want the caller's %q", me.Email, aliceEmail)
	}
	if strings.Contains(meBody, bobEmail) {
		t.Errorf("GET /players/me/stats leaks another player's address: %s", meBody)
	}

	for _, path := range []string{
		"/groups/" + group.ID.String() + "/players",
		"/standings/points?group_id=" + group.ID.String(),
		"/standings/scorers?group_id=" + group.ID.String(),
		"/standings/motm?group_id=" + group.ID.String(),
	} {
		body := get(t, path)
		if !strings.Contains(body, bobID.String()) {
			t.Fatalf("GET %s has no row for bob, nothing to check: %s", path, body)
		}
		for _, email := range []string{aliceEmail, bobEmail} {
			if strings.Contains(body, email) {
				t.Errorf("GET %s leaks %q: %s", path, email, body)
			}
		}
		if strings.Contains(strings.ToLower(body), `"email"`) {
			t.Errorf("GET %s has an email field: %s", path, body)
		}
	}
}
