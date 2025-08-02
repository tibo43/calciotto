package services

import (
	"app/internal/models"
	"sort"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MatchService struct {
	DB *gorm.DB
}

func NewMatchService(db *gorm.DB) *MatchService {
	return &MatchService{DB: db}
}

func (s *MatchService) CreateMatch(match *models.Match) error {
	match.ID = uuid.New().String()
	result := s.DB.Create(match)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// Player représente un joueur.
type Player struct {
	ID         string `gorm:"type:uuid;primaryKey"`
	Name       string `gorm:"type:string"`
	GoalNumber int    `gorm:"type:int"`
}

// TeamWithPlayers représente une équipe avec ses joueurs.
type TeamWithPlayers struct {
	ID      string   `gorm:"type:uuid;primaryKey"`
	Colour  string   `gorm:"type:string"`
	Score   int      `gorm:"type:int"`
	Players []Player `gorm:"foreignKey:TeamID"` // Indique à GORM que cette table a une relation avec Player
}

// MatchWithDetails représente un match avec ses équipes et joueurs.
type MatchWithDetails struct {
	ID    string            `gorm:"type:uuid;primaryKey"`
	Date  string            `gorm:"type:string"`
	Teams []TeamWithPlayers `gorm:"foreignKey:MatchID"` // Indique à GORM que cette table a une relation avec TeamWithPlayers
}

func (s *MatchService) GetMatchesDetails() ([]MatchWithDetails, error) {
	var rows []struct {
		MatchID    string
		MatchDate  string
		TeamID     string
		TeamColour string
		Score      int
		PlayerID   string
		PlayerName string
		GoalNumber int
	}

	// Execute the SQL query and scan the results into a flat structure
	result := s.DB.Raw(`
        SELECT matches.id as match_id, matches.date as match_date, 
               teams.id as team_id, teams.colour as team_colour, 
               players.id as player_id, players.name as player_name,
			   goals.number as goal_number
        FROM matches
        LEFT JOIN team_compositions ON team_compositions.match_id = matches.id
        LEFT JOIN teams ON teams.id = team_compositions.team_id
        LEFT JOIN players ON players.id = team_compositions.player_id
		LEFT JOIN goals ON goals.match_id = matches.id AND goals.team_id = teams.id AND goals.player_id = players.id
		ORDER BY match_date DESC
    `).Scan(&rows)

	if result.Error != nil {
		return nil, result.Error
	}

	// Map the flat results into the hierarchical structure
	matchesMap := make(map[string]*MatchWithDetails)

	for _, row := range rows {
		// Get or create the match
		match, exists := matchesMap[row.MatchID]
		if !exists {
			match = &MatchWithDetails{
				ID:    row.MatchID,
				Date:  row.MatchDate,
				Teams: []TeamWithPlayers{},
			}
			matchesMap[row.MatchID] = match
		}

		// Get or create the team
		var team *TeamWithPlayers
		for i := range match.Teams {
			if match.Teams[i].ID == row.TeamID {
				team = &match.Teams[i]
				break
			}
		}
		if team == nil {
			team = &TeamWithPlayers{
				ID:      row.TeamID,
				Colour:  row.TeamColour,
				Score:   0,
				Players: []Player{},
			}
			match.Teams = append(match.Teams, *team)
		}

		// Add the player to the team
		team.Players = append(team.Players, Player{
			ID:         row.PlayerID,
			Name:       row.PlayerName,
			GoalNumber: row.GoalNumber,
		})
	}

	// Convert the map to a slice
	var matches []MatchWithDetails
	for _, match := range matchesMap {
		// Filter out teams with missing ID or Colour
		var validTeams []TeamWithPlayers
		for _, team := range match.Teams {
			if team.ID != "" && team.Colour != "" && len(team.Players) > 0 {
				for _, player := range team.Players {
					// Update the team's score based on the number of goals scored by players
					team.Score += player.GoalNumber
				}
				validTeams = append(validTeams, team)
			}
		}

		// Only include matches with valid teams
		if len(validTeams) > 0 {
			match.Teams = validTeams
			matches = append(matches, *match)
		}
	}

	// Sort matches by date in descending order
	for i := 0; i < len(matches)-1; i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[i].Date < matches[j].Date {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}
	// Sort matched teams same colour in first position
	for _, match := range matches {
		sort.Slice(match.Teams, func(i, j int) bool {
			return match.Teams[i].Colour < match.Teams[j].Colour
		})
	}

	// sort players in each team by their goal number in descending order
	for _, match := range matches {
		for _, team := range match.Teams {
			sort.Slice(team.Players, func(i, j int) bool {
				return team.Players[i].GoalNumber > team.Players[j].GoalNumber
			})
		}
	}

	return matches, nil
}

func (s *MatchService) GetMatchDetailsByID(id string) (*MatchWithDetails, error) {
	var rows []struct {
		MatchID    string
		MatchDate  string
		TeamID     string
		TeamColour string
		Score      int
		PlayerID   string
		PlayerName string
		GoalNumber int
	}

	// Execute the SQL query and scan the results into a flat structure
	result := s.DB.Raw(`
        SELECT matches.id as match_id, matches.date as match_date, 
               teams.id as team_id, teams.colour as team_colour, 
               players.id as player_id, players.name as player_name,
			   goals.number as goal_number
        FROM matches
        LEFT JOIN team_compositions ON team_compositions.match_id = matches.id
        LEFT JOIN teams ON teams.id = team_compositions.team_id
        LEFT JOIN players ON players.id = team_compositions.player_id
		LEFT JOIN goals ON goals.match_id = matches.id AND goals.team_id = teams.id AND goals.player_id = players.id
		WHERE matches.id = ?
		ORDER BY match_date DESC`, id).Scan(&rows)

	if result.Error != nil {
		return nil, result.Error
	}

	// Initialisez l'objet match
	match := &MatchWithDetails{
		ID:    id,
		Date:  rows[0].MatchDate,
		Teams: []TeamWithPlayers{},
	}

	for _, row := range rows {
		// Get or create the team
		var team *TeamWithPlayers
		for i := range match.Teams {
			if match.Teams[i].ID == row.TeamID {
				team = &match.Teams[i]
				break
			}
		}

		if team == nil {
			team = &TeamWithPlayers{
				ID:      row.TeamID,
				Colour:  row.TeamColour,
				Score:   0,
				Players: []Player{},
			}
			match.Teams = append(match.Teams, *team)
		}

		team.Players = append(team.Players, Player{
			ID:         row.PlayerID,
			Name:       row.PlayerName,
			GoalNumber: row.GoalNumber,
		})

	}

	// Calculez le score pour chaque équipe
	for i := range match.Teams {
		score := 0
		for _, player := range match.Teams[i].Players {
			score += player.GoalNumber
		}
		match.Teams[i].Score = score
	}

	// Triez les équipes par couleur, en plaçant une couleur aléatoire en première position
	if len(match.Teams) > 0 {
		// Triez les équipes par couleur
		sort.Slice(match.Teams, func(i, j int) bool {
			return match.Teams[i].Colour < match.Teams[j].Colour
		})
	}

	// Triez les joueurs dans chaque équipe par leur nombre de buts en ordre décroissant
	for i := range match.Teams {
		team := &match.Teams[i]
		sort.Slice(team.Players, func(k, l int) bool {
			return team.Players[k].GoalNumber > team.Players[l].GoalNumber
		})
	}

	return match, nil
}
