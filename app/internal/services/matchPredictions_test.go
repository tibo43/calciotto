package services

import (
	"testing"
	"time"

	"app/internal/models"

	"github.com/google/uuid"
)

func TestPredictionWindowError(t *testing.T) {
	opensAt := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	kickOff := time.Date(2026, 9, 6, 18, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		match          models.Match
		rosterComposed bool
		now            time.Time
		want           error
	}{
		{
			name:           "unscheduled match",
			match:          models.Match{},
			rosterComposed: true,
			now:            opensAt,
			want:           ErrPredictionsNotAvailable,
		},
		{
			name:           "scheduled but roster not composed yet",
			match:          scheduledMatch(opensAt, kickOff, nil),
			rosterComposed: false,
			now:            opensAt,
			want:           ErrPredictionsNotAvailable,
		},
		{
			name:           "open once composed, before kick-off",
			match:          scheduledMatch(opensAt, kickOff, nil),
			rosterComposed: true,
			now:            kickOff.Add(-time.Second),
			want:           nil,
		},
		{
			name:           "closed at the instant of kick-off",
			match:          scheduledMatch(opensAt, kickOff, nil),
			rosterComposed: true,
			now:            kickOff,
			want:           ErrPredictionsClosed,
		},
		{
			name:           "closed well after kick-off",
			match:          scheduledMatch(opensAt, kickOff, nil),
			rosterComposed: true,
			now:            kickOff.Add(48 * time.Hour),
			want:           ErrPredictionsClosed,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := PredictionWindowError(tc.match, tc.rosterComposed, tc.now); got != tc.want {
				t.Errorf("PredictionWindowError() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRosterFullyComposed(t *testing.T) {
	alice, bob := newPlayer(uuid.New(), "alice", 0), newPlayer(uuid.New(), "bob", 0)

	tests := []struct {
		name  string
		teams []models.TeamWithPlayers
		want  bool
	}{
		{"both composed", []models.TeamWithPlayers{newTeam(uuid.New(), "black", 0, alice), newTeam(uuid.New(), "white", 0, bob)}, true},
		{"only one team composed", []models.TeamWithPlayers{newTeam(uuid.New(), "black", 0, alice), newTeam(uuid.New(), "white", 0)}, false},
		{"neither composed", []models.TeamWithPlayers{newTeam(uuid.New(), "black", 0), newTeam(uuid.New(), "white", 0)}, false},
		{"not exactly two teams", []models.TeamWithPlayers{newTeam(uuid.New(), "black", 0, alice)}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := rosterFullyComposed(tc.teams); got != tc.want {
				t.Errorf("rosterFullyComposed() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestEloRatingsBeforeMatch_ExcludesTheTargetMatchAndEverythingAfterIt(t *testing.T) {
	alice := []models.PlayerCustom{{ID: uuid.New(), Name: "alice"}}
	bob := []models.PlayerCustom{{ID: uuid.New(), Name: "bob"}}
	aliceID, bobID := alice[0].ID, bob[0].ID

	day1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 1, 8, 0, 0, 0, 0, time.UTC)
	day3 := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)

	// Alice wins the early match (she should end up rated above the
	// starting rating because of it), draws the target match, then wins the
	// later one — only the first result may leak into "before the target".
	early := eloMatch(day1, day1, alice, 1, bob, 0)
	target := eloMatch(day2, day2, alice, 1, bob, 1)
	later := eloMatch(day3, day3, alice, 1, bob, 0)

	ratingsBeforeTarget := EloRatingsBeforeMatch([]models.MatchWithDetails{early, target, later}, target.ID)
	ratingsFromEarlyOnly := EloRatingsBeforeMatch([]models.MatchWithDetails{early}, target.ID)

	if ratingsBeforeTarget[aliceID] != ratingsFromEarlyOnly[aliceID] || ratingsBeforeTarget[bobID] != ratingsFromEarlyOnly[bobID] {
		t.Errorf("ratings before the target match = %+v, want exactly the same as replaying only the earlier match (%+v) — target and later must not leak in", ratingsBeforeTarget, ratingsFromEarlyOnly)
	}
	if ratingsBeforeTarget[aliceID] <= EloStartingRating {
		t.Errorf("alice's rating before the target match = %.1f, want above the starting rating (%.1f) after winning the earlier one", ratingsBeforeTarget[aliceID], EloStartingRating)
	}
	// Passing the unsorted order (later, target, early) must make no
	// difference — EloRatingsBeforeMatch sorts defensively, the same
	// "never trust the caller's order" posture ComputeEloStandings itself
	// takes (see its own doc comment).
	ratingsFromUnsortedInput := EloRatingsBeforeMatch([]models.MatchWithDetails{later, target, early}, target.ID)
	if ratingsFromUnsortedInput[aliceID] != ratingsBeforeTarget[aliceID] {
		t.Errorf("ratings from out-of-order input = %.1f, want the same as sorted input (%.1f)", ratingsFromUnsortedInput[aliceID], ratingsBeforeTarget[aliceID])
	}
}

func TestEloRatingsBeforeMatch_UnknownPlayerAbsentFromResult(t *testing.T) {
	alice := []models.PlayerCustom{{ID: uuid.New(), Name: "alice"}}
	bob := []models.PlayerCustom{{ID: uuid.New(), Name: "bob"}}
	match := eloMatch(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Time{}, alice, 1, bob, 0)

	ratings := EloRatingsBeforeMatch([]models.MatchWithDetails{match}, match.ID)
	if len(ratings) != 0 {
		t.Errorf("ratings before the only match either player has ever played = %+v, want empty (neither has a prior match to be rated from)", ratings)
	}
}

func TestTeamAverageRating(t *testing.T) {
	alice, bob := uuid.New(), uuid.New()
	ratings := map[uuid.UUID]float64{alice: 1600, bob: 1400}

	tests := []struct {
		name string
		team models.TeamWithPlayers
		want float64
	}{
		{"averages known players", newTeam(uuid.New(), "black", 0, newPlayer(alice, "alice", 0), newPlayer(bob, "bob", 0)), 1500},
		{"unknown player falls back to the starting rating", newTeam(uuid.New(), "black", 0, newPlayer(uuid.New(), "carol", 0)), EloStartingRating},
		{"empty team falls back to the starting rating", newTeam(uuid.New(), "black", 0), EloStartingRating},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := teamAverageRating(tc.team, ratings); got != tc.want {
				t.Errorf("teamAverageRating() = %.1f, want %.1f", got, tc.want)
			}
		})
	}
}
