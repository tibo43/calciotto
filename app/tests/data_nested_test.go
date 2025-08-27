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

	// Récupération des joueurs, équipes et matchs de la base de données
	var dbPlayers []models.Player
	var dbMatches []models.Match
	var dbTeams []models.Team

	db.Find(&dbPlayers)
	db.Find(&dbMatches)
	db.Find(&dbTeams)

	// Création de quelques compositions d'équipe avec des références aléatoires
	for i := 0; i < len(dbMatches); i++ {
		match := dbMatches[i]
		for j := 0; j < len(dbTeams); j++ {
			team := dbTeams[j]
			for h := 0; h < len(dbPlayers)/2+1; h++ {
				player := dbPlayers[rand.Intn(len(dbPlayers))]
				teamComposition := models.TeamComposition{
					MatchID:  match.ID,
					TeamID:   team.ID,
					PlayerID: player.ID,
					Number:   rand.Intn(4),
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
