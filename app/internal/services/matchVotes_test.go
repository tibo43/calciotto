package services

import (
	"errors"
	"testing"
	"time"

	"app/internal/models"

	"github.com/google/uuid"
)

// The tests in this file are pure: ComputeMotmWinners is a function of an
// already-computed tally, so the whole "who won this match's award" rule is
// covered without a database — the same split standings_test.go exercises on
// ComputePointsStandings.

func tally(votes ...models.MatchVoteTally) []models.MatchVoteTally { return votes }

func candidate(id uuid.UUID, name string, votes int) models.MatchVoteTally {
	return models.MatchVoteTally{PlayerID: id, Name: name, Votes: votes}
}

func TestComputeMotmWinners_SingleClearWinner(t *testing.T) {
	alice, bob := uuid.New(), uuid.New()

	winners := ComputeMotmWinners(tally(
		candidate(alice, "alice", 3),
		candidate(bob, "bob", 1),
	))

	if len(winners) != 1 || winners[0] != alice {
		t.Errorf("winners = %v, want [alice] only", winners)
	}
}

// TestComputeMotmWinners_TieIsInclusive pins the product decision: no
// arbitrary tie-break, every player tied for the most votes gets the award.
func TestComputeMotmWinners_TieIsInclusive(t *testing.T) {
	alice, bob, carol := uuid.New(), uuid.New(), uuid.New()

	winners := ComputeMotmWinners(tally(
		candidate(alice, "alice", 2),
		candidate(bob, "bob", 2),
		candidate(carol, "carol", 1),
	))

	if len(winners) != 2 {
		t.Fatalf("winners = %v, want exactly 2 (alice and bob tied at the top)", winners)
	}
	got := map[uuid.UUID]bool{winners[0]: true, winners[1]: true}
	if !got[alice] || !got[bob] {
		t.Errorf("winners = %v, want {alice, bob}", winners)
	}
	if got[carol] {
		t.Errorf("carol (fewer votes) was included in the winners: %v", winners)
	}
}

func TestComputeMotmWinners_EmptyTallyHasNoWinner(t *testing.T) {
	if winners := ComputeMotmWinners(nil); len(winners) != 0 {
		t.Errorf("winners = %v, want none for an empty tally", winners)
	}
	if winners := ComputeMotmWinners(tally()); len(winners) != 0 {
		t.Errorf("winners = %v, want none for an empty tally", winners)
	}
}

// TestComputeMotmWinners_AllTiedMeansEveryoneWins: a match where every
// candidate got exactly one vote has no single "best" player, so the award
// goes to all of them rather than being withheld or arbitrarily picked.
func TestComputeMotmWinners_AllTiedMeansEveryoneWins(t *testing.T) {
	alice, bob, carol := uuid.New(), uuid.New(), uuid.New()

	winners := ComputeMotmWinners(tally(
		candidate(alice, "alice", 1),
		candidate(bob, "bob", 1),
		candidate(carol, "carol", 1),
	))

	if len(winners) != 3 {
		t.Errorf("winners = %v, want all 3 candidates tied at 1 vote each", winners)
	}
}

// TestVotingWindowError mirrors TestRegistrationWindowError's own table shape:
// VotingWindowError is a pure function of an already-loaded match, so both
// anchor cases (scheduled vs. recorded-after-the-fact) are covered without a
// database.
func TestVotingWindowError(t *testing.T) {
	kickoff := time.Date(2026, 9, 6, 18, 0, 0, 0, time.UTC)
	loggedAt := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		match models.Match
		now   time.Time
		want  error
	}{
		{
			name:  "scheduled match, well within the window after kick-off",
			match: models.Match{ScheduledAt: &kickoff, CreatedAt: loggedAt},
			now:   kickoff.Add(time.Hour),
			want:  nil,
		},
		{
			name:  "scheduled match, exactly at the 24h deadline after kick-off",
			match: models.Match{ScheduledAt: &kickoff, CreatedAt: loggedAt},
			now:   kickoff.Add(motmVotingWindow),
			want:  ErrVotingClosed,
		},
		{
			name:  "scheduled match, one second before the deadline",
			match: models.Match{ScheduledAt: &kickoff, CreatedAt: loggedAt},
			now:   kickoff.Add(motmVotingWindow - time.Second),
			want:  nil,
		},
		{
			// The anchor is kick-off, not CreatedAt: a match logged long ago
			// but scheduled for a recent kick-off is still votable.
			name:  "scheduled match anchors on kick-off, not on CreatedAt",
			match: models.Match{ScheduledAt: &kickoff, CreatedAt: kickoff.Add(-30 * 24 * time.Hour)},
			now:   kickoff.Add(time.Hour),
			want:  nil,
		},
		{
			name:  "unscheduled match anchors on CreatedAt, well within the window",
			match: models.Match{CreatedAt: loggedAt},
			now:   loggedAt.Add(time.Hour),
			want:  nil,
		},
		{
			name:  "unscheduled match, exactly at the 24h deadline after being logged",
			match: models.Match{CreatedAt: loggedAt},
			now:   loggedAt.Add(motmVotingWindow),
			want:  ErrVotingClosed,
		},
		{
			name:  "unscheduled match, one second before the deadline",
			match: models.Match{CreatedAt: loggedAt},
			now:   loggedAt.Add(motmVotingWindow - time.Second),
			want:  nil,
		},
		{
			name:  "unscheduled match, long past the deadline",
			match: models.Match{CreatedAt: loggedAt},
			now:   loggedAt.Add(30 * 24 * time.Hour),
			want:  ErrVotingClosed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := VotingWindowError(tt.match, tt.now); !errors.Is(got, tt.want) {
				t.Errorf("VotingWindowError() = %v, want %v", got, tt.want)
			}
		})
	}
}
