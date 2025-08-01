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
		{BaseModel: models.BaseModel{}, Name: "Thibaut"},
		{BaseModel: models.BaseModel{}, Name: "Matthias"},
		{BaseModel: models.BaseModel{}, Name: "Manfredi"},
		{BaseModel: models.BaseModel{}, Name: "Damien"},
		{BaseModel: models.BaseModel{}, Name: "Vincent"},
		{BaseModel: models.BaseModel{}, Name: "Pierre"},
		{BaseModel: models.BaseModel{}, Name: "Anthony"},
		{BaseModel: models.BaseModel{}, Name: "Jacopo"},
		{BaseModel: models.BaseModel{}, Name: "Mattheo"},
		{BaseModel: models.BaseModel{}, Name: "Ryan"},
		{BaseModel: models.BaseModel{}, Name: "Connor"},
		{BaseModel: models.BaseModel{}, Name: "Marco"},
		{BaseModel: models.BaseModel{}, Name: "Niccolo"},
		{BaseModel: models.BaseModel{}, Name: "Augustin"},
		{BaseModel: models.BaseModel{}, Name: "Esteban"},
		{BaseModel: models.BaseModel{}, Name: "Henry"},
	}

	for _, player := range players {
		result := db.Create(&player)
		if result.Error != nil {
			t.Fatalf("Failed to create player: %v", result.Error)
		}
	}

	// Création de quelques équipes
	teams := []models.Team{
		{BaseModel: models.BaseModel{}, Colour: "Black"},
		{BaseModel: models.BaseModel{}, Colour: "White"},
	}

	for _, team := range teams {
		result := db.Create(&team)
		if result.Error != nil {
			t.Fatalf("Failed to create team: %v", result.Error)
		}
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

	log.Println("Dataset created successfully!")
}
