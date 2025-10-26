package main

import (
	"log"

	"app/internal/handlers"
	"app/internal/services"

	"app/pkg/database"

	"github.com/gin-contrib/cors"
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
	// Configuration CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:4000", "http://127.0.0.1:4000", "https://*.koyeb.app"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		AllowWildcard:    true,
	}))

	// Initialize handlers
	playerHandler := handlers.NewPlayerHandler(services.NewPlayerService(db))
	matchHandler := handlers.NewMatchHandler(services.NewMatchService(db))

	// Setup routes
	// Players
	r.POST("/api/players", playerHandler.CreatePlayer)
	r.GET("/api/players", playerHandler.GetPlayers)
	r.GET("/api/players/search", playerHandler.SearchPlayer)

	// Matches
	r.POST("/api/matches", matchHandler.CreateMatch)
	r.GET("/api/matches/details", matchHandler.GetMatchesDetails)
	r.GET("/api/matches/:id/details", matchHandler.GetMatchDetailsByID)
	r.PUT("/api/matches/:id", matchHandler.UpdateMatch)
	// Add more routes as needed

	// Start server
	r.Run(":8080")
}
