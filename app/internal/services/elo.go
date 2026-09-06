package services

import (
	"math"
	"sort"
	"time"

	"app/internal/models"

	"github.com/google/uuid"
)

// Elo rating constants. Named rather than inlined so cmd/elo and this file's
// own tests reference the same values a future tuning change would touch in
// exactly one place. See CLAUDE.md's "Elo rating" section for why these
// particular numbers were picked (in short: a K-factor well above a typical
// chess federation's, because a calciotto player accumulates rated games far
// slower than a competitive chess player does, so each one has to carry more
// signal to say anything within a season or two).
const (
	// EloStartingRating is the rating a player enters a group's history at,
	// the first time they're encountered while replaying its matches in
	// chronological order — there is no separate up-front initialization
	// step, a player simply starts here the moment ComputeEloStandings first
	// sees their id.
	EloStartingRating = 1500.0

	// EloDivisor is the standard chess Elo logistic divisor: the rating gap
	// (in points) that corresponds to input scaling by one order of
	// magnitude in the expected-score formula.
	EloDivisor = 400.0

	// EloKProvisional is the K-factor applied to a player's own update while
	// they have fewer than EloProvisionalGameThreshold rated games in this
	// group's history — a new player's rating is allowed to move faster
	// while there isn't much history to weigh it against yet.
	EloKProvisional = 60.0

	// EloKSteady is the K-factor applied once a player has reached
	// EloProvisionalGameThreshold rated games — their rating is by then
	// assumed to be in roughly the right neighbourhood, so each further
	// match moves it less.
	EloKSteady = 40.0

	// EloProvisionalGameThreshold is the number of a player's own prior
	// rated games (in this group) below which EloKProvisional applies,
	// EloKSteady above it.
	EloProvisionalGameThreshold = 10
)

// ratingState is the mutable per-player accumulator ComputeEloStandings
// replays matches against — the same "accumulate into a map keyed by player
// id, then flatten to rows at the end" shape ComputePointsStandings/
// ComputeScorers already use, just carrying a rating and a games-played
// counter instead of win/draw/loss/goals.
type ratingState struct {
	name   string
	rating float64
	games  int
}

// eloKFactor returns the K-factor for a player who has gamesPlayedBefore
// prior rated games in this group — the player's *own* count, not the
// match's index in the group's history, since two players meeting in the
// same match can each be at a different point in their own rated history
// (a veteran facing a debutant, say).
func eloKFactor(gamesPlayedBefore int) float64 {
	if gamesPlayedBefore < EloProvisionalGameThreshold {
		return EloKProvisional
	}
	return EloKSteady
}

// sortMatchesChronologically returns a new slice ordered ascending by Date,
// then CreatedAt, then ID (UUIDs compared as strings) — the exact reverse of
// MatchService.GetMatchesDetails' own `ORDER BY match_date DESC`, and the
// order an Elo replay requires: a rating update must see every earlier match
// before a later one, or the whole history replays in the wrong sequence.
// The three-level tie-break is fully deterministic, mirroring
// MatchRegistrationService's own "CreatedAt ASC, id ASC" ordering
// convention, so two matches sharing a Date (the common case — Date is a
// calendar day, not an instant) can never be replayed in an arbitrary order
// between two calls.
//
// It never mutates its input (it copies before sorting) and does not trust
// the caller to have sorted already: ComputeEloStandings calls this
// unconditionally on its own input, so a caller who forgets to sort — or who
// hands it whatever order GetMatchesDetails happened to return — still gets
// a correct chronological replay. StandingsService.GetEloStandings also
// sorts explicitly before calling ComputeEloStandings, which makes this a
// deliberate belt-and-suspenders duplication rather than a required step:
// sorting an already-sorted slice again is cheap (a group's whole history is
// at most a few hundred matches) and keeps the "did I sort?" question from
// ever mattering at either layer.
func sortMatchesChronologically(matches []models.MatchWithDetails) []models.MatchWithDetails {
	sorted := make([]models.MatchWithDetails, len(matches))
	copy(sorted, matches)
	sort.SliceStable(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		if a.Date.Before(b.Date) {
			return true
		}
		if b.Date.Before(a.Date) {
			return false
		}
		if a.CreatedAt.Before(b.CreatedAt) {
			return true
		}
		if b.CreatedAt.Before(a.CreatedAt) {
			return false
		}
		return a.ID.String() < b.ID.String()
	})
	return sorted
}

