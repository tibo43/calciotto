package services

import (
	"math"
	"testing"
	"time"

	"app/internal/models"

	"github.com/google/uuid"
)

// eloRowsByID mirrors pointsRowsByID (standings_test.go) for EloStandingRow.
func eloRowsByID(rows []models.EloStandingRow) map[uuid.UUID]models.EloStandingRow {
	m := make(map[uuid.UUID]models.EloStandingRow, len(rows))
	for _, r := range rows {
		m[r.PlayerID] = r
	}
	return m
}

// eloMatch builds a MatchWithDetails suitable for ComputeEloStandings: a
// single-player-per-team match on a given date, with team scores standing in
// for the match result (win/draw/loss) exactly the way ComputePointsStandings
// reads Team.Score. createdAt lets a test pin the tie-break below Date, which
// matters once two matches share the same day (see the sort-order test).
func eloMatch(date time.Time, createdAt time.Time, teamA []models.PlayerCustom, scoreA int, teamB []models.PlayerCustom, scoreB int) models.MatchWithDetails {
	return models.MatchWithDetails{
		ID:        uuid.New(),
		Date:      models.Date(date),
		CreatedAt: createdAt,
		Teams: []models.TeamWithPlayers{
			{ID: uuid.New(), Colour: "black", Score: scoreA, Players: teamA},
			{ID: uuid.New(), Colour: "white", Score: scoreB, Players: teamB},
		},
	}
}

func assertRating(t *testing.T, label string, got, want float64) {
	t.Helper()
	const epsilon = 0.01
	if math.Abs(got-want) > epsilon {
		t.Errorf("%s rating = %v, want %v (±%v)", label, got, want, epsilon)
	}
}

// TestComputeEloStandings_FirstMatchBetweenNewPlayers covers case (a): two
// players meeting for the first time ever both start at EloStartingRating,
// so the match is an even contest (expected 0.5/0.5) and both use the
// provisional K-factor. Hand-computed: E=0.5 each, K=60 each, A wins so
// delta = 60*(1-0.5) = +30 for A and 60*(0-0.5) = -30 for B.
func TestComputeEloStandings_FirstMatchBetweenNewPlayers(t *testing.T) {
	alice, bob := uuid.New(), uuid.New()
	day := time.Date(2026, time.January, 4, 0, 0, 0, 0, time.UTC)

	match := eloMatch(day, day,
		[]models.PlayerCustom{{ID: alice, Name: "alice"}}, 1,
		[]models.PlayerCustom{{ID: bob, Name: "bob"}}, 0,
	)

	got := eloRowsByID(ComputeEloStandings([]models.MatchWithDetails{match}))

	assertRating(t, "alice", got[alice].Rating, 1530.0)
	assertRating(t, "bob", got[bob].Rating, 1470.0)
	if got[alice].GamesPlayed != 1 || got[bob].GamesPlayed != 1 {
		t.Errorf("games played = alice:%d bob:%d, want 1/1", got[alice].GamesPlayed, got[bob].GamesPlayed)
	}
}

// TestComputeEloStandings_Draw covers case (c): a draw (equal scores) is an
// actual score of 0.5/0.5 for both sides, applied against ratings that are
// *not* equal (so the two deltas differ even though the actual score is
// symmetric). alice arrives at the draw match having won 5 straight
// 1-a-side matches against fresh opponents (rating pulled above
// EloStartingRating, 5 games played — still provisional, 5 < 10) and bob
// arrives having lost 3 straight (rating pulled below EloStartingRating, 3
// games played — also still provisional). Values below (both the pre-draw
// ratings and the post-draw result) were computed by a throwaway script
// replaying this exact formula, not hand-solved — a nonlinear recurrence
// over 5/3 prior matches has no convenient closed form.
func TestComputeEloStandings_Draw(t *testing.T) {
	alice, bob := uuid.New(), uuid.New()
	base := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	matches := make([]models.MatchWithDetails, 0, 9)
	for i := 0; i < 5; i++ {
		day := base.AddDate(0, 0, i)
		matches = append(matches, eloMatch(day, day,
			[]models.PlayerCustom{{ID: alice, Name: "alice"}}, 1,
			[]models.PlayerCustom{{ID: uuid.New(), Name: "alice-warmup-opponent"}}, 0,
		))
	}
	for i := 0; i < 3; i++ {
		day := base.AddDate(0, 0, i)
		matches = append(matches, eloMatch(day, day,
			[]models.PlayerCustom{{ID: bob, Name: "bob"}}, 0,
			[]models.PlayerCustom{{ID: uuid.New(), Name: "bob-warmup-opponent"}}, 1,
		))
	}

	// Sanity check the warm-up alone before the draw: alice pulled above
	// EloStartingRating, bob pulled below, both still provisional (< 10
	// games).
	warmup := eloRowsByID(ComputeEloStandings(matches))
	assertRating(t, "alice after warm-up", warmup[alice].Rating, 1626.669664820)
	assertRating(t, "bob after warm-up", warmup[bob].Rating, 1417.497042884)

	drawDay := base.AddDate(0, 0, 10)
	matches = append(matches, eloMatch(drawDay, drawDay,
		[]models.PlayerCustom{{ID: alice, Name: "alice"}}, 1,
		[]models.PlayerCustom{{ID: bob, Name: "bob"}}, 1,
	))

	got := eloRowsByID(ComputeEloStandings(matches))
	assertRating(t, "alice", got[alice].Rating, 1610.514523576)
	assertRating(t, "bob", got[bob].Rating, 1433.652184128)
	if got[alice].GamesPlayed != 6 || got[bob].GamesPlayed != 4 {
		t.Errorf("games played = alice:%d bob:%d, want 6/4", got[alice].GamesPlayed, got[bob].GamesPlayed)
	}
}

