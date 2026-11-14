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
	matchIDs := make([]uuid.UUID, len(fixtures))
	for i, f := range fixtures {
		matchID, err := matchService.CreateMatch(services.MatchSpec{Date: f.date}, f.groupID)
		if err != nil {
			t.Fatalf("failed to create match on %s: %v", f.date, err)
		}
		matchIDs[i] = matchID
		if err := matchService.UpdateMatch(matchID, f.groupID, []models.TeamWithPlayers{
			{ID: f.teams[0].ID, Players: []models.PlayerCustom{{ID: aliceID, GoalsScored: f.aliceGoals}}},
			{ID: f.teams[1].ID, Players: []models.PlayerCustom{{ID: f.opponentID, GoalsScored: f.rivalGoals}}},
		}); err != nil {
			t.Fatalf("UpdateMatch on %s returned error: %v", f.date, err)
		}
	}

	// Bob votes for Alice as Man of the Match in group A's old-season win —
	// the only vote cast anywhere in this test — so the profile's MotmAwards
	// can be checked at both scopes (group A and overall) and both season
	// filters (present in seasonOld, absent in seasonNew) the same way the
	// points/goals assertions already are above. Inserted directly rather
	// than through MatchVoteService.Vote: that match is dated 2024-08-31,
	// long past the real Date-based voting window by the time this test
	// actually runs — this test is about GetPlayerProfile's aggregation, not
	// about VotingWindowError, so the row is seeded straight into the table.
	if err := tx.Create(&models.MatchVote{MatchID: matchIDs[0], VoterID: bobID, VotedForID: aliceID}).Error; err != nil {
		t.Fatalf("failed to cast Bob's MOTM vote for Alice: %v", err)
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
		checkRow(t, "overall", profile.Overall.PointsStandingRow, want{played: 3, won: 1, drawn: 1, lost: 1, goalsFor: 3, points: 4})
		if profile.Overall.Name != "Zzz Integration Profile Alice" {
			t.Errorf("overall Name = %q, want alice's name", profile.Overall.Name)
		}
		// The one vote cast anywhere in this test, for Alice, in group A.
		if profile.Overall.MotmAwards != 1 {
			t.Errorf("overall MotmAwards = %d, want 1", profile.Overall.MotmAwards)
		}
		// These matches are unscheduled, so none of them settle a prediction.
		if profile.Overall.PredictionPoints != 0 {
			t.Errorf("overall PredictionPoints = %v, want 0", profile.Overall.PredictionPoints)
		}
		assertSeasons(t, profile.Seasons, seasonOld, seasonNew)

		if len(profile.PerGroup) != 3 {
			t.Fatalf("PerGroup has %d entries, want 3 (one per group alice belongs to): %+v", len(profile.PerGroup), profile.PerGroup)
		}

		rowA := groupRow(t, profile, groupA.ID)
		checkRow(t, "group A", rowA.PointsStandingRow, want{played: 2, won: 1, lost: 1, goalsFor: 2, points: 3})
		if rowA.GroupName != groupA.Name {
			t.Errorf("group A GroupName = %q, want %q", rowA.GroupName, groupA.Name)
		}
		if rowA.MotmAwards != 1 {
			t.Errorf("group A MotmAwards = %d, want 1", rowA.MotmAwards)
		}

		rowB := groupRow(t, profile, groupB.ID)
		checkRow(t, "group B", rowB.PointsStandingRow, want{played: 1, drawn: 1, goalsFor: 1, points: 1})
		if rowB.MotmAwards != 0 {
			t.Errorf("group B MotmAwards = %d, want 0 (no votes cast there)", rowB.MotmAwards)
		}

		// Alice belongs to group C but never played there: the entry must
		// still be present, zeroed — omitting it would hide the membership.
		checkRow(t, "group C", groupRow(t, profile, groupC.ID).PointsStandingRow, want{})
	})

	t.Run("season filter applies to overall and per group", func(t *testing.T) {
		profile := getProfile(t, "/players/me/stats?season="+seasonOld)

		checkRow(t, "overall "+seasonOld, profile.Overall.PointsStandingRow, want{played: 2, won: 1, drawn: 1, goalsFor: 3, points: 4})
		checkRow(t, "group A "+seasonOld, groupRow(t, profile, groupA.ID).PointsStandingRow, want{played: 1, won: 1, goalsFor: 2, points: 3})
		checkRow(t, "group B "+seasonOld, groupRow(t, profile, groupB.ID).PointsStandingRow, want{played: 1, drawn: 1, goalsFor: 1, points: 1})
		checkRow(t, "group C "+seasonOld, groupRow(t, profile, groupC.ID).PointsStandingRow, want{})
		// The voted-for match falls in seasonOld — both scopes still see it.
		if profile.Overall.MotmAwards != 1 {
			t.Errorf("overall %s MotmAwards = %d, want 1", seasonOld, profile.Overall.MotmAwards)
		}
		if got := groupRow(t, profile, groupA.ID).MotmAwards; got != 1 {
			t.Errorf("group A %s MotmAwards = %d, want 1", seasonOld, got)
		}

		// The new season only has group A's loss — group B drops to zero
		// rather than disappearing, and so does the overall win/draw record.
		profile = getProfile(t, "/players/me/stats?season="+seasonNew)
		checkRow(t, "overall "+seasonNew, profile.Overall.PointsStandingRow, want{played: 1, lost: 1})
		checkRow(t, "group A "+seasonNew, groupRow(t, profile, groupA.ID).PointsStandingRow, want{played: 1, lost: 1})
		checkRow(t, "group B "+seasonNew, groupRow(t, profile, groupB.ID).PointsStandingRow, want{})
		// The vote was cast on the old-season match, so the new season's
		// filter must drop it from both scopes, the same way it drops the
		// win/draw record above.
		if profile.Overall.MotmAwards != 0 {
			t.Errorf("overall %s MotmAwards = %d, want 0", seasonNew, profile.Overall.MotmAwards)
		}
		if got := groupRow(t, profile, groupA.ID).MotmAwards; got != 0 {
			t.Errorf("group A %s MotmAwards = %d, want 0", seasonNew, got)
		}
		// The season list is the full set, not the filter that was asked for.
		assertSeasons(t, profile.Seasons, seasonOld, seasonNew)
	})

	t.Run("requires authentication", func(t *testing.T) {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/players/me/stats", nil))
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("unauthenticated GET /players/me/stats returned status %d, want 401", rec.Code)
		}
	})
}

