package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"app/internal/handlers"
	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const testStandingsJWTSecret = "zzz-integration-test-standings-secret"

func newStandingsTestRouter(tx *gorm.DB, authService *services.AuthService, membershipService *services.GroupMembershipService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	standingsHandler := handlers.NewStandingsHandler(services.NewStandingsService(tx, membershipService), membershipService)
	authRequired := handlers.AuthMiddleware(authService)
	requireGroupMember := handlers.RequireGroupMembership(membershipService)

	router.GET("/standings/points", authRequired, requireGroupMember, standingsHandler.GetPointsStandings)
	router.GET("/standings/scorers", authRequired, requireGroupMember, standingsHandler.GetScorers)
	router.GET("/standings/seasons", authRequired, requireGroupMember, standingsHandler.GetSeasons)

	return router
}

// TestStandingsSeasons_Integration exercises the season query param end to
// end: /standings/seasons lists both seasons the group played in, and
// /standings/points scoped to one season only reports that season's results.
func TestStandingsSeasons_Integration(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	teamService := services.NewTeamService(tx)
	playerService := services.NewPlayerService(tx)
	matchService := services.NewMatchService(tx)
	membershipService := services.NewGroupMembershipService(tx)
	authService := services.NewAuthService(tx, testStandingsJWTSecret)

	group, err := groupService.CreateGroup("Zzz Integration Standings Season Group", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	teams, err := teamService.GetTeamsByGroupID(group.ID)
	if err != nil || len(teams) != 2 {
		t.Fatalf("failed to load group teams: err=%v teams=%+v", err, teams)
	}

	aliceID, err := playerService.CreatePlayer("Zzz Integration Standings Season Alice")
	if err != nil {
		t.Fatalf("failed to create player alice: %v", err)
	}
	bobID, err := playerService.CreatePlayer("Zzz Integration Standings Season Bob")
	if err != nil {
		t.Fatalf("failed to create player bob: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(group.ID, aliceID); err != nil {
		t.Fatalf("failed to add alice to the group: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(group.ID, bobID); err != nil {
		t.Fatalf("failed to add bob to the group: %v", err)
	}
	if err := authService.Signup(aliceID, "standings-season-alice@example.com", "s3cret-pass"); err != nil {
		t.Fatalf("failed to sign up alice: %v", err)
	}
	token, err := authService.Login("standings-season-alice@example.com", "s3cret-pass")
	if err != nil {
		t.Fatalf("failed to log in alice: %v", err)
	}

	const (
		seasonOld = "2023-2024"
		seasonNew = "2024-2025"
	)
	// Alice wins 2-0 in the old season, loses 0-1 in the new one.
	matches := []struct {
		date                 models.Date
		aliceGoals, bobGoals int
	}{
		{models.Date(time.Date(2024, time.August, 31, 0, 0, 0, 0, time.UTC)), 2, 0},
		{models.Date(time.Date(2024, time.September, 1, 0, 0, 0, 0, time.UTC)), 0, 1},
	}
	for _, m := range matches {
		matchID, err := matchService.CreateMatch(m.date, group.ID)
		if err != nil {
			t.Fatalf("failed to create match on %s: %v", m.date, err)
		}
		if err := matchService.UpdateMatch(models.MatchWithDetails{
			ID: matchID,
			Teams: []models.TeamWithPlayers{
				{ID: teams[0].ID, Players: []models.PlayerCustom{{ID: aliceID, GoalsScored: m.aliceGoals}}},
				{ID: teams[1].ID, Players: []models.PlayerCustom{{ID: bobID, GoalsScored: m.bobGoals}}},
			},
		}); err != nil {
			t.Fatalf("UpdateMatch on %s returned error: %v", m.date, err)
		}
	}

	router := newStandingsTestRouter(tx, authService, membershipService)

	get := func(t *testing.T, path string, out any) {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s returned status %d, want 200, body: %s", path, rec.Code, rec.Body.String())
		}
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			t.Fatalf("failed to decode the response of GET %s: %v, body: %s", path, err, rec.Body.String())
		}
	}

	var seasons []string
	get(t, "/standings/seasons?group_id="+group.ID.String(), &seasons)
	if len(seasons) != 2 || seasons[0] != seasonOld || seasons[1] != seasonNew {
		t.Errorf("GET /standings/seasons = %v, want [%s %s]", seasons, seasonOld, seasonNew)
	}

	var pointsOld []models.PointsStandingRow
	get(t, "/standings/points?group_id="+group.ID.String()+"&season="+seasonOld, &pointsOld)
	aliceOld := pointsRowByPlayerID(pointsOld, aliceID)
	if aliceOld == nil || aliceOld.Played != 1 || aliceOld.Points != 3 || aliceOld.GoalsFor != 2 {
		t.Errorf("alice's %s standing = %+v, want 1 played / 3 points / 2 goals", seasonOld, aliceOld)
	}

	var pointsNew []models.PointsStandingRow
	get(t, "/standings/points?group_id="+group.ID.String()+"&season="+seasonNew, &pointsNew)
	aliceNew := pointsRowByPlayerID(pointsNew, aliceID)
	if aliceNew == nil || aliceNew.Played != 1 || aliceNew.Points != 0 || aliceNew.GoalsFor != 0 {
		t.Errorf("alice's %s standing = %+v, want 1 played / 0 points / 0 goals", seasonNew, aliceNew)
	}

	// Omitting the season keeps the pre-existing, all-matches behaviour.
	var pointsAll []models.PointsStandingRow
	get(t, "/standings/points?group_id="+group.ID.String(), &pointsAll)
	aliceAll := pointsRowByPlayerID(pointsAll, aliceID)
	if aliceAll == nil || aliceAll.Played != 2 || aliceAll.Points != 3 {
		t.Errorf("alice's unfiltered standing = %+v, want 2 played / 3 points", aliceAll)
	}

	var scorersNew []models.ScorerRow
	get(t, "/standings/scorers?group_id="+group.ID.String()+"&season="+seasonNew, &scorersNew)
	bobNew := scorerRowByPlayerID(scorersNew, bobID)
	if bobNew == nil || bobNew.Goals != 1 {
		t.Errorf("bob's %s scorer row = %+v, want 1 goal", seasonNew, bobNew)
	}
	if aliceScorerNew := scorerRowByPlayerID(scorersNew, aliceID); aliceScorerNew == nil || aliceScorerNew.Goals != 0 {
		t.Errorf("alice's %s scorer row = %+v, want 0 goals", seasonNew, aliceScorerNew)
	}
}

func pointsRowByPlayerID(rows []models.PointsStandingRow, id uuid.UUID) *models.PointsStandingRow {
	for i := range rows {
		if rows[i].PlayerID == id {
			return &rows[i]
		}
	}
	return nil
}

func scorerRowByPlayerID(rows []models.ScorerRow, id uuid.UUID) *models.ScorerRow {
	for i := range rows {
		if rows[i].PlayerID == id {
			return &rows[i]
		}
	}
	return nil
}
