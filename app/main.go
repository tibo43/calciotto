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
		AllowOrigins:     []string{"http://localhost:4000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Initialize handlers
	playerHandler := handlers.NewPlayerHandler(services.NewPlayerService(db))
	matchHandler := handlers.NewMatchHandler(services.NewMatchService(db))

	// Setup routes
	// Players
	r.POST("/players", playerHandler.CreatePlayer)
	r.GET("/players", playerHandler.GetPlayers)
	r.GET("/players/search", playerHandler.SearchPlayer)

	// Matches
	r.POST("/matches", matchHandler.CreateMatch)
	r.GET("/matches/details", matchHandler.GetMatchesDetails)
	r.GET("/matches/:id/details", matchHandler.GetMatchDetailsByID)
	r.PUT("/matches/:id", matchHandler.UpdateMatch)
	// Add more routes as needed

	// Start server
	r.Run(":8080")
}
