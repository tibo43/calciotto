package tests

import (
	"log"
	"testing"

	"app/internal/models"
	"app/pkg/database"
)

func TestCreateDataset(t *testing.T) {
	// Initialisation de la base de données SQLite pour les tests
	db, err := database.InitDB()
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Création de quelques joueurs
	players := []models.Player{
		{BaseModel: models.BaseModel{}, Name: "Thibaut Fabre"},
		{BaseModel: models.BaseModel{}, Name: "Alice Dupont"},
		{BaseModel: models.BaseModel{}, Name: "Jean Martin"},
	}

	for _, player := range players {
		result := db.Create(&player)
		if result.Error != nil {
			t.Fatalf("Failed to create player: %v", result.Error)
		}
	}

	// Création de quelques équipes
	teams := []models.Team{
		{BaseModel: models.BaseModel{}, Colour: "Rouge"},
		{BaseModel: models.BaseModel{}, Colour: "Bleu"},
	}

	for _, team := range teams {
		result := db.Create(&team)
		if result.Error != nil {
			t.Fatalf("Failed to create team: %v", result.Error)
		}
	}

	// Création de quelques matchs
	matches := []models.Match{
		{BaseModel: models.BaseModel{}, Date: "2023-10-01"},
		{BaseModel: models.BaseModel{}, Date: "2023-10-02"},
	}

	for _, match := range matches {
		result := db.Create(&match)
		if result.Error != nil {
			t.Fatalf("Failed to create match: %v", result.Error)
		}
	}

	log.Println("Dataset created successfully!")
}
