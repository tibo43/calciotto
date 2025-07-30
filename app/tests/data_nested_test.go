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
	var dbTeams []models.Team
	var dbMatches []models.Match

	db.Find(&dbPlayers)
	db.Find(&dbTeams)
	db.Find(&dbMatches)

	// Création de quelques compositions d'équipe avec des références aléatoires
	for i := 0; i < 5; i++ {
		match := dbMatches[rand.Intn(len(dbMatches))]
		team := dbTeams[rand.Intn(len(dbTeams))]
		player := dbPlayers[rand.Intn(len(dbPlayers))]

		teamComposition := models.TeamComposition{
			MatchID:  match.ID,
			TeamID:   team.ID,
			PlayerID: player.ID,
		}

		result := db.Create(&teamComposition)
		if result.Error != nil {
			t.Fatalf("Failed to create team composition: %v", result.Error)
		}
	}

	for i := 0; i < 5; i++ {
		match := dbMatches[rand.Intn(len(dbMatches))]
		team := dbTeams[rand.Intn(len(dbTeams))]
		player := dbPlayers[rand.Intn(len(dbPlayers))]

		goal := models.Goal{
			MatchID:  match.ID,
			TeamID:   team.ID,
			PlayerID: player.ID,
			Number:   i,
		}

		result := db.Create(&goal)
		if result.Error != nil {
			t.Fatalf("Failed to create goal: %v", result.Error)
		}
	}

	log.Println("Dataset created successfully!")
}
