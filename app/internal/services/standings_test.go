package services

import (
	"testing"
	"time"

	"app/internal/models"

	"github.com/google/uuid"
)

func newPlayer(id uuid.UUID, name string, goals int) models.PlayerCustom {
	return models.PlayerCustom{ID: id, Name: name, GoalsScored: goals}
}

func newTeam(id uuid.UUID, colour string, score int, players ...models.PlayerCustom) models.TeamWithPlayers {
	return models.TeamWithPlayers{ID: id, Colour: colour, Score: score, Players: players}
}

func newMatch(teams ...models.TeamWithPlayers) models.MatchWithDetails {
	return models.MatchWithDetails{ID: uuid.New(), Teams: teams}
}

func pointsRowsByID(rows []models.PointsStandingRow) map[uuid.UUID]models.PointsStandingRow {
	m := make(map[uuid.UUID]models.PointsStandingRow, len(rows))
	for _, r := range rows {
		m[r.PlayerID] = r
	}
	return m
}

func TestComputePointsStandings_WinLoss(t *testing.T) {
	alice, bob := uuid.New(), uuid.New()

	match := newMatch(
		newTeam(uuid.New(), "black", 2, newPlayer(alice, "alice", 2)),
		newTeam(uuid.New(), "white", 1, newPlayer(bob, "bob", 1)),
	)

	got := pointsRowsByID(ComputePointsStandings([]models.MatchWithDetails{match}))

	if r := got[alice]; r.Points != 3 || r.Won != 1 || r.Drawn != 0 || r.Lost != 0 || r.GoalsFor != 2 {
		t.Errorf("alice (winner) = %+v, want 3 points / 1 win / 2 goals", r)
	}
	if r := got[bob]; r.Points != 0 || r.Won != 0 || r.Drawn != 0 || r.Lost != 1 || r.GoalsFor != 1 {
		t.Errorf("bob (loser) = %+v, want 0 points / 1 loss / 1 goal", r)
	}
}

func TestComputePointsStandings_Draw(t *testing.T) {
	alice, bob := uuid.New(), uuid.New()

	match := newMatch(
		newTeam(uuid.New(), "black", 1, newPlayer(alice, "alice", 1)),
		newTeam(uuid.New(), "white", 1, newPlayer(bob, "bob", 1)),
	)

	got := pointsRowsByID(ComputePointsStandings([]models.MatchWithDetails{match}))

	for _, id := range []uuid.UUID{alice, bob} {
		if r := got[id]; r.Points != 1 || r.Drawn != 1 || r.Won != 0 || r.Lost != 0 {
			t.Errorf("player %s = %+v, want 1 point / 1 draw", id, r)
		}
	}
}

func TestComputePointsStandings_SkipsMatchWithAnEmptyTeam(t *testing.T) {
	alice := uuid.New()

	// White has no players yet: the match isn't "played" — it shouldn't count
	// as a win for black, nor a 0-0 draw for anyone.
	match := newMatch(
		newTeam(uuid.New(), "black", 0, newPlayer(alice, "alice", 0)),
		newTeam(uuid.New(), "white", 0),
	)

	rows := ComputePointsStandings([]models.MatchWithDetails{match})
	if len(rows) != 0 {
		t.Fatalf("expected no rows for a not-fully-rostered match, got %+v", rows)
	}
}

func TestComputePointsStandings_SkipsMatchWithoutExactlyTwoTeams(t *testing.T) {
	alice := uuid.New()

	match := newMatch(newTeam(uuid.New(), "black", 1, newPlayer(alice, "alice", 1)))

	rows := ComputePointsStandings([]models.MatchWithDetails{match})
	if len(rows) != 0 {
		t.Fatalf("expected no rows for a match with a single team, got %+v", rows)
	}
}

