package services

import (
	"app/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MatchService struct {
	DB *gorm.DB
}

func NewMatchService(db *gorm.DB) *MatchService {
	return &MatchService{DB: db}
}

func (s *MatchService) GetMatches() ([]models.Match, error) {
	var matches []models.Match
	result := s.DB.Find(&matches)
	if result.Error != nil {
		return nil, result.Error
	}
	return matches, nil
}

func (s *MatchService) CreateMatch(match *models.Match) error {
	match.ID = uuid.New().String()
	result := s.DB.Create(match)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (s *MatchService) GetMatchByID(id string) (*models.Match, error) {
	var match models.Match
	result := s.DB.First(&match, "id = ?", id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &match, nil
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

func (s *MatchService) GetMatchesWithDetail() ([]MatchWithDetails, error) {
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

	// log.Println(matches)

	return matches, nil
}

func (s *MatchService) GetMatchWithDetailByID(id string) ([]MatchWithDetails, error) {
	var matches []MatchWithDetails

	// Récupérer les matchs et les données associées
	result := s.DB.Table("matches").Where("match_id = ?", id).Select("matches.id, matches.date, teams.id as team_id, teams.colour, players.id as player_id, players.name").
		Joins("left join team_compositions on team_compositions.match_id = matches.id").
		Joins("left join teams on teams.id = team_compositions.team_id").
		Joins("left join players on players.id = team_compositions.player_id").
		Scan(&matches)

	if result.Error != nil {
		return nil, result.Error
	}
	return matches, nil
}
