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

// TestVotingWindowError pins both rules. A scheduled match closes 24 hours
// after its kick-off, the kick-off + 24h instant itself closed. A match
// without a kick-off accepts votes through the end of Date+1, closing at the
// stroke of Date+2.
func TestVotingWindowError(t *testing.T) {
	playedOn := models.Date(time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC))
	kickoff := time.Date(2026, 9, 3, 18, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		match models.Match
		now   time.Time
		want  error
	}{
		{
			name:  "still open on match day itself",
			match: models.Match{Date: playedOn},
			now:   time.Date(2026, 9, 3, 23, 0, 0, 0, time.UTC),
			want:  nil,
		},
		{
			name:  "still open the day after, right up to the last second",
			match: models.Match{Date: playedOn},
			now:   time.Date(2026, 9, 4, 23, 59, 59, 0, time.UTC),
			want:  nil,
		},
		{
			name:  "closed at the stroke of the second day after",
			match: models.Match{Date: playedOn},
			now:   time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC),
			want:  ErrVotingClosed,
		},
		{
			name:  "closed long after",
			match: models.Match{Date: playedOn},
			now:   time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
			want:  ErrVotingClosed,
		},
		{
			name:  "scheduled: open one second before kick-off + 24h",
			match: models.Match{Date: playedOn, ScheduledAt: &kickoff},
			now:   kickoff.Add(24*time.Hour - time.Second),
			want:  nil,
		},
		{
			name:  "scheduled: closed at kick-off + 24h",
			match: models.Match{Date: playedOn, ScheduledAt: &kickoff},
			now:   kickoff.Add(24 * time.Hour),
			want:  ErrVotingClosed,
		},
		{
			// Before midnight of Date+2, where a match without a kick-off
			// would still be open: the kick-off is what counts.
			name:  "scheduled: closed the next evening even though Date+1 isn't over",
			match: models.Match{Date: playedOn, ScheduledAt: &kickoff},
			now:   time.Date(2026, 9, 4, 20, 0, 0, 0, time.UTC),
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