func TestComputePointsStandings_AccumulatesAcrossMatches(t *testing.T) {
	alice, bob := uuid.New(), uuid.New()

	match1 := newMatch(
		newTeam(uuid.New(), "black", 2, newPlayer(alice, "alice", 2)),
		newTeam(uuid.New(), "white", 0, newPlayer(bob, "bob", 0)),
	)
	match2 := newMatch(
		newTeam(uuid.New(), "black", 1, newPlayer(alice, "alice", 1)),
		newTeam(uuid.New(), "white", 1, newPlayer(bob, "bob", 1)),
	)

	got := pointsRowsByID(ComputePointsStandings([]models.MatchWithDetails{match1, match2}))

	if r := got[alice]; r.Played != 2 || r.Points != 4 || r.GoalsFor != 3 {
		t.Errorf("alice = %+v, want 2 played / 4 points (3 win + 1 draw) / 3 goals", r)
	}
	if r := got[bob]; r.Played != 2 || r.Points != 1 || r.GoalsFor != 1 {
		t.Errorf("bob = %+v, want 2 played / 1 point (1 draw) / 1 goal", r)
	}
}

func TestComputePointsStandings_SortOrder(t *testing.T) {
	// Alice and Bob both end up with 3 points, but Alice scored more goals so
	// she ranks first. Carol has 0 points and ranks last.
	alice, bob, carol := uuid.New(), uuid.New(), uuid.New()

	match1 := newMatch(
		newTeam(uuid.New(), "black", 3, newPlayer(alice, "alice", 3)),
		newTeam(uuid.New(), "white", 0, newPlayer(carol, "carol", 0)),
	)
	match2 := newMatch(
		newTeam(uuid.New(), "black", 1, newPlayer(bob, "bob", 1)),
		newTeam(uuid.New(), "white", 0, newPlayer(carol, "carol", 0)),
	)

	rows := ComputePointsStandings([]models.MatchWithDetails{match1, match2})
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d: %+v", len(rows), rows)
	}
	if rows[0].PlayerID != alice || rows[1].PlayerID != bob || rows[2].PlayerID != carol {
		t.Errorf("unexpected order: %+v", rows)
	}
}

func TestComputeScorers_SumsGoalsAcrossMatches(t *testing.T) {
	alice, bob := uuid.New(), uuid.New()

	match1 := newMatch(
		newTeam(uuid.New(), "black", 2, newPlayer(alice, "alice", 2)),
		newTeam(uuid.New(), "white", 1, newPlayer(bob, "bob", 1)),
	)
	match2 := newMatch(
		newTeam(uuid.New(), "black", 1, newPlayer(alice, "alice", 1)),
		newTeam(uuid.New(), "white", 3, newPlayer(bob, "bob", 3)),
	)

	rows := ComputeScorers([]models.MatchWithDetails{match1, match2})
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d: %+v", len(rows), rows)
	}
	if rows[0].PlayerID != bob || rows[0].Goals != 4 || rows[0].Played != 2 {
		t.Errorf("rows[0] = %+v, want bob with 4 goals over 2 matches", rows[0])
	}
	if rows[1].PlayerID != alice || rows[1].Goals != 3 {
		t.Errorf("rows[1] = %+v, want alice with 3 goals", rows[1])
	}
}

func TestComputeScorers_TieBrokenByName(t *testing.T) {
	alice, zack := uuid.New(), uuid.New()

	match := newMatch(
		newTeam(uuid.New(), "black", 1, newPlayer(zack, "zack", 1)),
		newTeam(uuid.New(), "white", 1, newPlayer(alice, "alice", 1)),
	)

	rows := ComputeScorers([]models.MatchWithDetails{match})
	if len(rows) != 2 || rows[0].Name != "alice" || rows[1].Name != "zack" {
		t.Errorf("expected alphabetical tie-break (alice, zack), got %+v", rows)
	}
}

// Own goals: ComputePointsStandings/ComputeScorers are pure functions of
// already-loaded match data, so Team.Score here is what MatchService would
// have already computed upstream (crediting the own goal to the OTHER
// team — see matches.go) — these tests exist to pin that these two
// functions never read PlayerCustom.OwnGoals themselves, so an own goal
// never inflates the scoring player's own GoalsFor/Goals tally.

