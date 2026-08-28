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
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
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
	matchHandler := handlers.NewMatchHandler(services.NewMatchService(db), groupMembershipService)
	standingsHandler := handlers.NewStandingsHandler(services.NewStandingsService(db), groupMembershipService)
	authService := services.NewAuthService(db, jwtSecret)
	authHandler := handlers.NewAuthHandler(authService)

	// Auth middleware chains, reused across the group-scoped routes below.
	authRequired := handlers.AuthMiddleware(authService)
	requireGroupMember := handlers.RequireGroupMembership(groupMembershipService)
	requireGroupMemberByPathID := handlers.RequireGroupMembershipByPathParam(groupMembershipService, "id")

	// Setup routes
	// Players — public: no invite/bootstrapping flow yet, see CLAUDE.md.
	r.POST("/players", playerHandler.CreatePlayer)
	r.GET("/players", playerHandler.GetPlayers)
	r.GET("/players/search", playerHandler.SearchPlayer)

	// Groups
	r.POST("/groups", groupHandler.CreateGroup)
	r.GET("/groups", groupHandler.GetGroups)
	r.GET("/groups/:id", groupHandler.GetGroupByID)
	r.GET("/groups/:id/teams", authRequired, requireGroupMemberByPathID, teamHandler.GetTeamsByGroup)
	r.POST("/groups/:id/players", authRequired, requireGroupMemberByPathID, groupHandler.AddPlayerToGroup)
	r.GET("/groups/:id/players", authRequired, requireGroupMemberByPathID, groupHandler.GetGroupMembers)

	// Matches
	r.POST("/matches", authRequired, requireGroupMember, matchHandler.CreateMatch)
	r.GET("/matches/details", authRequired, requireGroupMember, matchHandler.GetMatchesDetails)
	r.GET("/matches/:id/details", authRequired, requireGroupMember, matchHandler.GetMatchDetailsByID)
	r.PUT("/matches/:id", authRequired, requireGroupMember, matchHandler.UpdateMatch)

	// Standings
	r.GET("/standings/points", authRequired, requireGroupMember, standingsHandler.GetPointsStandings)
	r.GET("/standings/scorers", authRequired, requireGroupMember, standingsHandler.GetScorers)
	r.GET("/standings/seasons", authRequired, requireGroupMember, standingsHandler.GetSeasons)

	// Auth
	r.POST("/auth/signup", authHandler.Signup)
	r.POST("/auth/login", authHandler.Login)
	// Add more routes as needed

	// Start server
	r.Run(":8080")
}
