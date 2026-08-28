package database

import (
	"fmt"
	"log"
	"os"

	"app/internal/models"

	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func InitDB() (*gorm.DB, error) {
	host := getEnv("DB_HOST", "db")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "calciotto")
	password := os.Getenv("DB_PASSWORD")
	dbname := getEnv("DB_NAME", "calciotto")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	log.Println("Successfully connected to the database!")

	// Auto-migration des modèles pour créer les tables
	err = db.AutoMigrate(&models.Group{}, &models.Player{}, &models.Team{}, &models.Match{}, &models.MatchPlayer{}, &models.GroupMembership{})
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Tables created successfully!")

	return db, nil
}
