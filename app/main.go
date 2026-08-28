package main

import (
	"log"
	"os"

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

	// Initialize Gin router
	r := gin.Default()

	// Configuration CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:4000", "http://127.0.0.1:4000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// JWT_SECRET has no hardcoded fallback: an empty signing key would let
	// anyone forge tokens, so fail fast at startup instead of running insecure.
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	// Initialize handlers
	groupService := services.NewGroupService(db)
	groupMembershipService := services.NewGroupMembershipService(db)
	playerHandler := handlers.NewPlayerHandler(services.NewPlayerService(db), groupService, groupMembershipService)
	groupHandler := handlers.NewGroupHandler(groupService, groupMembershipService)
	teamHandler := handlers.NewTeamHandler(services.NewTeamService(db))
	matchHandler := handlers.NewMatchHandler(services.NewMatchService(db), groupService)
	standingsHandler := handlers.NewStandingsHandler(services.NewStandingsService(db), groupService)
	authHandler := handlers.NewAuthHandler(services.NewAuthService(db, jwtSecret))

	// Setup routes
	// Players
	r.POST("/players", playerHandler.CreatePlayer)
	r.GET("/players", playerHandler.GetPlayers)
	r.GET("/players/search", playerHandler.SearchPlayer)

	// Groups
	r.POST("/groups", groupHandler.CreateGroup)
	r.GET("/groups", groupHandler.GetGroups)
	r.GET("/groups/:id", groupHandler.GetGroupByID)
	r.GET("/groups/:id/teams", teamHandler.GetTeamsByGroup)
	r.POST("/groups/:id/players", groupHandler.AddPlayerToGroup)
	r.GET("/groups/:id/players", groupHandler.GetGroupMembers)

	// Matches
	r.POST("/matches", matchHandler.CreateMatch)
	r.GET("/matches/details", matchHandler.GetMatchesDetails)
	r.GET("/matches/:id/details", matchHandler.GetMatchDetailsByID)
	r.PUT("/matches/:id", matchHandler.UpdateMatch)

	// Standings
	r.GET("/standings/points", standingsHandler.GetPointsStandings)
	r.GET("/standings/scorers", standingsHandler.GetScorers)

	// Auth
	r.POST("/auth/signup", authHandler.Signup)
	r.POST("/auth/login", authHandler.Login)
	// Add more routes as needed

	// Start server
	r.Run(":8080")
}
