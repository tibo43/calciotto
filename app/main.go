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

	// Initialize handlers
	playerHandler := handlers.NewPlayerHandler(services.NewPlayerService(db))
	teamHandler := handlers.NewTeamHandler(services.NewTeamService(db))
	matchHandler := handlers.NewMatchHandler(services.NewMatchService(db))
	teamCompositionHandler := handlers.NewTeamCompositionHandler(services.NewTeamCompositionService(db))

	// Setup routes
	// Players
	r.GET("/players", playerHandler.GetPlayers)
	r.POST("/players", playerHandler.CreatePlayer)
	r.GET("/players/:id", playerHandler.GetPlayerByID)
	// Teams
	r.GET("/teams", teamHandler.GetTeams)
	r.POST("/teams", teamHandler.CreateTeam)
	r.GET("/teams/:id", teamHandler.GetTeamByID)
	// Matches
	r.GET("/matches", matchHandler.GetMatches)
	r.POST("/matches", matchHandler.CreateMatch)
	r.GET("/matches/:id", matchHandler.GetMatchByID)
	// Team Composition
	r.POST("/composition", teamCompositionHandler.CreateTeamComposition)
	r.GET("/composition/match/:match_id", teamCompositionHandler.GetTeamCompositionByMatchID)
	r.GET("/composition/team/:team_id", teamCompositionHandler.GetTeamCompositionByTeamID)
	// Add more routes as needed

	// Start server
	r.Run(":8080")
}