func assertSeasons(t *testing.T, got []string, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("Seasons = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Seasons = %v, want %v", got, want)
		}
	}
}

func TestPlayerProfile_PredictionPointsAndSeasons(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	teamService := services.NewTeamService(tx)
	playerService := services.NewPlayerService(tx)
	matchService := services.NewMatchService(tx)
	membershipService := services.NewGroupMembershipService(tx)
	authService := services.NewAuthService(tx, testProfileJWTSecret)
	standings := services.NewStandingsService(tx, membershipService)

	groupA, err := groupService.CreateGroup("Zzz Profile Predictions A", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group A: %v", err)
	}
	groupB, err := groupService.CreateGroup("Zzz Profile Predictions B", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group B: %v", err)
	}
	teamsOf := func(groupID uuid.UUID) []models.Team {
		t.Helper()
		teams, err := teamService.GetTeamsByGroupID(groupID)
		if err != nil || len(teams) != 2 {
			t.Fatalf("teams of %s: err=%v teams=%d", groupID, err, len(teams))
		}
		return teams
	}
	teamsA, teamsB := teamsOf(groupA.ID), teamsOf(groupB.ID)

	aliceID, err := playerService.CreatePlayer("Zzz Profile Predictions Alice")
	if err != nil {
		t.Fatalf("failed to create alice: %v", err)
	}
	bobID, err := playerService.CreatePlayer("Zzz Profile Predictions Bob")
	if err != nil {
		t.Fatalf("failed to create bob: %v", err)
	}
	for _, membership := range []struct{ groupID, playerID uuid.UUID }{
		{groupA.ID, aliceID}, {groupA.ID, bobID},
		{groupB.ID, aliceID}, {groupB.ID, bobID},
	} {
		if err := membershipService.AddPlayerToGroup(membership.groupID, membership.playerID); err != nil {
			t.Fatalf("failed to add membership: %v", err)
		}
	}
	if err := authService.Signup(aliceID, "profile-predictions@example.com", "s3cret-pass"); err != nil {
		t.Fatalf("failed to sign up alice: %v", err)
	}
	token, err := authService.Login("profile-predictions@example.com", "s3cret-pass")
	if err != nil {
		t.Fatalf("failed to log in alice: %v", err)
	}

	// A past kick-off so recording the score sets ResultRecordedAt, which is
	// what makes the prediction settle. A future scheduled match contributes
	// its season and nothing else.
	playedAt := time.Date(2024, time.October, 1, 20, 0, 0, 0, time.UTC)
	futureAt := time.Date(2027, time.October, 1, 20, 0, 0, 0, time.UTC)
	schedule := func(groupID uuid.UUID, at time.Time) uuid.UUID {
		t.Helper()
		opens := at.Add(-24 * time.Hour)
		maxPlayers := 10
		id, err := matchService.CreateMatch(services.MatchSpec{
			Date:                models.DateOf(at),
			ScheduledAt:         &at,
			RegistrationOpensAt: &opens,
			MaxPlayers:          &maxPlayers,
		}, groupID)
		if err != nil {
			t.Fatalf("failed to schedule %s: %v", at, err)
		}
		return id
	}
	score := func(matchID, groupID, winner, loser, winnerTeam, loserTeam uuid.UUID) {
		t.Helper()
		if err := matchService.UpdateMatch(matchID, groupID, []models.TeamWithPlayers{
			{ID: winnerTeam, Players: []models.PlayerCustom{{ID: winner, GoalsScored: 2}}},
			{ID: loserTeam, Players: []models.PlayerCustom{{ID: loser, GoalsScored: 0}}},
		}); err != nil {
			t.Fatalf("failed to record the score: %v", err)
		}
	}

	matchA := schedule(groupA.ID, playedAt)
	score(matchA, groupA.ID, aliceID, bobID, teamsA[0].ID, teamsA[1].ID)
	matchB := schedule(groupB.ID, playedAt)
	score(matchB, groupB.ID, aliceID, bobID, teamsB[0].ID, teamsB[1].ID)
	schedule(groupA.ID, futureAt)

	// Alice calls group A's winner and group B's loser. A miss is always -1;
	// a hit is odds − 1, which is what GetPredictionStandings reports.
	winnerA := teamsA[0].ID
	loserB := teamsB[1].ID
	if err := tx.Create(&models.MatchPrediction{MatchID: matchA, PlayerID: aliceID, PredictedTeamID: &winnerA}).Error; err != nil {
		t.Fatalf("failed to store alice's hit: %v", err)
	}
	if err := tx.Create(&models.MatchPrediction{MatchID: matchB, PlayerID: aliceID, PredictedTeamID: &loserB}).Error; err != nil {
		t.Fatalf("failed to store alice's miss: %v", err)
	}

	rowsA, err := standings.GetPredictionStandings(groupA.ID, "")
	if err != nil {
		t.Fatalf("GetPredictionStandings: %v", err)
	}
	var wantHit float64
	found := false
	for _, row := range rowsA {
		if row.PlayerID == aliceID {
			wantHit = row.Points
			found = true
		}
	}
	if !found || wantHit == -1 || wantHit == 0 {
		t.Fatalf("group A leaderboard points for alice = %v (found %v), want a positive odds-1", wantHit, found)
	}

	router := newProfileTestRouter(tx, authService, membershipService)
	get := func(path string) models.PlayerProfileStats {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s = %d, body %s", path, rec.Code, rec.Body.String())
		}
		var profile models.PlayerProfileStats
		if err := json.Unmarshal(rec.Body.Bytes(), &profile); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		return profile
	}
	pointsOf := func(profile models.PlayerProfileStats, groupID uuid.UUID) float64 {
		t.Helper()
		for _, row := range profile.PerGroup {
			if row.GroupID == groupID {
				return row.PredictionPoints
			}
		}
		t.Fatalf("no row for %s", groupID)
		return 0
	}

	profile := get("/players/me/stats")
	assertSeasons(t, profile.Seasons, "2024-2025", "2027-2028")
	if got := pointsOf(profile, groupA.ID); got != wantHit {
		t.Errorf("group A PredictionPoints = %v, want %v (the group's own leaderboard)", got, wantHit)
	}
	if got := pointsOf(profile, groupB.ID); got != -1 {
		t.Errorf("group B PredictionPoints = %v, want -1", got)
	}
	if profile.Overall.PredictionPoints != wantHit-1 {
		t.Errorf("overall PredictionPoints = %v, want %v (sum of the groups, not a mixed Elo)", profile.Overall.PredictionPoints, wantHit-1)
	}

	// Filtering to the unplayed season does not drop it from Seasons, and
	// does not settle the other season's predictions into this total.
	filtered := get("/players/me/stats?season=2027-2028")
	assertSeasons(t, filtered.Seasons, "2024-2025", "2027-2028")
	if filtered.Overall.PredictionPoints != 0 || pointsOf(filtered, groupA.ID) != 0 {
		t.Errorf("unplayed season PredictionPoints = overall %v group A %v, want 0", filtered.Overall.PredictionPoints, pointsOf(filtered, groupA.ID))
	}
}
