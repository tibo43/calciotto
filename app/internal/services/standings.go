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

func (s *StandingsService) GetPointsStandings() ([]models.PointsStandingRow, error) {
	matches, err := s.MatchService.GetMatchesDetails()
	if err != nil {
		return nil, err
	}
	return ComputePointsStandings(matches), nil
}

func (s *StandingsService) GetScorers() ([]models.ScorerRow, error) {
	matches, err := s.MatchService.GetMatchesDetails()
	if err != nil {
		return nil, err
	}
	return ComputeScorers(matches), nil
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
