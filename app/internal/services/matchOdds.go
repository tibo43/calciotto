package services

import (
	"math"

	"app/internal/models"

	"github.com/google/uuid"
)

// Betting-odds constants. Named rather than inlined for the same reason
// elo.go's own constants are: a future tuning pass touches exactly one
// place, and this file's own tests reference the same values.
const (
	// PredictionDrawProbabilityAtEven is the modelled probability of a draw
	// when both teams carry the identical average rating — i.e. the ceiling
	// this model ever assigns a draw, since the chance of a draw is modelled
	// as strictly decreasing as the rating gap grows (see
	// PredictionDrawSlopePerEloPoint). 0.30 is a deliberately round starting
	// point, not fitted to any real calciotto data yet.
	PredictionDrawProbabilityAtEven = 0.30

	// PredictionDrawProbabilityFloor is how far the modelled draw probability
	// is allowed to fall as the rating gap widens — a real, if unlikely,
	// draw between very mismatched teams should never evaporate to
	// (and divide-by-near-zero toward) a degenerate odds value.
	PredictionDrawProbabilityFloor = 0.12

	// PredictionDrawSlopePerEloPoint is how much the draw probability drops
	// per Elo point of rating gap between the two teams — e.g. a 200-point
	// gap (a clear favourite, see CLAUDE.md's Elo section for what a typical
	// gap looks like at this K-factor) shaves 0.09 off
	// PredictionDrawProbabilityAtEven.
	PredictionDrawSlopePerEloPoint = 0.00045

	// PredictionOddsFloor and PredictionOddsCap bound every multiplier this
	// model produces, regardless of how lopsided the underlying probability
	// is. The floor keeps backing an overwhelming favourite from being
	// worth effectively nothing; the cap keeps backing a huge underlying
	// mismatch (most likely an early-season, still-provisional rating gap —
	// see EloKProvisional) from paying an implausible multiple for what is,
	// after all, a game between friends.
	PredictionOddsFloor = 1.1
	PredictionOddsCap   = 10.0
)

// ComputeMatchOdds derives one match's three-way betting odds from each
// team's own pre-match average Elo rating (see EloRatingsBeforeMatch for how
// those are obtained). It is a pure function of the two ratings, unit-tested
// directly without a database — the same split every other Compute*
// function in this codebase follows.
//
// The draw probability is modelled separately from, and before, the win/loss
// split: real football draws are most likely between evenly matched sides
// and strictly less likely the bigger the gap, which plain Elo "expected
// score" (a single number blending win and draw together) cannot express on
// its own. Having fixed the draw probability, the remaining probability mass
// (1 - pDraw) is split between the two teams in exactly the same ratio Elo's
// own logistic expected-score formula already gives — the identical formula
// ComputeEloStandings itself uses, just reused to divide a smaller pie
// instead of the whole one.
//
// A multiplier is simply 1/probability — "fair" odds with no house margin,
// since nothing here is real money and nobody is taking a cut — clamped to
// [PredictionOddsFloor, PredictionOddsCap].
func ComputeMatchOdds(ratingA, ratingB float64) models.MatchOdds {
	gap := math.Abs(ratingA - ratingB)
	pDraw := PredictionDrawProbabilityAtEven - PredictionDrawSlopePerEloPoint*gap
	if pDraw < PredictionDrawProbabilityFloor {
		pDraw = PredictionDrawProbabilityFloor
	}

	expectedA := 1.0 / (1.0 + math.Pow(10, (ratingB-ratingA)/EloDivisor))
	expectedB := 1.0 - expectedA

	pWinA := expectedA * (1 - pDraw)
	pWinB := expectedB * (1 - pDraw)

	return models.MatchOdds{
		TeamAOdds: clampOdds(1 / pWinA),
		DrawOdds:  clampOdds(1 / pDraw),
		TeamBOdds: clampOdds(1 / pWinB),
	}
}

func clampOdds(odds float64) float64 {
	switch {
	case odds < PredictionOddsFloor:
		return PredictionOddsFloor
	case odds > PredictionOddsCap:
		return PredictionOddsCap
	default:
		return odds
	}
}

// EloRatingsBeforeMatch returns every player's Elo rating as of right before
// targetMatchID, by replaying only the matches that come strictly earlier in
// the group's chronological history through the exact same ComputeEloStandings
// the CLI-only cmd/elo already uses — just handed a truncated slice, rather
// than a second, parallel implementation of the replay.
//
// This is what keeps a bet's odds anchored to the skill level *before* this
// match's own result could tell them anything, and what makes settling a bet
// later never retroactively change what the odds *were* at prediction time:
// nothing after targetMatchID in the sorted history is ever consulted.
//
// matches need not be pre-sorted or pre-filtered to one match's group —
// ComputeEloStandings already sorts defensively, and ordering (not identity)
// is all this function relies on; callers in practice always pass one
// group's own matches. A player not present in the result (new to the group,
// including on their very first-ever match) is simply absent from the map —
// callers fall back to EloStartingRating themselves, the same "enter at
// EloStartingRating the instant first seen" rule ComputeEloStandings applies
// internally.
func EloRatingsBeforeMatch(matches []models.MatchWithDetails, targetMatchID uuid.UUID) map[uuid.UUID]float64 {
	sorted := sortMatchesChronologically(matches)

	var before []models.MatchWithDetails
	for _, match := range sorted {
		if match.ID == targetMatchID {
			break
		}
		before = append(before, match)
	}

	ratings := make(map[uuid.UUID]float64)
	for _, row := range ComputeEloStandings(before) {
		ratings[row.PlayerID] = row.Rating
	}
	return ratings
}

// teamAverageRating is the average pre-match rating of a team's composed
// roster, falling back to EloStartingRating for any player absent from
// ratings (new to the group) and for a team with no players at all (there is
// nothing to average, so the neutral starting rating stands in rather than
// dividing by zero) — callers are expected to gate on both teams actually
// being composed before trusting the result as a real odds input (see
// MatchPredictionService's own composition check).
func teamAverageRating(team models.TeamWithPlayers, ratings map[uuid.UUID]float64) float64 {
	if len(team.Players) == 0 {
		return EloStartingRating
	}
	sum := 0.0
	for _, player := range team.Players {
		if rating, ok := ratings[player.ID]; ok {
			sum += rating
		} else {
			sum += EloStartingRating
		}
	}
	return sum / float64(len(team.Players))
}