func TestComputePointsStandings_OwnGoalsDoNotCountAsPersonalGoals(t *testing.T) {
	alice, bob := uuid.New(), uuid.New()

	// Alice concedes an own goal (credited to white's Score, hence white
	// shows 1-0 here) but scores nothing herself.
	match := newMatch(
		newTeam(uuid.New(), "black", 0, models.PlayerCustom{ID: alice, Name: "alice", GoalsScored: 0, OwnGoals: 1}),
		newTeam(uuid.New(), "white", 1, newPlayer(bob, "bob", 0)),
	)

	got := pointsRowsByID(ComputePointsStandings([]models.MatchWithDetails{match}))

	if r := got[alice]; r.GoalsFor != 0 || r.Won != 0 || r.Lost != 1 || r.OwnGoals != 1 {
		t.Errorf("alice = %+v, want 0 goals for, a loss, and OwnGoals 1 (an own goal is not a personal goal, but is still reported)", r)
	}
	if r := got[bob]; r.GoalsFor != 0 || r.Won != 1 || r.OwnGoals != 0 {
		t.Errorf("bob = %+v, want 0 personal goals, a win (credited from alice's own goal), and no own goals of his own", r)
	}
}

func TestComputePointsStandings_OwnGoalsAccumulateAcrossMatches(t *testing.T) {
	alice := uuid.New()

	match1 := newMatch(
		newTeam(uuid.New(), "black", 0, models.PlayerCustom{ID: alice, Name: "alice", GoalsScored: 1, OwnGoals: 1}),
		newTeam(uuid.New(), "white", 1, newPlayer(uuid.New(), "bob", 0)),
	)
	match2 := newMatch(
		newTeam(uuid.New(), "black", 2, models.PlayerCustom{ID: alice, Name: "alice", GoalsScored: 2, OwnGoals: 2}),
		newTeam(uuid.New(), "white", 0, newPlayer(uuid.New(), "carol", 0)),
	)

	got := pointsRowsByID(ComputePointsStandings([]models.MatchWithDetails{match1, match2}))

	if r := got[alice]; r.OwnGoals != 3 || r.GoalsFor != 3 {
		t.Errorf("alice = %+v, want OwnGoals 3 (1+2) and GoalsFor 3 (1+2), tracked independently", r)
	}
}

func TestComputeScorers_OwnGoalsExcludedFromPersonalTally(t *testing.T) {
	alice, bob := uuid.New(), uuid.New()

	match := newMatch(
		newTeam(uuid.New(), "black", 0, models.PlayerCustom{ID: alice, Name: "alice", GoalsScored: 0, OwnGoals: 1}),
		newTeam(uuid.New(), "white", 1, newPlayer(bob, "bob", 0)),
	)

	rows := ComputeScorers([]models.MatchWithDetails{match})
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows (one per rostered player), got %d: %+v", len(rows), rows)
	}
	byID := make(map[uuid.UUID]models.ScorerRow, len(rows))
	for _, r := range rows {
		byID[r.PlayerID] = r
	}
	if byID[alice].Goals != 0 {
		t.Errorf("alice's Goals = %d, want 0 (own goal must not count toward the scorers table)", byID[alice].Goals)
	}
	if byID[alice].OwnGoals != 1 {
		t.Errorf("alice's OwnGoals = %d, want 1 (still reported, just not folded into Goals)", byID[alice].OwnGoals)
	}
	if byID[bob].OwnGoals != 0 {
		t.Errorf("bob's OwnGoals = %d, want 0", byID[bob].OwnGoals)
	}
}

// motmRowsByID mirrors pointsRowsByID above, for ComputeMotmStandings' rows.
func motmRowsByID(rows []models.MotmStandingRow) map[uuid.UUID]models.MotmStandingRow {
	m := make(map[uuid.UUID]models.MotmStandingRow, len(rows))
	for _, r := range rows {
		m[r.PlayerID] = r
	}
	return m
}

// TestComputeMotmStandings_CountsOneAwardPerMatch: a match with a clear
// winner increments that player's award count by exactly one, and a player
// who was in the tally but not the eventual winner earns nothing that match.
func TestComputeMotmStandings_CountsOneAwardPerMatch(t *testing.T) {
	alice, bob := uuid.New(), uuid.New()
	match := newMatch(newTeam(uuid.New(), "black", 0, newPlayer(alice, "alice", 0), newPlayer(bob, "bob", 0)))

	votes := map[uuid.UUID][]models.MatchVoteTally{
		match.ID: {candidate(alice, "alice", 3), candidate(bob, "bob", 1)},
	}

	rows := motmRowsByID(ComputeMotmStandings([]models.MatchWithDetails{match}, votes))
	if r, ok := rows[alice]; !ok || r.Awards != 1 {
		t.Errorf("alice = %+v (present=%v), want 1 award", r, ok)
	}
	if _, ok := rows[bob]; ok {
		t.Errorf("bob has a standings row despite never winning the award: %+v", rows)
	}
}

