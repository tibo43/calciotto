package database

import (
	"fmt"
	"log"

	"app/internal/models"

	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	//host     = "localhost"
	//host     = "192.168.1.10"
	host     = "db"
	port     = 5432
	user     = "calciotto"
	password = "lee7ohnai9queeDoosh6"
	dbname   = "calciotto"
)

func InitDB() (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	log.Println("Successfully connected to the database!")

	// Auto-migration des modèles pour créer les tables
	err = db.AutoMigrate(&models.Player{}, &models.Team{}, &models.Match{}, &models.MatchPlayer{}, &models.User{})
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Tables created successfully!")

	return db, nil
}
