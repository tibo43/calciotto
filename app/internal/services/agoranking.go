package services

import (
	"math"
	"sort"
	"strings"
	"time"

	"app/internal/models"

	"github.com/google/uuid"
)

// agorankingTieEpsilon treats two ratings this close as the same value: two
// different (points, matches) pairs can give the same rating mathematically
// yet differ in the last bits of a float64.
const agorankingTieEpsilon = 1e-9

// GetAgoranking follows the same shape as GetPointsStandings: load the
// group's matches, season-filter and keep the counted ones, run the pure
// ComputeAgoranking, then tag IsMember.
func (s *StandingsService) GetAgoranking(groupID uuid.UUID, season string) (*models.Agoranking, error) {
	matches, err := s.MatchService.GetMatchesDetails(groupID, "")
	if err != nil {
		return nil, err
	}
	matches = FilterCountedMatches(FilterMatchesBySeason(matches, season), time.Now())
	result := ComputeAgoranking(matches)

	currentMembers, err := s.currentMemberIDs(groupID)
	if err != nil {
		return nil, err
	}
	for i := range result.Ranked {
		result.Ranked[i].IsMember = currentMembers[result.Ranked[i].PlayerID]
	}
	for i := range result.Unranked {
		result.Unranked[i].IsMember = currentMembers[result.Unranked[i].PlayerID]
	}
	return result, nil
}

// ComputeAgoranking ranks players by points per match, weighted by how many
// matches they played. A win is worth 3, a draw 1, a loss 0 (the same rules
// and the same "both sides have players" filter as ComputePointsStandings,
// whose rows it reuses).
//
//   - Matches: matches played by the group (the ones ComputePointsStandings counts).
//   - N = ceil(Matches / 3): the number of "average" matches every player
//     starts with, and the minimum number of matches to be ranked.
//   - GroupAverage = total points of every participation ÷ total participations.
//   - Rating = (points + N × GroupAverage) ÷ (matches + N).
//
// Ranked (matches ≥ N) is sorted by unrounded rating, then by matches played,
// then by name. Unranked (matches < N) is sorted by matches played, then by
// points per match, then by name. Names compare without case.
func ComputeAgoranking(matches []models.MatchWithDetails) *models.Agoranking {
	played := 0
	for _, match := range matches {
		if len(match.Teams) == 2 && len(match.Teams[0].Players) > 0 && len(match.Teams[1].Players) > 0 {
			played++
		}
	}

	points := ComputePointsStandings(matches)
	totalPoints, participations := 0, 0
	for _, row := range points {
		totalPoints += row.Points
		participations += row.Played
	}

	result := &models.Agoranking{
		Matches:  played,
		N:        AgorankingN(played),
		Ranked:   []models.AgorankingRow{},
		Unranked: []models.AgorankingRow{},
	}
	if participations > 0 {
		result.GroupAverage = float64(totalPoints) / float64(participations)
	}

	for _, row := range points {
		entry := models.AgorankingRow{
			PlayerID: row.PlayerID,
			Name:     row.Name,
			Played:   row.Played,
			Won:      row.Won,
			Drawn:    row.Drawn,
			Lost:     row.Lost,
			Points:   row.Points,
			Rating:   AgorankingRating(row.Points, row.Played, result.N, result.GroupAverage),
		}
		if row.Played >= result.N {
			result.Ranked = append(result.Ranked, entry)
		} else {
			result.Unranked = append(result.Unranked, entry)
		}
	}

	sortAgorankingRanked(result.Ranked)
	sortAgorankingUnranked(result.Unranked)
	return result
}

// sortAgorankingRanked orders ranked rows by unrounded rating, then by
// matches played (more first), then by name.
func sortAgorankingRanked(rows []models.AgorankingRow) {
	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if math.Abs(a.Rating-b.Rating) > agorankingTieEpsilon {
			return a.Rating > b.Rating
		}
		if a.Played != b.Played {
			return a.Played > b.Played
		}
		return agorankingNameLess(a, b)
	})
}

// sortAgorankingUnranked orders unranked rows by matches played (more
// first), then by points per match, then by name.
func sortAgorankingUnranked(rows []models.AgorankingRow) {
	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.Played != b.Played {
			return a.Played > b.Played
		}
		// Same number of matches: more points means more points per match.
		if a.Points != b.Points {
			return a.Points > b.Points
		}
		return agorankingNameLess(a, b)
	})
}

// agorankingNameLess is the last tie-break: by name, ignoring case (names
// are shown capitalised), then by id so the order never depends on map order.
func agorankingNameLess(a, b models.AgorankingRow) bool {
	if na, nb := strings.ToLower(a.Name), strings.ToLower(b.Name); na != nb {
		return na < nb
	}
	return a.PlayerID.String() < b.PlayerID.String()
}

// AgorankingN is a third of the group's matches, rounded up.
func AgorankingN(groupMatches int) int {
	if groupMatches <= 0 {
		return 0
	}
	return (groupMatches + 2) / 3
}

// AgorankingRating is (points + n × average) ÷ (matches + n), or 0 when
// there is nothing to divide by.
func AgorankingRating(points, matches, n int, average float64) float64 {
	if matches+n == 0 {
		return 0
	}
	return (float64(points) + float64(n)*average) / float64(matches+n)
}
