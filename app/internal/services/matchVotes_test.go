package services

import (
	"testing"

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