// ComputeEloStandings replays an already-loaded set of matches in
// chronological order and returns each player's resulting Elo rating. Like
// ComputePointsStandings/ComputeScorers/ComputeMotmStandings, it is a pure
// function of its input — which matches to consider (scoped to a group,
// filtered to completed ones) is entirely the caller's concern, so scoping
// never has to touch this logic. Unlike those three, order is not
// incidental here: it sorts its input itself via sortMatchesChronologically
// regardless of what order it arrives in, rather than documenting an
// "expects pre-sorted input" precondition a caller could silently violate —
// an Elo replay is only meaningful in chronological order, so this function
// makes that true unconditionally instead of trusting it.
//
// It re-derives the "is this match played" check ComputePointsStandings
// already applies (exactly two teams, both non-empty) rather than assuming
// the caller pre-filtered — the same defensive posture ComputePointsStandings
// itself takes, and for the same reason: a scheduled match with a still-empty
// roster must never be replayed as a 0-0 draw that moves anybody's rating.
//
// A player enters the ratings map at EloStartingRating the instant they are
// first seen (no separate initialization pass), and every player on a team
// is updated identically for that match: the team's expected score is the
// logistic function of the *average* rating gap between the two rosters (not
// weighted by individual rating — see CLAUDE.md), and the team's actual
// score is 1/0/0.5 for a win/loss/draw exactly as ComputePointsStandings
// derives it from Team.Score. Both teams' expected/actual scores and every
// player's resulting delta are computed from pre-match ratings before any of
// them are applied — team B's update never sees team A's already-applied
// delta for the same match, or vice versa. Each player's K-factor is their
// own — EloKProvisional below EloProvisionalGameThreshold rated games in
// this group, EloKSteady at or above it — using their games-played count
// *before* this match; the counter is only incremented once every delta for
// the match has already been computed, so it affects a player's next match,
// never this one.
func ComputeEloStandings(matches []models.MatchWithDetails) []models.EloStandingRow {
	sorted := sortMatchesChronologically(matches)

	players := make(map[uuid.UUID]*ratingState)
	ensure := func(id uuid.UUID, name string) *ratingState {
		st, ok := players[id]
		if !ok {
			st = &ratingState{name: name, rating: EloStartingRating}
			players[id] = st
		}
		return st
	}

	for _, match := range sorted {
		// Mirrors ComputePointsStandings' own "both sides have a full
		// roster" check exactly (see CLAUDE.md's "Standings" section) — a
		// scheduled match with an empty or partial roster is not "played"
		// and must not move anybody's rating.
		if len(match.Teams) != 2 {
			continue
		}
		teamA, teamB := match.Teams[0], match.Teams[1]
		if len(teamA.Players) == 0 || len(teamB.Players) == 0 {
			continue
		}

		for _, p := range teamA.Players {
			ensure(p.ID, p.Name)
		}
		for _, p := range teamB.Players {
			ensure(p.ID, p.Name)
		}

		teamAverage := func(team models.TeamWithPlayers) float64 {
			sum := 0.0
			for _, p := range team.Players {
				sum += players[p.ID].rating
			}
			return sum / float64(len(team.Players))
		}
		ratingA, ratingB := teamAverage(teamA), teamAverage(teamB)

		expectedA := 1.0 / (1.0 + math.Pow(10, (ratingB-ratingA)/EloDivisor))
		expectedB := 1.0 - expectedA

		var actualA, actualB float64
		switch {
		case teamA.Score > teamB.Score:
			actualA, actualB = 1.0, 0.0
		case teamB.Score > teamA.Score:
			actualA, actualB = 0.0, 1.0
		default:
			actualA, actualB = 0.5, 0.5
		}

		// Compute every participant's delta from pre-match state before
		// applying any of them, so a team's update can never leak into the
		// other team's own delta computation for the same match.
		type playerDelta struct {
			id    uuid.UUID
			value float64
		}
		deltas := make([]playerDelta, 0, len(teamA.Players)+len(teamB.Players))
		for _, p := range teamA.Players {
			k := eloKFactor(players[p.ID].games)
			deltas = append(deltas, playerDelta{p.ID, k * (actualA - expectedA)})
		}
		for _, p := range teamB.Players {
			k := eloKFactor(players[p.ID].games)
			deltas = append(deltas, playerDelta{p.ID, k * (actualB - expectedB)})
		}

		for _, d := range deltas {
			players[d.id].rating += d.value
		}
		// Only now, after every delta for this match has been both computed
		// and applied, does each participant's games-played counter advance
		// — so it affects their K-factor starting with their *next* match.
		for _, p := range teamA.Players {
			players[p.ID].games++
		}
		for _, p := range teamB.Players {
			players[p.ID].games++
		}
	}

	rows := make([]models.EloStandingRow, 0, len(players))
	for id, st := range players {
		rows = append(rows, models.EloStandingRow{
			PlayerID:    id,
			Name:        st.name,
			Rating:      st.rating,
			GamesPlayed: st.games,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Rating != rows[j].Rating {
			return rows[i].Rating > rows[j].Rating
		}
		return rows[i].Name < rows[j].Name
	})
	return rows
}

// GetEloStandings ranks a single group's players by their continuous Elo
// rating, computed fresh from that group's own match history — never
// persisted, never cached, and never scoped by season (see CLAUDE.md's "Elo
// rating" section for why: a rating pool is per group the way real matches
// are, and skill doesn't reset every September 1st the way an accumulator
// like points/goals implicitly does by being filtered per season).
//
// It follows GetPointsStandings/GetScorers/GetMotmStandings' exact shape
// minus the season parameter: load the group's matches (an empty season
// means no filtering, same as every other caller in this file),
// FilterCompletedMatches for the same reason GetPointsStandings does (a
// scheduled match's roster can be composed, and therefore "playable" by
// ComputeEloStandings' own definition, before its own kick-off — see
// IsMatchCompleted's comment), sort chronologically, run the pure
// aggregator, then tag IsMember as a post-processing step via the same
// currentMemberIDs helper the other Get*Standings methods already share.
// FilterMatchesBySeason is deliberately never called here — there is no
// season parameter to apply it with.
func (s *StandingsService) GetEloStandings(groupID uuid.UUID) ([]models.EloStandingRow, error) {
	matches, err := s.MatchService.GetMatchesDetails(groupID, "")
	if err != nil {
		return nil, err
	}
	matches = FilterCompletedMatches(matches, time.Now())
	matches = sortMatchesChronologically(matches)
	rows := ComputeEloStandings(matches)

	currentMembers, err := s.currentMemberIDs(groupID)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		rows[i].IsMember = currentMembers[rows[i].PlayerID]
	}
	return rows, nil
}
