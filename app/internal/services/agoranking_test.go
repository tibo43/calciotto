package services

import (
	"fmt"
	"math"
	"testing"

	"app/internal/models"

	"github.com/google/uuid"
)

// agorankingMatch builds a match between two sides; scoreA/scoreB decide the result.
func agorankingMatch(scoreA, scoreB int, sideA, sideB []models.PlayerCustom) models.MatchWithDetails {
	return newMatch(
		newTeam(uuid.New(), scoreA, sideA...),
		newTeam(uuid.New(), scoreB, sideB...),
	)
}

func players(names ...string) map[string]models.PlayerCustom {
	m := make(map[string]models.PlayerCustom, len(names))
	for _, n := range names {
		m[n] = newPlayer(uuid.New(), n, 0)
	}
	return m
}

func side(p map[string]models.PlayerCustom, names ...string) []models.PlayerCustom {
	out := make([]models.PlayerCustom, 0, len(names))
	for _, n := range names {
		out = append(out, p[n])
	}
	return out
}

func rowNames(rows []models.AgorankingRow) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.Name
	}
	return out
}

func equalNames(t *testing.T, label string, got []models.AgorankingRow, want ...string) {
	t.Helper()
	names := rowNames(got)
	if len(names) != len(want) {
		t.Fatalf("%s = %v, want %v", label, names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("%s = %v, want %v", label, names, want)
		}
	}
}

func TestAgorankingN_RoundsUp(t *testing.T) {
	cases := map[int]int{0: 0, -1: 0, 1: 1, 2: 1, 3: 1, 4: 2, 10: 4, 11: 4, 12: 4, 13: 5, 40: 14}
	for matches, want := range cases {
		if got := AgorankingN(matches); got != want {
			t.Errorf("AgorankingN(%d) = %d, want %d", matches, got, want)
		}
	}
}

func TestAgorankingRating(t *testing.T) {
	// 14 points in 8 matches, N = 4, average 1.318: (14 + 5.272) / 12.
	got := AgorankingRating(14, 8, 4, 1.318)
	if want := (14 + 4*1.318) / 12; math.Abs(got-want) > 1e-12 {
		t.Errorf("rating = %v, want %v", got, want)
	}
	if got := AgorankingRating(0, 0, 0, 0); got != 0 {
		t.Errorf("rating with nothing to divide by = %v, want 0", got)
	}
}

func TestComputeAgoranking_EmptySeason(t *testing.T) {
	got := ComputeAgoranking(nil)
	if got.Matches != 0 || got.N != 0 || got.GroupAverage != 0 {
		t.Errorf("empty season = %+v, want zeroes", got)
	}
	if got.Ranked == nil || got.Unranked == nil || len(got.Ranked) != 0 || len(got.Unranked) != 0 {
		t.Errorf("empty season lists = %v / %v, want two empty non-nil slices", got.Ranked, got.Unranked)
	}
}

func TestComputeAgoranking_IgnoresMatchesWithoutBothRosters(t *testing.T) {
	p := players("a", "b")
	matches := []models.MatchWithDetails{
		agorankingMatch(1, 0, side(p, "a"), nil),
		agorankingMatch(1, 0, side(p, "a"), side(p, "b")),
	}
	got := ComputeAgoranking(matches)
	if got.Matches != 1 || got.N != 1 {
		t.Errorf("Matches/N = %d/%d, want 1/1", got.Matches, got.N)
	}
	equalNames(t, "ranked", got.Ranked, "a", "b")
}

func TestComputeAgoranking_AverageThresholdAndOrder(t *testing.T) {
	p := players("ace", "steady", "rookie", "bench", "loser")
	// Four matches, so N = ceil(4/3) = 2.
	matches := []models.MatchWithDetails{
		agorankingMatch(2, 0, side(p, "ace", "steady"), side(p, "loser", "bench")),
		agorankingMatch(1, 1, side(p, "ace", "rookie"), side(p, "steady", "loser")),
		agorankingMatch(3, 1, side(p, "ace"), side(p, "loser")),
		agorankingMatch(0, 2, side(p, "loser"), side(p, "steady")),
	}
	got := ComputeAgoranking(matches)
	if got.Matches != 4 || got.N != 2 {
		t.Fatalf("Matches/N = %d/%d, want 4/2", got.Matches, got.N)
	}
	// Participations: ace 3, steady 3, rookie 1, bench 1, loser 4 = 12.
	// Points: ace 3+1+3=7, steady 3+1+3=7, rookie 1, bench 0, loser 0+1+0+0=1 = 16.
	if want := 16.0 / 12.0; math.Abs(got.GroupAverage-want) > 1e-12 {
		t.Errorf("GroupAverage = %v, want %v", got.GroupAverage, want)
	}
	// ace and steady tie exactly (7 points, 3 matches): the name decides.
	equalNames(t, "ranked", got.Ranked, "ace", "steady", "loser")
	equalNames(t, "unranked", got.Unranked, "rookie", "bench")

	ace := got.Ranked[0]
	want := (7 + 2*got.GroupAverage) / 5
	if math.Abs(ace.Rating-want) > 1e-12 || ace.Played != 3 || ace.Won != 2 || ace.Drawn != 1 || ace.Points != 7 {
		t.Errorf("ace = %+v, want rating %v from 7 points in 3 matches", ace, want)
	}
}

