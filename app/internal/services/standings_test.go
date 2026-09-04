package services

import (
	"testing"

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
