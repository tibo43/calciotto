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

const testProfileJWTSecret = "zzz-integration-test-profile-secret"

func newProfileTestRouter(tx *gorm.DB, authService *services.AuthService, membershipService *services.GroupMembershipService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	standingsHandler := handlers.NewStandingsHandler(services.NewStandingsService(tx, membershipService), membershipService)
	// authRequired only, exactly like main.go: there is no single group to
	// authorize against, the handler is scoped to the token's own player.
	router.GET("/players/me/stats", handlers.AuthMiddleware(authService), standingsHandler.GetPlayerProfile)

	return router
}

// TestPlayerProfile_Integration covers the cross-group profile end to end:
// Alice belongs to three groups — she wins and loses in one, draws in
// another, and has never played in the third. Overall must aggregate every
// group, PerGroup must keep each group's numbers separate (including a zero
// row for the group she's never played in), and the season filter must apply
// to both.
func TestPlayerProfile_Integration(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	teamService := services.NewTeamService(tx)
	playerService := services.NewPlayerService(tx)
	matchService := services.NewMatchService(tx)
	membershipService := services.NewGroupMembershipService(tx)
	authService := services.NewAuthService(tx, testProfileJWTSecret)

	// Three groups: two Alice plays in, one she only belongs to.
	groupA, err := groupService.CreateGroup("Zzz Integration Profile Group A", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group A: %v", err)
	}
	groupB, err := groupService.CreateGroup("Zzz Integration Profile Group B", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group B: %v", err)
	}
	groupC, err := groupService.CreateGroup("Zzz Integration Profile Group C", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group C: %v", err)
	}

	teamsOf := func(t *testing.T, groupID uuid.UUID) []models.Team {
		t.Helper()
		teams, err := teamService.GetTeamsByGroupID(groupID)
		if err != nil || len(teams) != 2 {
			t.Fatalf("failed to load teams of group %s: err=%v teams=%+v", groupID, err, teams)
		}
		return teams
	}
	teamsA, teamsB := teamsOf(t, groupA.ID), teamsOf(t, groupB.ID)

	aliceID, err := playerService.CreatePlayer("Zzz Integration Profile Alice")
	if err != nil {
		t.Fatalf("failed to create player alice: %v", err)
	}
	bobID, err := playerService.CreatePlayer("Zzz Integration Profile Bob")
	if err != nil {
		t.Fatalf("failed to create player bob: %v", err)
	}
	carolID, err := playerService.CreatePlayer("Zzz Integration Profile Carol")
	if err != nil {
		t.Fatalf("failed to create player carol: %v", err)
	}
	for _, membership := range []struct {
		groupID  uuid.UUID
		playerID uuid.UUID
	}{
		{groupA.ID, aliceID}, {groupA.ID, bobID},
		{groupB.ID, aliceID}, {groupB.ID, carolID},
		{groupC.ID, aliceID},
	} {
		if err := membershipService.AddPlayerToGroup(membership.groupID, membership.playerID); err != nil {
			t.Fatalf("failed to add player %s to group %s: %v", membership.playerID, membership.groupID, err)
		}
	}

	if err := authService.Signup(aliceID, "profile-alice@example.com", "s3cret-pass"); err != nil {
		t.Fatalf("failed to sign up alice: %v", err)
	}
	token, err := authService.Login("profile-alice@example.com", "s3cret-pass")
	if err != nil {
		t.Fatalf("failed to log in alice: %v", err)
	}

	const (
		seasonOld = "2023-2024"
		seasonNew = "2024-2025"
	)
	// Group A: Alice wins 2-0 in the old season, loses 0-1 in the new one.
	// Group B: Alice draws 1-1 in the old season. Group C: nothing at all.
	fixtures := []struct {
		groupID                uuid.UUID
		teams                  []models.Team
		date                   models.Date
		opponentID             uuid.UUID
		aliceGoals, rivalGoals int
	}{
		{groupA.ID, teamsA, models.Date(time.Date(2024, time.August, 31, 0, 0, 0, 0, time.UTC)), bobID, 2, 0},
		{groupA.ID, teamsA, models.Date(time.Date(2024, time.September, 1, 0, 0, 0, 0, time.UTC)), bobID, 0, 1},
		{groupB.ID, teamsB, models.Date(time.Date(2024, time.August, 15, 0, 0, 0, 0, time.UTC)), carolID, 1, 1},
	}
	for _, f := range fixtures {
		matchID, err := matchService.CreateMatch(f.date, f.groupID)
		if err != nil {
			t.Fatalf("failed to create match on %s: %v", f.date, err)
		}
		if err := matchService.UpdateMatch(models.MatchWithDetails{
			ID: matchID,
			Teams: []models.TeamWithPlayers{
				{ID: f.teams[0].ID, Players: []models.PlayerCustom{{ID: aliceID, GoalsScored: f.aliceGoals}}},
				{ID: f.teams[1].ID, Players: []models.PlayerCustom{{ID: f.opponentID, GoalsScored: f.rivalGoals}}},
			},
		}); err != nil {
			t.Fatalf("UpdateMatch on %s returned error: %v", f.date, err)
		}
	}

	router := newProfileTestRouter(tx, authService, membershipService)

	getProfile := func(t *testing.T, path string) models.PlayerProfileStats {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s returned status %d, want 200, body: %s", path, rec.Code, rec.Body.String())
		}
		var profile models.PlayerProfileStats
		if err := json.Unmarshal(rec.Body.Bytes(), &profile); err != nil {
			t.Fatalf("failed to decode the response of GET %s: %v, body: %s", path, err, rec.Body.String())
		}
		return profile
	}

	// want compares only the counters — PlayerID/Name are checked separately.
	type want struct {
		played, won, drawn, lost, goalsFor, points int
	}
	checkRow := func(t *testing.T, label string, got models.PointsStandingRow, w want) {
		t.Helper()
		if got.PlayerID != aliceID {
			t.Errorf("%s: PlayerID = %s, want alice (%s)", label, got.PlayerID, aliceID)
		}
		actual := want{got.Played, got.Won, got.Drawn, got.Lost, got.GoalsFor, got.Points}
		if actual != w {
			t.Errorf("%s = %+v, want %+v", label, actual, w)
		}
	}
	groupRow := func(t *testing.T, profile models.PlayerProfileStats, groupID uuid.UUID) models.PlayerGroupStanding {
		t.Helper()
		for _, row := range profile.PerGroup {
			if row.GroupID == groupID {
				return row
			}
		}
		t.Fatalf("no PerGroup entry for group %s in %+v", groupID, profile.PerGroup)
		return models.PlayerGroupStanding{}
	}

	t.Run("all seasons", func(t *testing.T) {
		profile := getProfile(t, "/players/me/stats")

		// 1 win + 1 draw + 1 loss across two groups = 4 points, 3 goals.
		checkRow(t, "overall", profile.Overall, want{played: 3, won: 1, drawn: 1, lost: 1, goalsFor: 3, points: 4})
		if profile.Overall.Name != "Zzz Integration Profile Alice" {
			t.Errorf("overall Name = %q, want alice's name", profile.Overall.Name)
		}

		if len(profile.PerGroup) != 3 {
			t.Fatalf("PerGroup has %d entries, want 3 (one per group alice belongs to): %+v", len(profile.PerGroup), profile.PerGroup)
		}

		rowA := groupRow(t, profile, groupA.ID)
		checkRow(t, "group A", rowA.PointsStandingRow, want{played: 2, won: 1, lost: 1, goalsFor: 2, points: 3})
		if rowA.GroupName != groupA.Name {
			t.Errorf("group A GroupName = %q, want %q", rowA.GroupName, groupA.Name)
		}

		checkRow(t, "group B", groupRow(t, profile, groupB.ID).PointsStandingRow, want{played: 1, drawn: 1, goalsFor: 1, points: 1})

		// Alice belongs to group C but never played there: the entry must
		// still be present, zeroed — omitting it would hide the membership.
		checkRow(t, "group C", groupRow(t, profile, groupC.ID).PointsStandingRow, want{})
	})

	t.Run("season filter applies to overall and per group", func(t *testing.T) {
		profile := getProfile(t, "/players/me/stats?season="+seasonOld)

		checkRow(t, "overall "+seasonOld, profile.Overall, want{played: 2, won: 1, drawn: 1, goalsFor: 3, points: 4})
		checkRow(t, "group A "+seasonOld, groupRow(t, profile, groupA.ID).PointsStandingRow, want{played: 1, won: 1, goalsFor: 2, points: 3})
		checkRow(t, "group B "+seasonOld, groupRow(t, profile, groupB.ID).PointsStandingRow, want{played: 1, drawn: 1, goalsFor: 1, points: 1})
		checkRow(t, "group C "+seasonOld, groupRow(t, profile, groupC.ID).PointsStandingRow, want{})

		// The new season only has group A's loss — group B drops to zero
		// rather than disappearing, and so does the overall win/draw record.
		profile = getProfile(t, "/players/me/stats?season="+seasonNew)
		checkRow(t, "overall "+seasonNew, profile.Overall, want{played: 1, lost: 1})
		checkRow(t, "group A "+seasonNew, groupRow(t, profile, groupA.ID).PointsStandingRow, want{played: 1, lost: 1})
		checkRow(t, "group B "+seasonNew, groupRow(t, profile, groupB.ID).PointsStandingRow, want{})
	})

	t.Run("requires authentication", func(t *testing.T) {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/players/me/stats", nil))
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("unauthenticated GET /players/me/stats returned status %d, want 401", rec.Code)
		}
	})
}
