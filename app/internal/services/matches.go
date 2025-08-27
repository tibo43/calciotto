package services

import (
	"app/internal/models"
	"sort"

	"gorm.io/gorm"
)

type MatchService struct {
	DB *gorm.DB
}

func NewMatchService(db *gorm.DB) *MatchService {
	return &MatchService{DB: db}
}

func (s *MatchService) CreateMatch(date string) (string, error) {
	match := &models.Match{
		BaseModel: models.BaseModel{},
		Date:      date,
	}
	result := s.DB.Create(match)
	if result.Error != nil {
		return "", result.Error
	}
	return match.ID, nil
}

func (s *MatchService) GetMatchesDetails() ([]models.MatchWithDetails, error) {
	var rowsMatches []models.RowsMatchDetails

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
    `).Scan(&rowsMatches)

	if result.Error != nil {
		return nil, result.Error
	}

	// Map the flat results into the hierarchical structure
	matchesMap := make(map[string]*models.MatchWithDetails)

	for _, rowMatches := range rowsMatches {
		// Get or create the match
		match, exists := matchesMap[rowMatches.MatchID]
		if !exists {
			match = &models.MatchWithDetails{
				ID:    rowMatches.MatchID,
				Date:  rowMatches.MatchDate,
				Teams: []models.TeamWithPlayers{},
			}
			matchesMap[rowMatches.MatchID] = match
		}

		// Get or create the team
		var team *models.TeamWithPlayers
		for i := range match.Teams {
			if match.Teams[i].ID == rowMatches.TeamID {
				team = &match.Teams[i]
				break
			}
		}
		if team == nil {
			newTeam := models.TeamWithPlayers{
				ID:      rowMatches.TeamID,
				Colour:  rowMatches.TeamColour,
				Score:   0,
				Players: []models.PlayerCustom{},
			}
			match.Teams = append(match.Teams, newTeam)
			team = &match.Teams[len(match.Teams)-1] // Point to the newly appended team
		}

		// Add the player to the team
		team.Players = append(team.Players, models.PlayerCustom{
			ID:         rowMatches.PlayerID,
			Name:       rowMatches.PlayerName,
			GoalNumber: rowMatches.GoalNumber,
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

		if len(validTeams) == 0 {
			var rowsTeams []models.RowsTeamDetails
			result := s.DB.Raw(`
					SELECT teams.id as team_id, teams.colour as team_colour
					FROM teams
				`).Scan(&rowsTeams)
			if result.Error != nil {
				return nil, result.Error
			}
			for _, rowTeams := range rowsTeams {
				var team = &models.TeamWithPlayers{
					ID:      rowTeams.TeamID,
					Colour:  rowTeams.TeamColour,
					Score:   0,
					Players: []models.PlayerCustom{},
				}
				validTeams = append(validTeams, *team)
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

	var rowsMatch []*models.RowsMatchDetails

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
		ORDER BY match_date DESC`, id).Scan(&rowsMatch)

	if result.Error != nil {
		return nil, result.Error
	}

	// Initialisez l'objet match
	match := &models.MatchWithDetails{
		ID:    id,
		Date:  rowsMatch[0].MatchDate,
		Teams: []models.TeamWithPlayers{},
	}

	if len(rowsMatch) > 1 {
		for _, rowMatch := range rowsMatch {
			// Get or create the team
			var team *models.TeamWithPlayers
			for i := range match.Teams {
				if match.Teams[i].ID == rowMatch.TeamID {
					team = &match.Teams[i]
					break
				}
			}

			if team == nil {
				newTeam := models.TeamWithPlayers{
					ID:      rowMatch.TeamID,
					Colour:  rowMatch.TeamColour,
					Score:   0,
					Players: []models.PlayerCustom{},
				}
				match.Teams = append(match.Teams, newTeam)
				team = &match.Teams[len(match.Teams)-1] // Point to the newly appended team
			}

			team.Players = append(team.Players, models.PlayerCustom{
				ID:         rowMatch.PlayerID,
				Name:       rowMatch.PlayerName,
				GoalNumber: rowMatch.GoalNumber,
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
	} else {
		var rowsTeams []models.RowsTeamDetails
		result := s.DB.Raw(`
				SELECT teams.id as team_id, teams.colour as team_colour
				FROM teams
			`).Scan(&rowsTeams)
		if result.Error != nil {
			return nil, result.Error
		}
		for _, rowTeams := range rowsTeams {
			var team = &models.TeamWithPlayers{
				ID:      rowTeams.TeamID,
				Colour:  rowTeams.TeamColour,
				Score:   0,
				Players: []models.PlayerCustom{},
			}
			match.Teams = append(match.Teams, *team)
		}
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

	var dbTeamCompositions []models.TeamComposition

	for i := range match.Teams {
		team := &match.Teams[i]
		result := s.DB.Where("match_id = ?", match.ID).Where("team_id = ?", team.ID).Find(&dbTeamCompositions)
		if result.Error != nil {
			return result.Error
		}
		for j := range team.Players {
			player := &team.Players[j]
			// Check if the player already exists in the team composition
			exists := false
			for _, dbTeamComposition := range dbTeamCompositions {
				if dbTeamComposition.PlayerID == player.ID {
					exists = true
					break
				}
			}
			if !exists {
				// Create a new team composition if it doesn't exist
				newTeamComposition := models.TeamComposition{
					MatchID:  match.ID,
					TeamID:   team.ID,
					PlayerID: player.ID,
				}
				result := s.DB.Create(&newTeamComposition)
				if result.Error != nil {
					return result.Error
				}
			}
		}

		for _, dbTeamComposition := range dbTeamCompositions {
			toDelete := false
			for j := range team.Players {
				player := &team.Players[j]
				if dbTeamComposition.PlayerID != player.ID {
					toDelete = true
				}
				if dbTeamComposition.PlayerID == player.ID {
					toDelete = false
					break
				}
			}
			if toDelete {
				result := s.DB.Where("match_id = ?", match.ID).Where("team_id = ?", team.ID).Where("player_id = ?", dbTeamComposition.PlayerID).Delete(&models.TeamComposition{})
				if result.Error != nil {
					return result.Error
				}
			}
		}
	}
	return nil
}