// TestComputeMotmStandings_TieAwardsEveryone: a tie in one match's tally
// increments every tied player's count, matching ComputeMotmWinners' own
// tie-inclusive rule.
func TestComputeMotmStandings_TieAwardsEveryone(t *testing.T) {
	alice, bob := uuid.New(), uuid.New()
	match := newMatch(newTeam(uuid.New(), "black", 0, newPlayer(alice, "alice", 0), newPlayer(bob, "bob", 0)))

	votes := map[uuid.UUID][]models.MatchVoteTally{
		match.ID: {candidate(alice, "alice", 2), candidate(bob, "bob", 2)},
	}

	rows := motmRowsByID(ComputeMotmStandings([]models.MatchWithDetails{match}, votes))
	if r := rows[alice]; r.Awards != 1 {
		t.Errorf("alice = %+v, want 1 award", r)
	}
	if r := rows[bob]; r.Awards != 1 {
		t.Errorf("bob = %+v, want 1 award", r)
	}
}

// TestComputeMotmStandings_AccumulatesAcrossMatches: winning two matches'
// award counts as 2, not a new row each time.
func TestComputeMotmStandings_AccumulatesAcrossMatches(t *testing.T) {
	alice := uuid.New()
	match1 := newMatch(newTeam(uuid.New(), "black", 0, newPlayer(alice, "alice", 0)))
	match2 := newMatch(newTeam(uuid.New(), "black", 0, newPlayer(alice, "alice", 0)))

	votes := map[uuid.UUID][]models.MatchVoteTally{
		match1.ID: {candidate(alice, "alice", 1)},
		match2.ID: {candidate(alice, "alice", 5)},
	}

	rows := motmRowsByID(ComputeMotmStandings([]models.MatchWithDetails{match1, match2}, votes))
	if r := rows[alice]; r.Awards != 2 {
		t.Errorf("alice = %+v, want 2 awards (won both matches)", r)
	}
}

// TestComputeMotmStandings_MatchWithNoVotesContributesNothing: a match absent
// from the votesByMatch map (nobody voted) must not blow up or fabricate a
// row.
func TestComputeMotmStandings_MatchWithNoVotesContributesNothing(t *testing.T) {
	match := newMatch(newTeam(uuid.New(), "black", 0, newPlayer(uuid.New(), "alice", 0)))

	rows := ComputeMotmStandings([]models.MatchWithDetails{match}, map[uuid.UUID][]models.MatchVoteTally{})
	if len(rows) != 0 {
		t.Errorf("rows = %+v, want none for a match with no votes at all", rows)
	}
}

// TestComputeMotmStandings_SortOrder: most awards first, alphabetical name as
// the tie-break, mirroring ComputePointsStandings/ComputeScorers' own
// convention.
func TestComputeMotmStandings_SortOrder(t *testing.T) {
	alice, bob, carol := uuid.New(), uuid.New(), uuid.New()
	match1 := newMatch(newTeam(uuid.New(), "black", 0, newPlayer(alice, "alice", 0)))
	match2 := newMatch(newTeam(uuid.New(), "black", 0, newPlayer(bob, "bob", 0)))
	match3 := newMatch(newTeam(uuid.New(), "black", 0, newPlayer(carol, "carol", 0)))

	votes := map[uuid.UUID][]models.MatchVoteTally{
		match1.ID: {candidate(alice, "alice", 1)},
		match2.ID: {candidate(bob, "bob", 1)},
		match3.ID: {candidate(carol, "carol", 1)},
	}

	rows := ComputeMotmStandings([]models.MatchWithDetails{match1, match2, match3}, votes)
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d: %+v", len(rows), rows)
	}
	// All three are tied at 1 award each, so alphabetical order decides.
	if rows[0].Name != "alice" || rows[1].Name != "bob" || rows[2].Name != "carol" {
		t.Errorf("expected alphabetical tie-break (alice, bob, carol), got %+v", rows)
	}
}

