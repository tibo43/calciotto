package main

import (
	"log"

	"app/internal/handlers"
	"app/internal/services"
	"app/pkg/database"

	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize database connection
	db, err := database.InitDB()
	if err != nil {
		log.Fatal(err)
	}
	log.Println(db)

	// Initialize Gin router
	r := gin.Default()

	// Initialisation du service et du gestionnaire
	playerService := services.NewPlayerService(db)
	playerHandler := handlers.NewPlayerHandler(playerService)

	// Setup routes
	r.GET("/players", playerHandler.GetPlayers)
	r.POST("/players", playerHandler.CreatePlayer)
	// Add more routes as needed

	// Start server
	r.Run(":8080")
}
