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

func (s *MatchService) CreateMatch(date models.Date) (uuid.UUID, error) {
	match := &models.Match{
		Date: date,
	}
	result := s.DB.Create(match)
	if result.Error != nil {
		return uuid.Nil, result.Error
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
			   match_players.goals_scored as goals_scored
        FROM matches
        LEFT JOIN match_players ON match_players.match_id = matches.id
        LEFT JOIN teams ON teams.id = match_players.team_id
        LEFT JOIN players ON players.id = match_players.player_id
		ORDER BY match_date DESC
    `).Scan(&rowsMatches)

	if result.Error != nil {
		return nil, result.Error
	}

	// Map the flat results into the hierarchical structure
	matchesMap := make(map[uuid.UUID]*models.MatchWithDetails)

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
			ID:          rowMatches.PlayerID,
			Name:        rowMatches.PlayerName,
			GoalsScored: rowMatches.GoalsScored,
		})
	}

	// Fetch all teams once, used below to backfill any team that has no roster yet in a given match.
	var allTeams []models.Team
	if result := s.DB.Find(&allTeams); result.Error != nil {
		return nil, result.Error
	}

	// Convert the map to a slice
	var matches []models.MatchWithDetails
	for _, match := range matchesMap {
		// Filter out teams with missing ID or Colour
		var validTeams []models.TeamWithPlayers
		for _, team := range match.Teams {
			if team.ID != uuid.Nil && team.Colour != "" && len(team.Players) > 0 {
				for _, player := range team.Players {
					// Update the team's score based on the number of goals scored by players
					team.Score += player.GoalsScored
				}
				validTeams = append(validTeams, team)
			}
		}

		// Backfill any team that has no players assigned yet in this match, so a match
		// always shows every team instead of only the ones with a roster so far.
		for _, allTeam := range allTeams {
			found := false
			for _, team := range validTeams {
				if team.ID == allTeam.ID {
					found = true
					break
				}
			}
			if !found {
				validTeams = append(validTeams, models.TeamWithPlayers{
					ID:      allTeam.ID,
					Colour:  allTeam.Colour,
					Score:   0,
					Players: []models.PlayerCustom{},
				})
			}
		}

		// Only include matches with valid teams
		if len(validTeams) > 0 {
			match.Teams = validTeams
			matches = append(matches, *match)
		}
	}

	// Sort matches by date in descending order
	sort.Slice(matches, func(i, j int) bool {
		return matches[j].Date.Before(matches[i].Date)
	})
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
				return team.Players[i].GoalsScored > team.Players[j].GoalsScored
			})
		}
	}
	return matches, nil
}

func (s *MatchService) GetMatchDetailsByID(id uuid.UUID) (*models.MatchWithDetails, error) {

	var rowsMatch []*models.RowsMatchDetails

	// Execute the SQL query and scan the results into a flat structure
	result := s.DB.Raw(`
        SELECT matches.id as match_id, matches.date as match_date, 
               teams.id as team_id, teams.colour as team_colour, 
               players.id as player_id, players.name as player_name,
			   match_players.goals_scored as goals_scored
        FROM matches
        LEFT JOIN match_players ON match_players.match_id = matches.id
        LEFT JOIN teams ON teams.id = match_players.team_id
        LEFT JOIN players ON players.id = match_players.player_id
		WHERE matches.id = ?
		ORDER BY match_date DESC`, id).Scan(&rowsMatch)

	if result.Error != nil {
		return nil, result.Error
	}

	if len(rowsMatch) == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	// Initialisez l'objet match
	match := &models.MatchWithDetails{
		ID:    id,
		Date:  rowsMatch[0].MatchDate,
		Teams: []models.TeamWithPlayers{},
	}

	for _, rowMatch := range rowsMatch {
		if rowMatch.TeamID == uuid.Nil {
			// No match_players row at all for this match yet.
			continue
		}

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
			ID:          rowMatch.PlayerID,
			Name:        rowMatch.PlayerName,
			GoalsScored: rowMatch.GoalsScored,
		})
	}

	// Calculez le score pour chaque équipe
	for i := range match.Teams {
		score := 0
		for _, player := range match.Teams[i].Players {
			score += player.GoalsScored
		}
		match.Teams[i].Score = score
	}

	// Backfill any team that has no players assigned yet in this match, so the match
	// always shows every team instead of only the ones with a roster so far.
	var allTeams []models.Team
	if result := s.DB.Find(&allTeams); result.Error != nil {
		return nil, result.Error
	}
	for _, allTeam := range allTeams {
		found := false
		for _, team := range match.Teams {
			if team.ID == allTeam.ID {
				found = true
				break
			}
		}
		if !found {
			match.Teams = append(match.Teams, models.TeamWithPlayers{
				ID:      allTeam.ID,
				Colour:  allTeam.Colour,
				Score:   0,
				Players: []models.PlayerCustom{},
			})
		}
	}

	// Triez les équipes par couleur (ordre alphabétique)
	if len(match.Teams) > 0 {
		sort.Slice(match.Teams, func(i, j int) bool {
			return match.Teams[i].Colour < match.Teams[j].Colour
		})
	}

	// Triez les joueurs dans chaque équipe par leur nombre de buts en ordre décroissant
	for i := range match.Teams {
		team := &match.Teams[i]
		sort.Slice(team.Players, func(k, l int) bool {
			return team.Players[k].GoalsScored > team.Players[l].GoalsScored
		})
	}

	return match, nil
}

func (s *MatchService) UpdateMatch(match models.MatchWithDetails) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		var dbMatchPlayers []models.MatchPlayer

		for i := range match.Teams {
			team := &match.Teams[i]
			result := tx.Where("match_id = ?", match.ID).Where("team_id = ?", team.ID).Find(&dbMatchPlayers)
			if result.Error != nil {
				return result.Error
			}
			for j := range team.Players {
				player := &team.Players[j]
				// Check if the player already exists in the team composition
				exists := false
				for _, dbMatchPlayer := range dbMatchPlayers {
					if dbMatchPlayer.PlayerID == player.ID {
						exists = true
						break
					}
				}
				if !exists {
					// Create a new team composition if it doesn't exist
					newMatchPlayer := models.MatchPlayer{
						MatchID:     match.ID,
						TeamID:      team.ID,
						PlayerID:    player.ID,
						GoalsScored: player.GoalsScored,
					}
					result := tx.Create(&newMatchPlayer)
					if result.Error != nil {
						return result.Error
					}
				}
			}

			for _, dbMatchPlayer := range dbMatchPlayers {
				toDelete := true
				for j := range team.Players {
					player := &team.Players[j]
					if dbMatchPlayer.PlayerID == player.ID {
						toDelete = false
						result := tx.Model(&models.MatchPlayer{}).Where("match_id = ?", match.ID).Where("team_id = ?", team.ID).Where("player_id = ?", player.ID).Update("goals_scored", player.GoalsScored)
						if result.Error != nil {
							return result.Error
						}
						break
					}
				}
				if toDelete {
					result := tx.Where("match_id = ?", match.ID).Where("team_id = ?", team.ID).Where("player_id = ?", dbMatchPlayer.PlayerID).Delete(&models.MatchPlayer{})
					if result.Error != nil {
						return result.Error
					}
				}
			}
		}
		return nil
	})
}
