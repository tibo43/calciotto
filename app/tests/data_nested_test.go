package tests

import (
	"log"
	"math/rand"
	"testing"

	"app/internal/models"
	"app/pkg/database"
)

func TestCreateDatasetNested(t *testing.T) {
	// Initialisation de la base de données PostgreSQL pour les tests
	db, err := database.InitDB()
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Création de quelques matchs
	matches := []models.Match{
		{BaseModel: models.BaseModel{}, Date: "2025-10-01"},
		{BaseModel: models.BaseModel{}, Date: "2025-10-08"},
		{BaseModel: models.BaseModel{}, Date: "2025-10-15"},
		{BaseModel: models.BaseModel{}, Date: "2025-10-22"},
	}
	for _, match := range matches {
		result := db.Create(&match)
		if result.Error != nil {
			t.Fatalf("Failed to create match: %v", result.Error)
		}
	}

	// Récupération des joueurs, équipes de la base de données
	var dbPlayers []models.Player
	var dbTeams []models.Team

	db.Find(&dbPlayers)
	db.Find(&dbTeams)

	// Création de quelques compositions d'équipe avec des références aléatoires
	for i := 0; i < len(matches); i++ {
		match := matches[i]
		for j := 0; j < len(dbTeams); j++ {
			team := dbTeams[j]
			for h := 0; h < len(dbPlayers)/2+1; h++ {
				player := dbPlayers[rand.Intn(len(dbPlayers))]
				teamComposition := models.MatchPlayer{
					MatchID:     match.ID,
					TeamID:      team.ID,
					PlayerID:    player.ID,
					GoalsScored: rand.Intn(4),
				}
				result := db.Create(&teamComposition)
				if result.Error != nil {
					t.Fatalf("Failed to create team composition: %v", result.Error)
				}
			}
		}
	}

	log.Println("Dataset created successfully!")
}