// TestIsMatchCompleted mirrors the frontend's own matchStatus tests
// (MatchesPanel.spec.js/MatchDetails.spec.js): "Completed" starts at
// midnight the day after the match, "Upcoming" covers every instant up to
// and including the match's own day — independent of anything else about
// the match.
func TestIsMatchCompleted(t *testing.T) {
	playedOn := models.Date(time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC))

	tests := []struct {
		name string
		now  time.Time
		want bool
	}{
		{"on match day itself", time.Date(2026, 9, 3, 23, 0, 0, 0, time.UTC), false},
		{"one second before the day after", time.Date(2026, 9, 3, 23, 59, 59, 0, time.UTC), false},
		{"at the stroke of the day after", time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC), true},
		{"well after", time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match := models.MatchWithDetails{Date: playedOn}
			if got := IsMatchCompleted(match, tt.now); got != tt.want {
				t.Errorf("IsMatchCompleted() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestFilterCompletedMatches_ExcludesAComposedButUpcomingMatch is the exact
// scenario real feedback flagged: an admin can compose a scheduled match's
// roster (via "Fill teams from sign-ups") before its own kick-off, so
// ComputePointsStandings'/ComputeScorers' own "both teams have players"
// check alone can't tell an upcoming match from a played one. The filter
// applied upstream of those functions (see GetPointsStandings/GetScorers/
// GetMotmStandings in standings.go) is what actually excludes it.
func TestFilterCompletedMatches_ExcludesAComposedButUpcomingMatch(t *testing.T) {
	alice, bob := uuid.New(), uuid.New()
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)

	upcoming := models.MatchWithDetails{
		ID:   uuid.New(),
		Date: models.Date(now), // today — not completed until tomorrow
		Teams: []models.TeamWithPlayers{
			newTeam(uuid.New(), "black", 2, newPlayer(alice, "alice", 2)),
			newTeam(uuid.New(), "white", 1, newPlayer(bob, "bob", 1)),
		},
	}
	played := models.MatchWithDetails{
		ID:   uuid.New(),
		Date: models.Date(now.AddDate(0, 0, -2)),
		Teams: []models.TeamWithPlayers{
			newTeam(uuid.New(), "black", 2, newPlayer(alice, "alice", 2)),
			newTeam(uuid.New(), "white", 1, newPlayer(bob, "bob", 1)),
		},
	}

	filtered := FilterCompletedMatches([]models.MatchWithDetails{upcoming, played}, now)
	if len(filtered) != 1 || filtered[0].ID != played.ID {
		t.Fatalf("FilterCompletedMatches = %+v, want only the played match", filtered)
	}

	// Threaded all the way through: an upcoming composed match must not
	// inflate a player's points/goals until it's actually completed.
	rows := pointsRowsByID(ComputePointsStandings(filtered))
	if r := rows[alice]; r.Played != 1 {
		t.Errorf("alice.Played = %d after filtering, want 1 (only the played match should count)", r.Played)
	}
}

// predictionRowsByID mirrors pointsRowsByID for PredictionStandingRow.
func predictionRowsByID(rows []models.PredictionStandingRow) map[uuid.UUID]models.PredictionStandingRow {
	m := make(map[uuid.UUID]models.PredictionStandingRow, len(rows))
	for _, r := range rows {
		m[r.PlayerID] = r
	}
	return m
}

// With no prior match history, EloRatingsBeforeMatch returns every player
// at EloStartingRating on both sides, so ComputeMatchOdds(1500, 1500)'s own
// fixed values apply — 2.857.. for either team, 3.333.. for a draw (see
// matchOdds_test.go's own TestComputeMatchOdds_EvenMatch for the same
// figures pinned directly).
const (
	testEvenOdds     = 1.0 / 0.35
	testEvenDrawOdds = 1.0 / 0.30
)

func TestComputePredictionStandings_CorrectPredictionNetsOddsMinusOne(t *testing.T) {
	alice := uuid.New()
	teamAID, teamBID := uuid.New(), uuid.New()
	match := models.MatchWithDetails{
		ID:   uuid.New(),
		Date: models.Date(time.Now()),
		Teams: []models.TeamWithPlayers{
			{ID: teamAID, Colour: "black", Score: 2, Players: []models.PlayerCustom{{ID: uuid.New(), Name: "scorer"}}},
			{ID: teamBID, Colour: "white", Score: 0, Players: []models.PlayerCustom{{ID: uuid.New(), Name: "other"}}},
		},
	}
	predictions := map[uuid.UUID][]models.MatchPredictionEntry{
		match.ID: {{PlayerID: alice, Name: "alice", PredictedTeamID: &teamAID}},
	}

	rows := predictionRowsByID(ComputePredictionStandings([]models.MatchWithDetails{match}, []models.MatchWithDetails{match}, predictions))

	r, ok := rows[alice]
	if !ok {
		t.Fatalf("alice has no row at all, want one")
	}
	if r.Predictions != 1 || r.Correct != 1 {
		t.Errorf("alice = %+v, want 1 prediction, 1 correct", r)
	}
	wantPoints := testEvenOdds - 1
	if !almostEqual(r.Points, wantPoints) {
		t.Errorf("alice.Points = %.3f, want ~%.3f (even-match team odds minus the one token staked)", r.Points, wantPoints)
	}
}

func TestComputePredictionStandings_WrongPredictionNetsMinusOne(t *testing.T) {
	alice := uuid.New()
	teamAID, teamBID := uuid.New(), uuid.New()
	match := models.MatchWithDetails{
		ID:   uuid.New(),
		Date: models.Date(time.Now()),
		Teams: []models.TeamWithPlayers{
			{ID: teamAID, Colour: "black", Score: 2, Players: []models.PlayerCustom{{ID: uuid.New(), Name: "scorer"}}},
			{ID: teamBID, Colour: "white", Score: 0, Players: []models.PlayerCustom{{ID: uuid.New(), Name: "other"}}},
		},
	}
	// alice backs team B, but team A wins.
	predictions := map[uuid.UUID][]models.MatchPredictionEntry{
		match.ID: {{PlayerID: alice, Name: "alice", PredictedTeamID: &teamBID}},
	}

	rows := predictionRowsByID(ComputePredictionStandings([]models.MatchWithDetails{match}, []models.MatchWithDetails{match}, predictions))

	r := rows[alice]
	if r.Predictions != 1 || r.Correct != 0 {
		t.Errorf("alice = %+v, want 1 prediction, 0 correct", r)
	}
	if !almostEqual(r.Points, -1) {
		t.Errorf("alice.Points = %.3f, want -1 (the one token staked, lost)", r.Points)
	}
}

func TestComputePredictionStandings_CorrectDrawPrediction(t *testing.T) {
	alice := uuid.New()
	teamAID, teamBID := uuid.New(), uuid.New()
	match := models.MatchWithDetails{
		ID:   uuid.New(),
		Date: models.Date(time.Now()),
		Teams: []models.TeamWithPlayers{
			{ID: teamAID, Colour: "black", Score: 1, Players: []models.PlayerCustom{{ID: uuid.New(), Name: "a"}}},
			{ID: teamBID, Colour: "white", Score: 1, Players: []models.PlayerCustom{{ID: uuid.New(), Name: "b"}}},
		},
	}
	predictions := map[uuid.UUID][]models.MatchPredictionEntry{
		match.ID: {{PlayerID: alice, Name: "alice", PredictedTeamID: nil}},
	}

	rows := predictionRowsByID(ComputePredictionStandings([]models.MatchWithDetails{match}, []models.MatchWithDetails{match}, predictions))

	r := rows[alice]
	wantPoints := testEvenDrawOdds - 1
	if r.Correct != 1 || !almostEqual(r.Points, wantPoints) {
		t.Errorf("alice = %+v, want 1 correct and ~%.3f points (even-match draw odds minus stake)", r, wantPoints)
	}
}

func TestComputePredictionStandings_SkipsMatchesWithNoPredictions(t *testing.T) {
	teamAID, teamBID := uuid.New(), uuid.New()
	match := models.MatchWithDetails{
		ID:   uuid.New(),
		Date: models.Date(time.Now()),
		Teams: []models.TeamWithPlayers{
			{ID: teamAID, Players: []models.PlayerCustom{{ID: uuid.New(), Name: "a"}}},
			{ID: teamBID, Players: []models.PlayerCustom{{ID: uuid.New(), Name: "b"}}},
		},
	}

	rows := ComputePredictionStandings([]models.MatchWithDetails{match}, []models.MatchWithDetails{match}, map[uuid.UUID][]models.MatchPredictionEntry{})
	if len(rows) != 0 {
		t.Fatalf("expected no rows for a match nobody predicted, got %+v", rows)
	}
}

func TestComputePredictionStandings_SkipsAnUncomposedMatch(t *testing.T) {
	alice := uuid.New()
	teamAID, teamBID := uuid.New(), uuid.New()
	// Scheduled, but white's side has no roster yet — no derivable result.
	match := models.MatchWithDetails{
		ID:   uuid.New(),
		Date: models.Date(time.Now()),
		Teams: []models.TeamWithPlayers{
			{ID: teamAID, Players: []models.PlayerCustom{{ID: uuid.New(), Name: "a"}}},
			{ID: teamBID, Players: []models.PlayerCustom{}},
		},
	}
	predictions := map[uuid.UUID][]models.MatchPredictionEntry{
		match.ID: {{PlayerID: alice, Name: "alice", PredictedTeamID: &teamAID}},
	}

	rows := ComputePredictionStandings([]models.MatchWithDetails{match}, []models.MatchWithDetails{match}, predictions)
	if len(rows) != 0 {
		t.Fatalf("expected no rows for an uncomposed match, got %+v", rows)
	}
}

// TestComputePredictionStandings_UsesFullHistoryForOddsNotJustCountedMatches
// pins the exact bug this function's own doc comment warns about: settling
// a counted (season-filtered) match's prediction must replay Elo against the
// group's *entire* history, not just the matches passed in as "counted" —
// otherwise a group with more than one season would understate (or reset)
// every player's true rating the moment a season filter narrows the input.
func TestComputePredictionStandings_UsesFullHistoryForOddsNotJustCountedMatches(t *testing.T) {
	alice := []models.PlayerCustom{{ID: uuid.New(), Name: "alice"}}
	bob := []models.PlayerCustom{{ID: uuid.New(), Name: "bob"}}
	alicePredicts := alice[0].ID

	// A prior "season" of matches, all won by alice's side, pushing her Elo
	// rating well above the starting value before the counted match below.
	day1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 1, 8, 0, 0, 0, 0, time.UTC)
	day3 := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	priorSeason := []models.MatchWithDetails{
		eloMatch(day1, day1, alice, 3, bob, 0),
		eloMatch(day2, day2, alice, 3, bob, 0),
		eloMatch(day3, day3, alice, 3, bob, 0),
	}

	// This season's only match: alice's team (now the clear favourite) wins
	// again, and alice correctly predicted it.
	counted := eloMatch(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), time.Time{}, alice, 2, bob, 0)
	teamAID := counted.Teams[0].ID
	predictions := map[uuid.UUID][]models.MatchPredictionEntry{
		counted.ID: {{PlayerID: alicePredicts, Name: "alice", PredictedTeamID: &teamAID}},
	}

	allMatches := append(append([]models.MatchWithDetails{}, priorSeason...), counted)

	withFullHistory := predictionRowsByID(ComputePredictionStandings(allMatches, []models.MatchWithDetails{counted}, predictions))
	// The bug this guards against: passing only the counted match as if it
	// were also the full history, so EloRatingsBeforeMatch sees no prior
	// season at all and treats both sides as freshly started.
	withOnlyCountedAsHistory := predictionRowsByID(ComputePredictionStandings([]models.MatchWithDetails{counted}, []models.MatchWithDetails{counted}, predictions))

	if withFullHistory[alicePredicts].Points == withOnlyCountedAsHistory[alicePredicts].Points {
		t.Fatalf("points were identical (%.3f) whether or not the prior season was included — the odds must differ once alice's side is a known favourite rather than an even match", withFullHistory[alicePredicts].Points)
	}
	// Alice's side is now the proven favourite, so backing them correctly
	// nets *less* than the even-match baseline (2.857-1), not the same.
	if withFullHistory[alicePredicts].Points >= testEvenOdds-1 {
		t.Errorf("points with full history = %.3f, want below the even-match baseline (%.3f) now that alice's side is a real favourite", withFullHistory[alicePredicts].Points, testEvenOdds-1)
	}
}
