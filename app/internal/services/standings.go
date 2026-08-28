package services

import (
	"sort"

	"app/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type StandingsService struct {
	MatchService *MatchService
}

func NewStandingsService(db *gorm.DB) *StandingsService {
	return &StandingsService{MatchService: NewMatchService(db)}
}

func (s *StandingsService) GetPointsStandings(groupID uuid.UUID, season string) ([]models.PointsStandingRow, error) {
	matches, err := s.MatchService.GetMatchesDetails(groupID)
	if err != nil {
		return nil, err
	}
	return ComputePointsStandings(FilterMatchesBySeason(matches, season)), nil
}

func (s *StandingsService) GetScorers(groupID uuid.UUID, season string) ([]models.ScorerRow, error) {
	matches, err := s.MatchService.GetMatchesDetails(groupID)
	if err != nil {
		return nil, err
	}
	return ComputeScorers(FilterMatchesBySeason(matches, season)), nil
}

// GetSeasons lists the seasons a group actually has matches in. Seasons have
// no table of their own — they're derived from the match dates already loaded
// for the group, so no extra query is needed.
func (s *StandingsService) GetSeasons(groupID uuid.UUID) ([]string, error) {
	matches, err := s.MatchService.GetMatchesDetails(groupID)
	if err != nil {
		return nil, err
	}
	return ComputeSeasons(matches), nil
}

// FilterMatchesBySeason keeps only the matches belonging to season (a label as
// produced by models.SeasonOf). An empty season means "no filtering", so
// callers that don't scope by season keep the previous behaviour.
func FilterMatchesBySeason(matches []models.MatchWithDetails, season string) []models.MatchWithDetails {
	if season == "" {
		return matches
	}
	filtered := make([]models.MatchWithDetails, 0, len(matches))
	for _, match := range matches {
		if models.SeasonOf(match.Date) == season {
			filtered = append(filtered, match)
		}
	}
	return filtered
}

// ComputeSeasons returns the sorted, deduplicated season labels covered by an
// already-loaded set of matches. Like the Compute* functions below, it is a
// pure function of its input: which matches to consider is the caller's call.
func ComputeSeasons(matches []models.MatchWithDetails) []string {
	seen := make(map[string]bool, len(matches))
	seasons := make([]string, 0, len(matches))
	for _, match := range matches {
		season := models.SeasonOf(match.Date)
		if seen[season] {
			continue
		}
		seen[season] = true
		seasons = append(seasons, season)
	}
	sort.Strings(seasons)
	return seasons
}

// ComputePointsStandings aggregates win/draw/loss points (3/1/0) per player over
// an already-loaded set of matches. It is a pure function of its input — data
// selection (which matches to include, e.g. scoped to a group) is the caller's
// responsibility, so that scope can change later without touching this logic.
func ComputePointsStandings(matches []models.MatchWithDetails) []models.PointsStandingRow {
	type acc struct {
		name                                    string
		played, won, drawn, lost, goals, points int
	}
	stats := make(map[uuid.UUID]*acc)

	rowFor := func(id uuid.UUID, name string) *acc {
		row, ok := stats[id]
		if !ok {
			row = &acc{name: name}
			stats[id] = row
		}
		return row
	}

	applyResult := func(team models.TeamWithPlayers, result string) {
		for _, player := range team.Players {
			row := rowFor(player.ID, player.Name)
			row.played++
			row.goals += player.GoalsScored
			switch result {
			case "won":
				row.won++
				row.points += 3
			case "drawn":
				row.drawn++
				row.points++
			default:
				row.lost++
			}
		}
	}

	for _, match := range matches {
		// Only count matches where both sides have a full roster.
		if len(match.Teams) != 2 {
			continue
		}
		teamA, teamB := match.Teams[0], match.Teams[1]
		if len(teamA.Players) == 0 || len(teamB.Players) == 0 {
			continue
		}

		switch {
		case teamA.Score > teamB.Score:
			applyResult(teamA, "won")
			applyResult(teamB, "lost")
		case teamB.Score > teamA.Score:
			applyResult(teamB, "won")
			applyResult(teamA, "lost")
		default:
			applyResult(teamA, "drawn")
			applyResult(teamB, "drawn")
		}
	}

	rows := make([]models.PointsStandingRow, 0, len(stats))
	for id, row := range stats {
		rows = append(rows, models.PointsStandingRow{
			PlayerID: id,
			Name:     row.name,
			Played:   row.played,
			Won:      row.won,
			Drawn:    row.drawn,
			Lost:     row.lost,
			GoalsFor: row.goals,
			Points:   row.points,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Points != rows[j].Points {
			return rows[i].Points > rows[j].Points
		}
		if rows[i].GoalsFor != rows[j].GoalsFor {
			return rows[i].GoalsFor > rows[j].GoalsFor
		}
		return rows[i].Name < rows[j].Name
	})
	return rows
}

// ComputeScorers totals goals scored per player over an already-loaded set of
// matches. Like ComputePointsStandings, it takes the match slice as input so
// scoping (e.g. to a group) is the caller's concern, not this function's.
func ComputeScorers(matches []models.MatchWithDetails) []models.ScorerRow {
	type acc struct {
		name          string
		played, goals int
	}
	stats := make(map[uuid.UUID]*acc)

	for _, match := range matches {
		for _, team := range match.Teams {
			for _, player := range team.Players {
				row, ok := stats[player.ID]
				if !ok {
					row = &acc{name: player.Name}
					stats[player.ID] = row
				}
				row.played++
				row.goals += player.GoalsScored
			}
		}
	}

	rows := make([]models.ScorerRow, 0, len(stats))
	for id, row := range stats {
		rows = append(rows, models.ScorerRow{
			PlayerID: id,
			Name:     row.name,
			Played:   row.played,
			Goals:    row.goals,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Goals != rows[j].Goals {
			return rows[i].Goals > rows[j].Goals
		}
		return rows[i].Name < rows[j].Name
	})
	return rows
}
