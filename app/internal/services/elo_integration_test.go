package services_test

import (
	"math"
	"testing"
	"time"

	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/google/uuid"
)

// TestGetEloStandings_Integration_ReplaysHistoryChronologicallyAndTagsMembership
// exercises the whole wiring GetEloStandings adds on top of the pure
// ComputeEloStandings: loading a group's real match history via
// MatchService.GetMatchesDetails (which returns newest-first — see its own
// `ORDER BY match_date DESC`), filtering to completed matches, sorting back
// into chronological order, running the aggregator, and tagging IsMember
// from real group membership.
//
// Two real matches are created five and three days ago respectively (alice
// beats bob 1-0, then they draw 1-1) — the exact scenario, and the exact
// expected numbers, as the pure-function test
// TestComputeEloStandings_SortsChronologicallyRegardlessOfInputOrder's
// "correct chronological order" case, since both replay the identical
// win-then-draw sequence. carol also appears on a roster (an earlier match,
// so her presence doesn't disturb alice/bob's numbers) but is deliberately
// never added to the group, to prove IsMember is tagged from real,
// current membership rather than defaulting to true for anyone who ever
// played.
func TestGetEloStandings_Integration_ReplaysHistoryChronologicallyAndTagsMembership(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	teamService := services.NewTeamService(tx)
	playerService := services.NewPlayerService(tx)
	matchService := services.NewMatchService(tx)
	membershipService := services.NewGroupMembershipService(tx)
	standingsService := services.NewStandingsService(tx, membershipService)

	group, err := groupService.CreateGroup("Zzz Elo Integration", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	teams, err := teamService.GetTeamsByGroupID(group.ID)
	if err != nil {
		t.Fatalf("failed to load group's teams: %v", err)
	}
	black, white := teams[0], teams[1]

	alice, err := playerService.CreatePlayer("Zzz Elo Alice " + uuid.NewString())
	if err != nil {
		t.Fatalf("failed to create alice: %v", err)
	}
	bob, err := playerService.CreatePlayer("Zzz Elo Bob " + uuid.NewString())
	if err != nil {
		t.Fatalf("failed to create bob: %v", err)
	}
	carol, err := playerService.CreatePlayer("Zzz Elo Carol " + uuid.NewString())
	if err != nil {
		t.Fatalf("failed to create carol: %v", err)
	}
	dave, err := playerService.CreatePlayer("Zzz Elo Dave " + uuid.NewString())
	if err != nil {
		t.Fatalf("failed to create dave: %v", err)
	}

	if err := membershipService.AddPlayerToGroupWithRole(group.ID, alice, models.RoleAdmin); err != nil {
		t.Fatalf("failed to add alice to group: %v", err)
	}
	if err := membershipService.AddPlayerToGroupWithRole(group.ID, bob, models.RoleMember); err != nil {
		t.Fatalf("failed to add bob to group: %v", err)
	}
	// carol deliberately never joins the group.

	now := time.Now()

	// An early match (10 days ago) between carol and dave, neither of whom
	// is alice or bob — purely to prove a non-member's (carol's) historical
	// rating still surfaces, tagged IsMember: false, without disturbing
	// alice/bob's own numbers below (which reuse the exact scenario, and
	// therefore the exact expected values, of
	// TestComputeEloStandings_SortsChronologicallyRegardlessOfInputOrder).
	earlyMatchID, err := matchService.CreateMatch(services.MatchSpec{Date: models.Date(now.AddDate(0, 0, -10))}, group.ID)
	if err != nil {
		t.Fatalf("failed to create early match: %v", err)
	}
	if err := matchService.UpdateMatch(earlyMatchID, group.ID, []models.TeamWithPlayers{
		{ID: black.ID, Players: []models.PlayerCustom{{ID: carol, GoalsScored: 0}}},
		{ID: white.ID, Players: []models.PlayerCustom{{ID: dave, GoalsScored: 2}}},
	}); err != nil {
		t.Fatalf("failed to compose early match roster: %v", err)
	}

	// Match 1 (5 days ago): alice beats bob 1-0.
	match1ID, err := matchService.CreateMatch(services.MatchSpec{Date: models.Date(now.AddDate(0, 0, -5))}, group.ID)
	if err != nil {
		t.Fatalf("failed to create match 1: %v", err)
	}
	if err := matchService.UpdateMatch(match1ID, group.ID, []models.TeamWithPlayers{
		{ID: black.ID, Players: []models.PlayerCustom{{ID: alice, GoalsScored: 1}}},
		{ID: white.ID, Players: []models.PlayerCustom{{ID: bob, GoalsScored: 0}}},
	}); err != nil {
		t.Fatalf("failed to compose match 1 roster: %v", err)
	}

	// Match 2 (3 days ago, i.e. strictly after match 1): alice and bob draw 1-1.
	match2ID, err := matchService.CreateMatch(services.MatchSpec{Date: models.Date(now.AddDate(0, 0, -3))}, group.ID)
	if err != nil {
		t.Fatalf("failed to create match 2: %v", err)
	}
	if err := matchService.UpdateMatch(match2ID, group.ID, []models.TeamWithPlayers{
		{ID: black.ID, Players: []models.PlayerCustom{{ID: alice, GoalsScored: 1}}},
		{ID: white.ID, Players: []models.PlayerCustom{{ID: bob, GoalsScored: 1}}},
	}); err != nil {
		t.Fatalf("failed to compose match 2 roster: %v", err)
	}

	rows, err := standingsService.GetEloStandings(group.ID)
	if err != nil {
		t.Fatalf("GetEloStandings returned error: %v", err)
	}

	byID := make(map[uuid.UUID]models.EloStandingRow, len(rows))
	for _, r := range rows {
		byID[r.PlayerID] = r
	}

	// alice/bob's ratings reflect exactly the win-then-draw scenario in
	// TestComputeEloStandings_SortsChronologicallyRegardlessOfInputOrder,
	// replayed here through the real read/write path instead of hand-built
	// fixtures. This assertion focuses on what only the service layer adds:
	// that the SQL-level newest-first order (GetMatchesDetails' own
	// `ORDER BY match_date DESC`) got corrected back to chronological order
	// before replay at all. If it hadn't been, the draw would have been
	// applied before the win, producing a different (and wrong) number —
	// see that pure test's "WRONG order" comment for the exact contrast.
	const epsilon = 0.01
	assertClose := func(label string, got, want float64) {
		t.Helper()
		if math.Abs(got-want) > epsilon {
			t.Errorf("%s rating = %v, want %v (±%v)", label, got, want, epsilon)
		}
	}

	if got := byID[bob].GamesPlayed; got != 2 {
		t.Errorf("bob games played = %d, want 2", got)
	}
	assertClose("bob", byID[bob].Rating, 1475.1299207203)
	if !byID[bob].IsMember {
		t.Errorf("bob.IsMember = false, want true (bob is a current group member)")
	}

	if got := byID[carol].GamesPlayed; got != 1 {
		t.Errorf("carol games played = %d, want 1", got)
	}
	if byID[carol].IsMember {
		t.Errorf("carol.IsMember = true, want false (carol never joined the group)")
	}

	if !byID[alice].IsMember {
		t.Errorf("alice.IsMember = false, want true (alice is a current group member)")
	}
	if got := byID[alice].GamesPlayed; got != 2 {
		t.Errorf("alice games played = %d, want 2", got)
	}
	assertClose("alice", byID[alice].Rating, 1524.8700792797)
}

// TestGetEloStandings_Integration_IgnoresUpcomingScheduledMatch mirrors the
// standings package's own guarantee (see
// TestComputePointsStandings_IgnoresScheduledMatchWithNoAssignments): a
// scheduled match that hasn't kicked off yet, and therefore has no roster,
// must not appear in the Elo replay at all — same reasoning as
// FilterCompletedMatches/ComputeEloStandings' own "both teams have players"
// guard, just exercised end to end through the real GetMatchesDetails read
// path this time.
func TestGetEloStandings_Integration_IgnoresUpcomingScheduledMatch(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	playerService := services.NewPlayerService(tx)
	matchService := services.NewMatchService(tx)
	membershipService := services.NewGroupMembershipService(tx)
	standingsService := services.NewStandingsService(tx, membershipService)

	group, err := groupService.CreateGroup("Zzz Elo Upcoming", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	alice, err := playerService.CreatePlayer("Zzz Elo Upcoming Alice " + uuid.NewString())
	if err != nil {
		t.Fatalf("failed to create alice: %v", err)
	}
	if err := membershipService.AddPlayerToGroupWithRole(group.ID, alice, models.RoleAdmin); err != nil {
		t.Fatalf("failed to add alice to group: %v", err)
	}

	kickoff := time.Now().Add(48 * time.Hour)
	opensAt := time.Now().Add(-time.Hour)
	maxPlayers := 10
	if _, err := matchService.CreateMatch(services.MatchSpec{
		ScheduledAt:         &kickoff,
		RegistrationOpensAt: &opensAt,
		MaxPlayers:          &maxPlayers,
	}, group.ID); err != nil {
		t.Fatalf("failed to create the scheduled match: %v", err)
	}

	rows, err := standingsService.GetEloStandings(group.ID)
	if err != nil {
		t.Fatalf("GetEloStandings returned error: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("rows = %+v, want none: an upcoming match with no roster must not move any rating", rows)
	}
}