func TestComputeAgoranking_ThresholdIsInclusive(t *testing.T) {
	p := players("a", "b", "c", "d")
	// Three matches: N = 1, so a single match is enough to be ranked.
	matches := []models.MatchWithDetails{
		agorankingMatch(1, 0, side(p, "a"), side(p, "b")),
		agorankingMatch(1, 0, side(p, "a"), side(p, "b")),
		agorankingMatch(1, 0, side(p, "c"), side(p, "d")),
	}
	got := ComputeAgoranking(matches)
	if got.N != 1 || len(got.Unranked) != 0 || len(got.Ranked) != 4 {
		t.Fatalf("N=%d ranked=%v unranked=%v, want everyone ranked with N=1", got.N, rowNames(got.Ranked), rowNames(got.Unranked))
	}
}

func TestSortAgorankingRanked_UnroundedValueFirst(t *testing.T) {
	// Both display as 1.66: the full value decides, before matches played.
	rows := []models.AgorankingRow{
		{Name: "low", Played: 10, Rating: 1.6612},
		{Name: "high", Played: 4, Rating: 1.6649},
	}
	if a, b := fmt.Sprintf("%.2f", rows[0].Rating), fmt.Sprintf("%.2f", rows[1].Rating); a != b {
		t.Fatalf("fixture should round to the same value, got %s and %s", a, b)
	}
	sortAgorankingRanked(rows)
	equalNames(t, "ranked", rows, "high", "low")
}

func TestSortAgorankingRanked_ExactTie(t *testing.T) {
	rows := []models.AgorankingRow{
		{Name: "bob", Played: 4, Rating: 1.5},
		{Name: "zed", Played: 6, Rating: 1.5},
		{Name: "amy", Played: 4, Rating: 1.5 + 1e-12}, // float noise is still a tie
	}
	sortAgorankingRanked(rows)
	equalNames(t, "ranked", rows, "zed", "amy", "bob")
}

func TestSortAgorankingUnranked(t *testing.T) {
	rows := []models.AgorankingRow{
		{Name: "one", Played: 1, Points: 3},
		{Name: "twoLow", Played: 2, Points: 1},
		{Name: "twoHigh", Played: 2, Points: 6},
		{Name: "TwoHighToo", Played: 2, Points: 6},
	}
	sortAgorankingUnranked(rows)
	// Names compare without case: "twoHigh" < "TwoHighToo".
	equalNames(t, "unranked", rows, "twoHigh", "TwoHighToo", "twoLow", "one")
}

func TestComputeAgoranking_ExactTieMoreMatchesFirst(t *testing.T) {
	p := players("few", "many", "opp1", "opp2", "opp3", "opp4")
	// Group average is exactly 1.5 here (every match is won by one side of
	// equal size, so half the participations get 3 points), and both "few"
	// (3 points in 2 matches) and "many" (6 points in 4 matches) are at 1.5
	// per match, so their ratings are equal: the one with more matches comes first.
	matches := []models.MatchWithDetails{
		agorankingMatch(1, 0, side(p, "few"), side(p, "opp1")),
		agorankingMatch(0, 1, side(p, "few"), side(p, "opp2")),
		agorankingMatch(1, 0, side(p, "many"), side(p, "opp3")),
		agorankingMatch(1, 0, side(p, "many"), side(p, "opp4")),
		agorankingMatch(0, 1, side(p, "many"), side(p, "opp3")),
		agorankingMatch(0, 1, side(p, "many"), side(p, "opp4")),
	}
	got := ComputeAgoranking(matches)
	if got.N != 2 || math.Abs(got.GroupAverage-1.5) > 1e-12 {
		t.Fatalf("N/avg = %d/%v, want 2/1.5", got.N, got.GroupAverage)
	}
	if got.Ranked[0].Name != "many" || got.Ranked[1].Name != "few" {
		t.Fatalf("ranked = %v, want many before few", rowNames(got.Ranked))
	}
}

func TestComputeAgoranking_UnrankedOrder(t *testing.T) {
	p := players("a", "b", "c", "d", "e", "f")
	var matches []models.MatchWithDetails
	for i := 0; i < 7; i++ {
		matches = append(matches, agorankingMatch(1, 0, side(p, "a"), side(p, "b")))
	}
	matches = append(matches,
		agorankingMatch(1, 0, side(p, "c"), side(p, "d")),
		agorankingMatch(1, 1, side(p, "c"), side(p, "e")),
		agorankingMatch(1, 0, side(p, "e"), side(p, "f")),
	)
	got := ComputeAgoranking(matches)
	if got.Matches != 10 || got.N != 4 {
		t.Fatalf("Matches/N = %d/%d, want 10/4", got.Matches, got.N)
	}
	equalNames(t, "ranked", got.Ranked, "a", "b")
	// c and e: 2 matches, 4 points each (name decides); d and f: 1 match, 0 points.
	equalNames(t, "unranked", got.Unranked, "c", "e", "d", "f")
}
