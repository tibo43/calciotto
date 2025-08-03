package services

import (
	"app/internal/models"
	"encoding/json"
	"log"
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

func (s *MatchService) GetMatchesDetails() ([]models.MatchWithDetails, error) {
	var rows []models.RowsMatchDetails

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
	matchesMap := make(map[string]*models.MatchWithDetails)

	for _, row := range rows {
		// Get or create the match
		match, exists := matchesMap[row.MatchID]
		if !exists {
			match = &models.MatchWithDetails{
				ID:    row.MatchID,
				Date:  row.MatchDate,
				Teams: []models.TeamWithPlayers{},
			}
			matchesMap[row.MatchID] = match
		}

		// Get or create the team
		var team *models.TeamWithPlayers
		for i := range match.Teams {
			if match.Teams[i].ID == row.TeamID {
				team = &match.Teams[i]
				break
			}
		}
		if team == nil {
			team = &models.TeamWithPlayers{
				ID:      row.TeamID,
				Colour:  row.TeamColour,
				Score:   0,
				Players: []models.PlayerCustom{},
			}
			match.Teams = append(match.Teams, *team)
		}

		// Add the player to the team
		team.Players = append(team.Players, models.PlayerCustom{
			ID:         row.PlayerID,
			Name:       row.PlayerName,
			GoalNumber: row.GoalNumber,
		})
	}

	// Convert the map to a slice
	var matches []models.MatchWithDetails
	for _, match := range matchesMap {
		// Filter out teams with missing ID or Colour
		var validTeams []models.TeamWithPlayers
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

func (s *MatchService) GetMatchDetailsByID(id string) (*models.MatchWithDetails, error) {
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
	match := &models.MatchWithDetails{
		ID:    id,
		Date:  rows[0].MatchDate,
		Teams: []models.TeamWithPlayers{},
	}

	for _, row := range rows {
		// Get or create the team
		var team *models.TeamWithPlayers
		for i := range match.Teams {
			if match.Teams[i].ID == row.TeamID {
				team = &match.Teams[i]
				break
			}
		}

		if team == nil {
			team = &models.TeamWithPlayers{
				ID:      row.TeamID,
				Colour:  row.TeamColour,
				Score:   0,
				Players: []models.PlayerCustom{},
			}
			match.Teams = append(match.Teams, *team)
		}

		team.Players = append(team.Players, models.PlayerCustom{
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

func (s *MatchService) UpdateMatch(match models.MatchWithDetails) error {
	b, err := json.MarshalIndent(match, "", "  ")
	if err != nil {
		log.Println(err)
	}
	log.Print(string(b))

	// var dbTeamCompositions []models.TeamComposition

	// for i := range match.Teams {
	// 	team := &match.Teams[i]
	// 	result := s.DB.Where("match_id = ?", match.ID).Where("team_id = ?", team.ID).Find(&dbTeamCompositions)
	// 	if result.Error != nil {
	// 		return result.Error
	// 	}
	// 	toDelete := []string{}
	// 	for j := range team.Players {
	// 		player := &team.Players[j]
	// 		// Check if the player already exists in the team composition
	// 		exists := false
	// 		for _, dbTeamComposition := range dbTeamCompositions {
	// 			if dbTeamComposition.PlayerID != player.ID {
	// 				toDelete = append(toDelete, dbTeamComposition.PlayerID)
	// 			}
	// 			if dbTeamComposition.PlayerID == player.ID {
	// 				exists = true
	// 				toDelete = common.RemoveValueSlice(toDelete, dbTeamComposition.PlayerID)
	// 				break
	// 			}
	// 		}
	// 		if !exists {
	// 			// Create a new team composition if it doesn't exist
	// 			newTeamComposition := models.TeamComposition{
	// 				MatchID:  match.ID,
	// 				TeamID:   team.ID,
	// 				PlayerID: player.ID,
	// 			}
	// 			result := s.DB.Create(&newTeamComposition)
	// 			if result.Error != nil {
	// 				return result.Error
	// 			}
	// 		}
	// 		// drop the player if it is not in the team anymore
	// 		if len(toDelete) > 0 {
	// 			result := s.DB.Where("match_id = ?", match.ID).Where("team_id = ?", team.ID).Where("player_id IN ?", toDelete).Delete(&models.TeamComposition{})
	// 			if result.Error != nil {
	// 				return result.Error
	// 			}
	// 		}
	// 	}
	// }
	return nil
}
