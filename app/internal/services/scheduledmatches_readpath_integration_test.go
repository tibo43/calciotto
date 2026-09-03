package services_test

import (
	"testing"
	"time"

	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Fixed timestamps rather than time.Now() offsets: the assertions below compare
// them for equality after a round-trip through a timestamptz column, and a
// wall-clock value would also make the test's own registration window depend on
// when it happens to run. The kick-off is far enough in the future that the
// registration window is open (RegistrationWindowError treats kick-off as a
// hard backstop), and the opening time is in the past so sign-ups are allowed.
var (
	testKickoff = time.Date(2030, time.October, 5, 19, 30, 0, 0, time.UTC)
	testOpensAt = time.Date(2020, time.January, 1, 8, 0, 0, 0, time.UTC)
)

type scheduledReadEnv struct {
	tx            *gorm.DB
	matches       *services.MatchService
	registrations *services.MatchRegistrationService
	standings     *services.StandingsService
	groupID       uuid.UUID
	teams         []models.Team
	players       []uuid.UUID
}

func newScheduledReadEnv(t *testing.T, label string, playerCount int) *scheduledReadEnv {
	t.Helper()

	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	teamService := services.NewTeamService(tx)
	playerService := services.NewPlayerService(tx)
	membershipService := services.NewGroupMembershipService(tx)

	group, err := groupService.CreateGroup("Zzz Scheduled Read "+label, services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	teams, err := teamService.GetTeamsByGroupID(group.ID)
	if err != nil || len(teams) != 2 {
		t.Fatalf("failed to load group teams: err=%v teams=%+v", err, teams)
	}

	env := &scheduledReadEnv{
		tx:            tx,
		matches:       services.NewMatchService(tx),
		registrations: services.NewMatchRegistrationService(tx),
		standings:     services.NewStandingsService(tx, membershipService),
		groupID:       group.ID,
		teams:         teams,
	}
	for i := 0; i < playerCount; i++ {
		name := "Zzz Scheduled Read " + label + " Player " + string(rune('A'+i))
		playerID, err := playerService.CreatePlayer(name)
		if err != nil {
			t.Fatalf("failed to create player %s: %v", name, err)
		}
		if err := membershipService.AddPlayerToGroup(group.ID, playerID); err != nil {
			t.Fatalf("failed to add player %s to the group: %v", name, err)
		}
		env.players = append(env.players, playerID)
	}
	return env
}

func findMatch(matches []models.MatchWithDetails, id uuid.UUID) *models.MatchWithDetails {
	for i := range matches {
		if matches[i].ID == id {
			return &matches[i]
		}
	}
	return nil
}

// TestMatchesDetails_Integration_ScheduledMatchWithSignUpsButNoTeams is the
// case the read path is easiest to get wrong: a scheduled match nobody has been
// assigned to yet. That is not an edge case, it is the *normal* state of an
// upcoming match — it has no match_players rows at all, so the LEFT JOIN yields
// a single all-NULL-team row and the teams come from the backfill path rather
// than from the join. Both read functions must still report the schedule and the
// sign-up count.
func TestMatchesDetails_Integration_ScheduledMatchWithSignUpsButNoTeams(t *testing.T) {
	env := newScheduledReadEnv(t, "NoTeams", 3)

	maxPlayers := 16
	scheduledID, err := env.matches.CreateMatch(services.MatchSpec{
		ScheduledAt:         &testKickoff,
		RegistrationOpensAt: &testOpensAt,
		MaxPlayers:          &maxPlayers,
	}, env.groupID)
	if err != nil {
		t.Fatalf("failed to create the scheduled match: %v", err)
	}

	// An ordinary match alongside it, to prove the two read back differently.
	plainDate := models.Date(time.Date(2025, time.October, 5, 0, 0, 0, 0, time.UTC))
	plainID, err := env.matches.CreateMatch(services.MatchSpec{Date: plainDate}, env.groupID)
	if err != nil {
		t.Fatalf("failed to create the plain match: %v", err)
	}

	for _, playerID := range env.players {
		if err := env.registrations.Register(scheduledID, playerID); err != nil {
			t.Fatalf("failed to register player %s: %v", playerID, err)
		}
	}

	assertScheduled := func(source string, match *models.MatchWithDetails) {
		t.Helper()
		if match == nil {
			t.Fatalf("%s did not return the scheduled match at all", source)
		}
		if match.ScheduledAt == nil || !match.ScheduledAt.Equal(testKickoff) {
			t.Errorf("%s: ScheduledAt = %v, want %v", source, match.ScheduledAt, testKickoff)
		}
		if match.RegistrationOpensAt == nil || !match.RegistrationOpensAt.Equal(testOpensAt) {
			t.Errorf("%s: RegistrationOpensAt = %v, want %v", source, match.RegistrationOpensAt, testOpensAt)
		}
		if match.RegistrationsClosedAt != nil {
			t.Errorf("%s: RegistrationsClosedAt = %v, want nil while sign-ups are open", source, match.RegistrationsClosedAt)
		}
		if match.MaxPlayers == nil || *match.MaxPlayers != maxPlayers {
			t.Errorf("%s: MaxPlayers = %v, want %d", source, match.MaxPlayers, maxPlayers)
		}
		if match.RegistrationCount == nil || *match.RegistrationCount != len(env.players) {
			t.Errorf("%s: RegistrationCount = %v, want %d", source, match.RegistrationCount, len(env.players))
		}
		// Date is derived from the kick-off day, so ordering and seasons keep
		// working without the caller ever supplying it.
		if got, want := match.Date.String(), "2030-10-05"; got != want {
			t.Errorf("%s: Date = %s, want %s", source, got, want)
		}
		// The teams are still there, as empty shells: a scheduled match must be
		// renderable before anyone has been assigned.
		if len(match.Teams) != 2 {
			t.Fatalf("%s: match has %d teams, want the group's 2 empty shells: %+v", source, len(match.Teams), match.Teams)
		}
		for _, team := range match.Teams {
			if len(team.Players) != 0 || team.Score != 0 {
				t.Errorf("%s: team %s should have no players and no score, got %+v", source, team.Name, team)
			}
		}
	}

	assertUnscheduled := func(source string, match *models.MatchWithDetails) {
		t.Helper()
		if match == nil {
			t.Fatalf("%s did not return the plain match at all", source)
		}
		if match.ScheduledAt != nil || match.RegistrationOpensAt != nil ||
			match.RegistrationsClosedAt != nil || match.MaxPlayers != nil {
			t.Errorf("%s: plain match carries scheduling data: %+v", source, match)
		}
		// nil, not 0: an unscheduled match has no sign-up list at all, and the
		// omitempty on RegistrationCount is what keeps its JSON unchanged.
		if match.RegistrationCount != nil {
			t.Errorf("%s: plain match RegistrationCount = %v, want nil", source, match.RegistrationCount)
		}
	}

	list, err := env.matches.GetMatchesDetails(env.groupID, "")
	if err != nil {
		t.Fatalf("GetMatchesDetails returned error: %v", err)
	}
	assertScheduled("GetMatchesDetails", findMatch(list, scheduledID))
	assertUnscheduled("GetMatchesDetails", findMatch(list, plainID))

	byID, err := env.matches.GetMatchDetailsByID(scheduledID, env.groupID)
	if err != nil {
		t.Fatalf("GetMatchDetailsByID(scheduled) returned error: %v", err)
	}
	assertScheduled("GetMatchDetailsByID", byID)

	plainByID, err := env.matches.GetMatchDetailsByID(plainID, env.groupID)
	if err != nil {
		t.Fatalf("GetMatchDetailsByID(plain) returned error: %v", err)
	}
	assertUnscheduled("GetMatchDetailsByID", plainByID)

	// The future kick-off sorts first, which is what an "upcoming match" needs
	// and comes for free from the existing ORDER BY match_date DESC.
	if len(list) == 0 || list[0].ID != scheduledID {
		t.Errorf("GetMatchesDetails put %v first, want the future match %s", list[0].ID, scheduledID)
	}

	// Closing sign-ups is visible on the read path too — that timestamp is how
	// the frontend will know to stop offering "Participate".
	if err := env.registrations.CloseRegistrations(scheduledID, env.groupID); err != nil {
		t.Fatalf("CloseRegistrations returned error: %v", err)
	}
	closedByID, err := env.matches.GetMatchDetailsByID(scheduledID, env.groupID)
	if err != nil {
		t.Fatalf("GetMatchDetailsByID after closing returned error: %v", err)
	}
	if closedByID.RegistrationsClosedAt == nil {
		t.Error("GetMatchDetailsByID: RegistrationsClosedAt is still nil after CloseRegistrations")
	}
	closedList, err := env.matches.GetMatchesDetails(env.groupID, "")
	if err != nil {
		t.Fatalf("GetMatchesDetails after closing returned error: %v", err)
	}
	if closed := findMatch(closedList, scheduledID); closed == nil || closed.RegistrationsClosedAt == nil {
		t.Errorf("GetMatchesDetails: RegistrationsClosedAt is still nil after CloseRegistrations: %+v", closed)
	}
}

// TestMatchesDetails_Integration_SignUpCountDoesNotMultiplyScores guards the
// specific trap the sign-up count was designed around: match_registrations is a
// second one-to-many on matches, so joining it into the existing
// matches → match_players → teams/players query would multiply the rows and
// inflate every score and goal total derived from them. The count is fetched
// separately for that reason, and this test is what would fail if someone folded
// it back into the join.
func TestMatchesDetails_Integration_SignUpCountDoesNotMultiplyScores(t *testing.T) {
	env := newScheduledReadEnv(t, "NoFanOut", 3)

	maxPlayers := 16
	matchID, err := env.matches.CreateMatch(services.MatchSpec{
		ScheduledAt:         &testKickoff,
		RegistrationOpensAt: &testOpensAt,
		MaxPlayers:          &maxPlayers,
	}, env.groupID)
	if err != nil {
		t.Fatalf("failed to create the scheduled match: %v", err)
	}

	// Three sign-ups, then teams composed from two of them: the two counts are
	// deliberately different so a fan-out can't accidentally look correct.
	for _, playerID := range env.players {
		if err := env.registrations.Register(matchID, playerID); err != nil {
			t.Fatalf("failed to register player %s: %v", playerID, err)
		}
	}
	if err := env.matches.UpdateMatch(models.MatchWithDetails{
		ID: matchID,
		Teams: []models.TeamWithPlayers{
			{ID: env.teams[0].ID, Players: []models.PlayerCustom{{ID: env.players[0], GoalsScored: 2}}},
			{ID: env.teams[1].ID, Players: []models.PlayerCustom{{ID: env.players[1], GoalsScored: 1}}},
		},
	}); err != nil {
		t.Fatalf("UpdateMatch returned error: %v", err)
	}

	check := func(source string, match *models.MatchWithDetails) {
		t.Helper()
		if match == nil {
			t.Fatalf("%s did not return the match", source)
		}
		if match.RegistrationCount == nil || *match.RegistrationCount != 3 {
			t.Errorf("%s: RegistrationCount = %v, want 3", source, match.RegistrationCount)
		}
		total := 0
		for _, team := range match.Teams {
			if len(team.Players) != 1 {
				t.Errorf("%s: team %s has %d players, want 1 (a fan-out would duplicate them): %+v",
					source, team.Name, len(team.Players), team.Players)
			}
			total += team.Score
		}
		if total != 3 {
			t.Errorf("%s: combined score = %d, want 3 (2-1)", source, total)
		}
	}

	list, err := env.matches.GetMatchesDetails(env.groupID, "")
	if err != nil {
		t.Fatalf("GetMatchesDetails returned error: %v", err)
	}
	check("GetMatchesDetails", findMatch(list, matchID))

	byID, err := env.matches.GetMatchDetailsByID(matchID, env.groupID)
	if err != nil {
		t.Fatalf("GetMatchDetailsByID returned error: %v", err)
	}
	check("GetMatchDetailsByID", byID)
}

// TestStandings_Integration_ScheduledMatchWithSignUpsIsNotPlayed is the
// end-to-end version of the guarantee slice 1 was designed around: sign-ups are
// MatchRegistration rows, not MatchPlayer rows, so an upcoming match with a full
// sign-up list contributes nothing to the standings — no phantom 0-0 draw, no
// inflated "played" count.
//
// It also pins the deliberate exception: GET /standings/seasons *does* list the
// upcoming match's season, before it has been played. See ComputeSeasons' doc
// comment for why removing it would be a regression rather than a fix.
func TestStandings_Integration_ScheduledMatchWithSignUpsIsNotPlayed(t *testing.T) {
	env := newScheduledReadEnv(t, "NotPlayed", 2)
	alice, bob := env.players[0], env.players[1]

	// One real, played match in an earlier season.
	playedDate := models.Date(time.Date(2025, time.May, 4, 0, 0, 0, 0, time.UTC)) // 2024-2025
	playedID, err := env.matches.CreateMatch(services.MatchSpec{Date: playedDate}, env.groupID)
	if err != nil {
		t.Fatalf("failed to create the played match: %v", err)
	}
	if err := env.matches.UpdateMatch(models.MatchWithDetails{
		ID: playedID,
		Teams: []models.TeamWithPlayers{
			{ID: env.teams[0].ID, Players: []models.PlayerCustom{{ID: alice, GoalsScored: 2}}},
			{ID: env.teams[1].ID, Players: []models.PlayerCustom{{ID: bob, GoalsScored: 1}}},
		},
	}); err != nil {
		t.Fatalf("UpdateMatch returned error: %v", err)
	}

	// One scheduled match, in a *later* season, that both players signed up for
	// and that nobody has been assigned to.
	maxPlayers := 16
	scheduledID, err := env.matches.CreateMatch(services.MatchSpec{
		ScheduledAt:         &testKickoff, // 2030-10-05 → season 2030-2031
		RegistrationOpensAt: &testOpensAt,
		MaxPlayers:          &maxPlayers,
	}, env.groupID)
	if err != nil {
		t.Fatalf("failed to create the scheduled match: %v", err)
	}
	for _, playerID := range env.players {
		if err := env.registrations.Register(scheduledID, playerID); err != nil {
			t.Fatalf("failed to register player %s: %v", playerID, err)
		}
	}

	points, err := env.standings.GetPointsStandings(env.groupID, "")
	if err != nil {
		t.Fatalf("GetPointsStandings returned error: %v", err)
	}
	aliceRow := pointsRowByID(points, alice)
	if aliceRow == nil || aliceRow.Played != 1 || aliceRow.Won != 1 || aliceRow.Points != 3 {
		t.Errorf("alice = %+v, want 1 played / 1 won / 3 points — the scheduled match must not count", aliceRow)
	}
	bobRow := pointsRowByID(points, bob)
	if bobRow == nil || bobRow.Played != 1 || bobRow.Lost != 1 || bobRow.Points != 0 {
		t.Errorf("bob = %+v, want 1 played / 1 lost / 0 points", bobRow)
	}

	// Scoped to the upcoming match's own season, there is nothing to report at
	// all — the clearest statement of "not played yet".
	upcomingSeason, err := env.standings.GetPointsStandings(env.groupID, "2030-2031")
	if err != nil {
		t.Fatalf("GetPointsStandings(2030-2031) returned error: %v", err)
	}
	if len(upcomingSeason) != 0 {
		t.Errorf("standings for the upcoming match's season = %+v, want none", upcomingSeason)
	}
	upcomingScorers, err := env.standings.GetScorers(env.groupID, "2030-2031")
	if err != nil {
		t.Fatalf("GetScorers(2030-2031) returned error: %v", err)
	}
	if len(upcomingScorers) != 0 {
		t.Errorf("scorers for the upcoming match's season = %+v, want none", upcomingScorers)
	}

	// ...but the season itself *is* offered, on purpose: the matches list is
	// filtered by the selected season, so hiding 2030-2031 here would make the
	// upcoming match unreachable in the UI.
	seasons, err := env.standings.GetSeasons(env.groupID)
	if err != nil {
		t.Fatalf("GetSeasons returned error: %v", err)
	}
	var sawUpcoming, sawPlayed bool
	for _, season := range seasons {
		switch season {
		case "2030-2031":
			sawUpcoming = true
		case "2024-2025":
			sawPlayed = true
		}
	}
	if !sawPlayed {
		t.Errorf("GetSeasons = %v, want it to include the played season 2024-2025", seasons)
	}
	if !sawUpcoming {
		t.Errorf("GetSeasons = %v, want it to include the scheduled match's season 2030-2031 "+
			"(this is a decision, not an oversight — see ComputeSeasons)", seasons)
	}
}
