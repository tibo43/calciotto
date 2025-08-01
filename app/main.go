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
	teamHandler := handlers.NewTeamHandler(services.NewTeamService(db))
	matchHandler := handlers.NewMatchHandler(services.NewMatchService(db))
	teamCompositionHandler := handlers.NewTeamCompositionHandler(services.NewTeamCompositionService(db))
	goalHandler := handlers.NewGoalHandler(services.NewGoalService(db))

	// Setup routes
	// Players
	r.POST("/players", playerHandler.CreatePlayer)
	r.GET("/players/all", playerHandler.GetPlayers)
	r.GET("/players/:id", playerHandler.GetPlayerByID)
	// Teams
	r.POST("/teams", teamHandler.CreateTeam)
	r.GET("/teams/all", teamHandler.GetTeams)
	r.GET("/teams/:id", teamHandler.GetTeamByID)
	// Matches
	r.POST("/matches", matchHandler.CreateMatch)
	r.GET("/matches/all", matchHandler.GetMatches)
	r.GET("/matches/:id", matchHandler.GetMatchByID)
	r.GET("/matches/details/all", matchHandler.GetMatchesWithDetail)
	r.GET("/matches/details/:id", matchHandler.GetMatchWithDetailByID)
	// Team Composition
	r.POST("/compositions", teamCompositionHandler.CreateTeamComposition)
	r.GET("/compositions/all", teamCompositionHandler.GetTeamCompositionAll)
	r.GET("/compositions/match/:match_id", teamCompositionHandler.GetTeamCompositionByMatchID)
	r.GET("/compositions/team/:team_id", teamCompositionHandler.GetTeamCompositionByTeamID)
	// Team Composition
	r.POST("/goals", goalHandler.CreateGoal)
	r.GET("/goals/all", goalHandler.GetGoalAll)
	r.GET("/goals/match/:match_id", goalHandler.GetGoalByMatchID)
	r.GET("/goals/player/:player_id", goalHandler.GetGoalByPlayerID)
	r.GET("/goals/team/:team_id", goalHandler.GetGoalByTeamID)
	// Add more routes as needed

	// Start server
	r.Run(":8080")
}