// TestComputeEloStandings_DifferentKFactorsInSameMatch covers case (b): X has
// already played 10 rated matches in this group (crossing
// EloProvisionalGameThreshold, so X's own K-factor is now EloKSteady) while Y
// is brand new (EloKProvisional). X beats the same opponent Z 2-0 ten times
// in a row (both starting fresh, so this is also a self-contained,
// hand-verifiable sub-computation), then meets brand-new Y and wins 3-1.
// Exact values computed via a throwaway script replaying the same formula
// this test exercises:
//
//	after 10 straight 2-0 wins over a fresh Z each time... no — same Z
//	reused throughout: X = 1662.695247268551 (10 games), Z = 1337.304752731...
//	Match 11 (X vs brand-new Y, X wins 3-1): ratingA=1662.695247268551 (K=40,
//	since 10 >= threshold), ratingB=1500 (K=60, since 0 < threshold).
//	E_X = 1/(1+10^((1500-1662.695247268551)/400)) ≈ 0.7241469...
//	delta_X = 40*(1-0.7241469) ≈ +11.263917 -> X ≈ 1673.959164
//	delta_Y = 60*(0-0.2758531) ≈ -16.551186 -> Y ≈ 1483.104124
func TestComputeEloStandings_DifferentKFactorsInSameMatch(t *testing.T) {
	x, z, y := uuid.New(), uuid.New(), uuid.New()
	base := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)

	matches := make([]models.MatchWithDetails, 0, 11)
	for i := 0; i < 10; i++ {
		day := base.AddDate(0, 0, i)
		matches = append(matches, eloMatch(day, day,
			[]models.PlayerCustom{{ID: x, Name: "x"}}, 2,
			[]models.PlayerCustom{{ID: z, Name: "z"}}, 0,
		))
	}
	// Sanity check on the warm-up alone: X should have exactly 10 games and
	// a rating strictly above EloStartingRating (ten straight wins).
	warmup := eloRowsByID(ComputeEloStandings(matches))
	if warmup[x].GamesPlayed != 10 {
		t.Fatalf("warm-up: X played %d games, want 10", warmup[x].GamesPlayed)
	}
	assertRating(t, "X after warm-up", warmup[x].Rating, 1662.695247269)

	match11Day := base.AddDate(0, 0, 10)
	match11 := eloMatch(match11Day, match11Day,
		[]models.PlayerCustom{{ID: x, Name: "x"}}, 3,
		[]models.PlayerCustom{{ID: y, Name: "y"}}, 1,
	)
	matches = append(matches, match11)

	got := eloRowsByID(ComputeEloStandings(matches))
	if got[x].GamesPlayed != 11 {
		t.Errorf("X games played = %d, want 11 (K should have been EloKSteady for match 11)", got[x].GamesPlayed)
	}
	if got[y].GamesPlayed != 1 {
		t.Errorf("Y games played = %d, want 1 (K should have been EloKProvisional for match 11)", got[y].GamesPlayed)
	}
	assertRating(t, "X after match 11", got[x].Rating, 1673.959164447)
	assertRating(t, "Y after match 11", got[y].Rating, 1483.104124232)
}

