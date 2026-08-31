package services

import (
	"testing"
	"time"

	"app/internal/models"

	"github.com/google/uuid"
)

// upcomingMatchDetails is a match that has been scheduled and, per the normal state
// of one, has no team assignments yet: the group's two teams are still there as
// empty shells (that is what GetMatchesDetails backfills), which is exactly the
// shape the aggregators have to ignore.
func upcomingMatchDetails(date models.Date, kickoff time.Time) models.MatchWithDetails {
	maxPlayers, count := 16, 12
	return models.MatchWithDetails{
		ID:                  uuid.New(),
		Date:                date,
		ScheduledAt:         &kickoff,
		RegistrationOpensAt: &kickoff,
		MaxPlayers:          &maxPlayers,
		RegistrationCount:   &count,
		Teams: []models.TeamWithPlayers{
			{ID: uuid.New(), Colour: "black", Players: []models.PlayerCustom{}},
			{ID: uuid.New(), Colour: "white", Players: []models.PlayerCustom{}},
		},
	}
}

// TestComputePointsStandings_IgnoresScheduledMatchWithNoAssignments is the
// guarantee the whole "sign-ups are not MatchPlayer rows" design was built
// around: an upcoming match, however many players have signed up for it, must
// not show up in the standings as a 0-0 draw for anybody. Nothing in
// ComputePointsStandings looks at the scheduling fields — the empty-team check
// it already had is what does the work — so this is a regression test on that
// existing check rather than on new logic.
func TestComputePointsStandings_IgnoresScheduledMatchWithNoAssignments(t *testing.T) {
	alice, bob := uuid.New(), uuid.New()
	played := newMatch(
		newTeam(uuid.New(), "black", 2, newPlayer(alice, "alice", 2)),
		newTeam(uuid.New(), "white", 1, newPlayer(bob, "bob", 1)),
	)
	upcoming := upcomingMatchDetails(
		models.Date(time.Date(2025, time.October, 12, 0, 0, 0, 0, time.UTC)),
		time.Date(2025, time.October, 12, 19, 30, 0, 0, time.UTC),
	)

	got := pointsRowsByID(ComputePointsStandings([]models.MatchWithDetails{played, upcoming}))
	if len(got) != 2 {
		t.Fatalf("standings contain %d players, want only the two who actually played: %+v", len(got), got)
	}
	if got[alice].Played != 1 || got[alice].Points != 3 {
		t.Errorf("alice = %+v, want 1 played / 3 points (the scheduled match must not count)", got[alice])
	}
	if got[bob].Played != 1 || got[bob].Points != 0 {
		t.Errorf("bob = %+v, want 1 played / 0 points", got[bob])
	}
}

// TestComputeScorers_IgnoresScheduledMatchWithNoAssignments is the same
// guarantee on the other aggregator: an upcoming match has no goals and no
// assigned players, so it must not inflate anybody's "played" count either.
func TestComputeScorers_IgnoresScheduledMatchWithNoAssignments(t *testing.T) {
	alice := uuid.New()
	played := newMatch(
		newTeam(uuid.New(), "black", 2, newPlayer(alice, "alice", 2)),
		newTeam(uuid.New(), "white", 0),
	)
	upcoming := upcomingMatchDetails(
		models.Date(time.Date(2025, time.October, 12, 0, 0, 0, 0, time.UTC)),
		time.Date(2025, time.October, 12, 19, 30, 0, 0, time.UTC),
	)

	rows := ComputeScorers([]models.MatchWithDetails{played, upcoming})
	if len(rows) != 1 || rows[0].PlayerID != alice || rows[0].Played != 1 || rows[0].Goals != 2 {
		t.Errorf("scorers = %+v, want only alice with 1 played / 2 goals", rows)
	}
}

// TestComputeSeasons_IncludesFutureScheduledSeason pins a *decision*, not an
// oversight: a match scheduled into a season nobody has played in yet still
// contributes that season's label. Removing it would look tidier and would be
// wrong — GET /matches/details is filtered by the season the frontend has
// selected, so a season missing from this list makes every upcoming match in it
// unreachable in the UI. If you came here to "fix" this, read ComputeSeasons'
// doc comment first.
func TestComputeSeasons_IncludesFutureScheduledSeason(t *testing.T) {
	playedLastSeason := newMatch(
		newTeam(uuid.New(), "black", 1, newPlayer(uuid.New(), "alice", 1)),
		newTeam(uuid.New(), "white", 0, newPlayer(uuid.New(), "bob", 0)),
	)
	playedLastSeason.Date = models.Date(time.Date(2025, time.May, 4, 0, 0, 0, 0, time.UTC)) // 2024-2025

	upcomingNextSeason := upcomingMatchDetails(
		models.Date(time.Date(2025, time.September, 7, 0, 0, 0, 0, time.UTC)), // 2025-2026
		time.Date(2025, time.September, 7, 19, 30, 0, 0, time.UTC),
	)

	seasons := ComputeSeasons([]models.MatchWithDetails{playedLastSeason, upcomingNextSeason})
	if len(seasons) != 2 || seasons[0] != "2024-2025" || seasons[1] != "2025-2026" {
		t.Errorf("ComputeSeasons = %v, want [2024-2025 2025-2026] — the future season is included on purpose", seasons)
	}
}
