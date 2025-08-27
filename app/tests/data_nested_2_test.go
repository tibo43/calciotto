package tests

import (
	"log"
	"math/rand"
	"testing"

	"app/internal/models"
	"app/pkg/database"
)

func TestCreateDatasetNested2(t *testing.T) {
	// Initialisation de la base de données PostgreSQL pour les tests
	db, err := database.InitDB()
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Création de quelques matchs
	match := models.Match{BaseModel: models.BaseModel{}, Date: "2025-10-29"}
	log.Println("Creating match:", match)
	result := db.Create(&match)
	if result.Error != nil {
		t.Fatalf("Failed to create match: %v", result.Error)
	}
	// Récupération des joueurs, équipes et matchs de la base de données
	var dbPlayers []models.Player
	var dbTeams []models.Team

	db.Find(&dbPlayers)
	db.Find(&dbTeams)

	// Création de quelques compositions d'équipe avec des références aléatoires
	for j := 0; j < len(dbTeams); j++ {
		team := dbTeams[j]
		for h := 0; h < len(dbPlayers)/2+1; h++ {
			player := dbPlayers[rand.Intn(len(dbPlayers))]
			teamComposition := models.TeamComposition{
				MatchID:  match.ID,
				TeamID:   team.ID,
				PlayerID: player.ID,
				Number:   0,
			}
			result := db.Create(&teamComposition)
			if result.Error != nil {
				t.Fatalf("Failed to create team composition: %v", result.Error)
			}
		}
	}

	log.Println("Dataset created successfully!")
}