// TestComputeEloStandings_SortsChronologicallyRegardlessOfInputOrder covers
// case (d): the same two matches (a win, then a draw) produce a different
// final rating depending on which one is treated as "first" — the Elo
// update is nonlinear, so applying a win then a draw does not commute with
// applying a draw then a win. Feeding the *slice* in reverse (draw first,
// win second) must still produce the win-then-draw result, proving
// ComputeEloStandings — not the caller — is what puts them back in
// chronological (Date, then CreatedAt, then ID) order before replaying.
//
// Hand-computed win-then-draw (the correct chronological order, day1 then
// day2): after the day-1 win, A=1530, B=1470 (both provisional K=60, 1 game
// each). The day-2 draw: E_A = 1/(1+10^((1470-1530)/400)) ≈ 0.585055,
// E_B ≈ 0.414945; K still 60 each (1 < 10). delta_A = 60*(0.5-0.585055) ≈
// -5.1033, delta_B = 60*(0.5-0.414945) ≈ +5.1033 -> A ≈ 1524.8701,
// B ≈ 1475.1299. Feeding the draw first and the win second (the WRONG
// order) instead yields A=1530/B=1470 (the draw between two equal-rated
// players is a zero-sum no-op, then the win reduces to the exact same
// single-match computation as TestComputeEloStandings_FirstMatchBetween
// NewPlayers) — a different, and wrong, result this test also guards
// against by asserting the correct one.
func TestComputeEloStandings_SortsChronologicallyRegardlessOfInputOrder(t *testing.T) {
	alice, bob := uuid.New(), uuid.New()
	day1 := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, time.March, 2, 0, 0, 0, 0, time.UTC)

	winMatch := eloMatch(day1, day1,
		[]models.PlayerCustom{{ID: alice, Name: "alice"}}, 2,
		[]models.PlayerCustom{{ID: bob, Name: "bob"}}, 0,
	)
	drawMatch := eloMatch(day2, day2,
		[]models.PlayerCustom{{ID: alice, Name: "alice"}}, 1,
		[]models.PlayerCustom{{ID: bob, Name: "bob"}}, 1,
	)

	// Deliberately supplied out of chronological order: the draw (day2)
	// before the win (day1).
	got := eloRowsByID(ComputeEloStandings([]models.MatchWithDetails{drawMatch, winMatch}))

	assertRating(t, "alice", got[alice].Rating, 1524.8700792797)
	assertRating(t, "bob", got[bob].Rating, 1475.1299207203)
	if got[alice].GamesPlayed != 2 || got[bob].GamesPlayed != 2 {
		t.Errorf("games played = alice:%d bob:%d, want 2/2", got[alice].GamesPlayed, got[bob].GamesPlayed)
	}

	// Confirm the natural (already-sorted) order agrees exactly, as a cross-
	// check that the assertion above isn't accidentally pinning the wrong
	// (unsorted-replay) numbers.
	gotSorted := eloRowsByID(ComputeEloStandings([]models.MatchWithDetails{winMatch, drawMatch}))
	assertRating(t, "alice (already sorted)", gotSorted[alice].Rating, got[alice].Rating)
	assertRating(t, "bob (already sorted)", gotSorted[bob].Rating, got[bob].Rating)
}

// TestComputeEloStandings_SkipsUnplayedMatch mirrors
// TestComputePointsStandings_SkipsMatchWithAnEmptyTeam /
// TestComputeSeasons_IncludesFutureScheduledSeason's sibling test for the
// points aggregator: a scheduled match with no roster yet (or fewer than two
// teams) must not move any rating at all, the same "is this match played"
// guard ComputePointsStandings applies.
func TestComputeEloStandings_SkipsUnplayedMatch(t *testing.T) {
	alice := uuid.New()
	day := time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC)

	upcoming := models.MatchWithDetails{
		ID:   uuid.New(),
		Date: models.Date(day),
		Teams: []models.TeamWithPlayers{
			{ID: uuid.New(), Colour: "black", Players: []models.PlayerCustom{}},
			{ID: uuid.New(), Colour: "white", Players: []models.PlayerCustom{}},
		},
	}
	played := eloMatch(day, day,
		[]models.PlayerCustom{{ID: alice, Name: "alice"}}, 1,
		[]models.PlayerCustom{{ID: uuid.New(), Name: "bob"}}, 0,
	)

	rows := ComputeEloStandings([]models.MatchWithDetails{upcoming, played})
	if len(rows) != 2 {
		t.Fatalf("rows = %+v, want exactly the 2 players from the played match", rows)
	}
}

// TestComputeEloStandings_SortOrder_RatingDescNameAsc mirrors
// TestComputePointsStandings_SortOrder's own convention: rating descending,
// name ascending as the tie-break.
func TestComputeEloStandings_SortOrder_RatingDescNameAsc(t *testing.T) {
	alice, bob := uuid.New(), uuid.New()
	day := time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC)

	// alice beats bob: alice ends above EloStartingRating, bob below.
	match := eloMatch(day, day,
		[]models.PlayerCustom{{ID: alice, Name: "alice"}}, 1,
		[]models.PlayerCustom{{ID: bob, Name: "bob"}}, 0,
	)

	rows := ComputeEloStandings([]models.MatchWithDetails{match})
	if len(rows) != 2 || rows[0].PlayerID != alice || rows[1].PlayerID != bob {
		t.Fatalf("rows = %+v, want alice (winner) ranked above bob", rows)
	}
}
