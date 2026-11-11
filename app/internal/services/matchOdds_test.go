package services

import (
	"math"
	"testing"
)

// almostEqual guards against float rounding noise in the assertions below —
// these constants were hand-derived during design and are checked to two
// decimal places, not bit-for-bit.
func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.005
}

func TestComputeMatchOdds_EvenMatch(t *testing.T) {
	odds := ComputeMatchOdds(1500, 1500)

	if !almostEqual(odds.DrawOdds, 2.50) {
		t.Errorf("DrawOdds = %.3f, want ~2.50 (pDraw at even rating = %.2f)", odds.DrawOdds, PredictionDrawProbabilityAtEven)
	}
	if !almostEqual(odds.TeamAOdds, 3.33) || !almostEqual(odds.TeamBOdds, 3.33) {
		t.Errorf("TeamAOdds/TeamBOdds = %.3f/%.3f, want ~3.33 each (symmetric at equal rating)", odds.TeamAOdds, odds.TeamBOdds)
	}
}

func TestComputeMatchOdds_FavouriteGetsLowerOddsThanUnderdog(t *testing.T) {
	odds := ComputeMatchOdds(1700, 1500)

	if odds.TeamAOdds >= odds.TeamBOdds {
		t.Errorf("odds = %+v, want the 200-point favourite (TeamA) priced lower than the underdog (TeamB)", odds)
	}
	if odds.TeamAOdds <= PredictionOddsFloor {
		t.Errorf("TeamAOdds = %.3f, want strictly above the floor for a merely-200-point gap", odds.TeamAOdds)
	}
}

// TestComputeMatchOdds_DrawProbabilityShrinksWithGap pins the exact
// complaint a real conversation raised about an earlier version of this
// model: a flat, gap-independent draw probability put every match's draw
// odds at the same fixed 4.00x regardless of how lopsided the sides were —
// unrealistic, since a draw between very mismatched teams should be rarer,
// not equally likely, so it must pay out *more*, not the same, as the gap
// widens.
func TestComputeMatchOdds_DrawProbabilityShrinksWithGap(t *testing.T) {
	even := ComputeMatchOdds(1500, 1500)
	gap200 := ComputeMatchOdds(1700, 1500)
	gap300 := ComputeMatchOdds(1800, 1500)

	if !(even.DrawOdds < gap200.DrawOdds && gap200.DrawOdds < gap300.DrawOdds) {
		t.Errorf("draw odds = %.3f (gap 0) / %.3f (gap 200) / %.3f (gap 300), want strictly increasing with the rating gap", even.DrawOdds, gap200.DrawOdds, gap300.DrawOdds)
	}
}

func TestComputeMatchOdds_WorkedExamplesFromDesign(t *testing.T) {
	// The exact figures walked through with the product owner before
	// implementation — pinned here so a future tuning change has to
	// deliberately touch this test, not silently drift past it.
	cases := []struct {
		name                       string
		ratingA, ratingB           float64
		wantAOdds, wantDraw, wantB float64
	}{
		{"even (0 gap)", 1500, 1500, 3.33, 2.50, 3.33},
		{"slight favourite (100 gap)", 1600, 1500, 2.42, 2.82, 4.31},
		{"clear favourite (200 gap)", 1700, 1500, 1.91, 3.23, 6.03},
		{"big gap (300 gap)", 1800, 1500, 1.60, 3.77, 9.01},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			odds := ComputeMatchOdds(tc.ratingA, tc.ratingB)
			if !almostEqual(odds.TeamAOdds, tc.wantAOdds) {
				t.Errorf("TeamAOdds = %.3f, want ~%.2f", odds.TeamAOdds, tc.wantAOdds)
			}
			if !almostEqual(odds.DrawOdds, tc.wantDraw) {
				t.Errorf("DrawOdds = %.3f, want ~%.2f", odds.DrawOdds, tc.wantDraw)
			}
			if !almostEqual(odds.TeamBOdds, tc.wantB) {
				t.Errorf("TeamBOdds = %.3f, want ~%.2f", odds.TeamBOdds, tc.wantB)
			}
		})
	}
}

func TestComputeMatchOdds_ExtremeGapIsCapped(t *testing.T) {
	// A 500-point gap (plausible early-season, still-provisional ratings —
	// see EloKProvisional) would otherwise price the underdog at ~25x
	// without a cap, which is wildly disproportionate for a game between
	// friends with no real stake — the whole reason PredictionOddsCap
	// exists.
	odds := ComputeMatchOdds(1900, 1400)

	if odds.TeamBOdds != PredictionOddsCap {
		t.Errorf("TeamBOdds = %.3f, want exactly the cap (%.1f) on this large a gap", odds.TeamBOdds, PredictionOddsCap)
	}
	if odds.TeamAOdds < PredictionOddsFloor || odds.TeamAOdds > PredictionOddsCap {
		t.Errorf("TeamAOdds = %.3f, want within [%.1f, %.1f]", odds.TeamAOdds, PredictionOddsFloor, PredictionOddsCap)
	}
}

func TestComputeMatchOdds_NeverBelowFloorOrAboveCap(t *testing.T) {
	// An absurd gap (nobody in a real calciotto group reaches this, but the
	// function must still never return a degenerate value) exercises both
	// bounds on both sides at once.
	odds := ComputeMatchOdds(3000, 500)

	for name, value := range map[string]float64{"TeamAOdds": odds.TeamAOdds, "DrawOdds": odds.DrawOdds, "TeamBOdds": odds.TeamBOdds} {
		if value < PredictionOddsFloor || value > PredictionOddsCap {
			t.Errorf("%s = %.3f, want within [%.1f, %.1f]", name, value, PredictionOddsFloor, PredictionOddsCap)
		}
	}
}
