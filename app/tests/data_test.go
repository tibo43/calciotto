package tests

import (
	"log"
	"testing"

	"app/internal/models"
	"app/pkg/common"
	"app/pkg/database"
)

func TestCreateDataset(t *testing.T) {
	// Initialisation de la base de données SQLite pour les tests
	db, err := database.InitDB()
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// // Création de quelques joueurs
	players := []models.Player{
		{BaseModel: models.BaseModel{}, Name: "thibaut"},
		{BaseModel: models.BaseModel{}, Name: "matthias"},
		{BaseModel: models.BaseModel{}, Name: "manfredi"},
		{BaseModel: models.BaseModel{}, Name: "damien"},
		{BaseModel: models.BaseModel{}, Name: "vincent"},
		{BaseModel: models.BaseModel{}, Name: "pierre"},
		{BaseModel: models.BaseModel{}, Name: "anthony"},
		{BaseModel: models.BaseModel{}, Name: "jacopo"},
		{BaseModel: models.BaseModel{}, Name: "mattheo"},
		{BaseModel: models.BaseModel{}, Name: "ryan"},
		{BaseModel: models.BaseModel{}, Name: "connor"},
		{BaseModel: models.BaseModel{}, Name: "marcos"},
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

	users := []models.User{
		{BaseModel: models.BaseModel{}, UserName: "captain", Password: common.HashPassword("azerty"), IsAdmin: true},
		{BaseModel: models.BaseModel{}, UserName: "player", Password: common.HashPassword("azerty"), IsAdmin: false},
	}

	for _, user := range users {
		result := db.Create(&user)
		if result.Error != nil {
			t.Fatalf("Failed to create users: %v", result.Error)
		}
	}

	log.Println("Dataset created successfully!")
}
