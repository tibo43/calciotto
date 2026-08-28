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
	standingsHandler := handlers.NewStandingsHandler(services.NewStandingsService(db, groupMembershipService), groupMembershipService)
	authService := services.NewAuthService(db, jwtSecret)
	authHandler := handlers.NewAuthHandler(authService)

	// Auth middleware chains, reused across the group-scoped routes below.
	authRequired := handlers.AuthMiddleware(authService)
	requireGroupMember := handlers.RequireGroupMembership(groupMembershipService)
	requireGroupMemberByPathID := handlers.RequireGroupMembershipByPathParam(groupMembershipService, "id")

	// Setup routes
	// Players — public: creating a player still needs no token (the group
	// invite flow below covers group membership, not player creation), see
	// CLAUDE.md.
	r.POST("/players", playerHandler.CreatePlayer)
	r.GET("/players", playerHandler.GetPlayers)
	r.GET("/players/search", playerHandler.SearchPlayer)
	// Cross-group profile of the caller themselves: authRequired only, with no
	// requireGroupMember — there is no single group_id to authorize against
	// here, and the handler only ever reports on the groups the JWT's own
	// player belongs to (see StandingsHandler.GetPlayerProfile).
	r.GET("/players/me/stats", authRequired, standingsHandler.GetPlayerProfile)

	// Groups
	// POST /groups and POST /groups/join take authRequired but no
	// requireGroupMember: creating a group means there is no group to be a
	// member of yet, and joining one by invite code is precisely the way in
	// for a non-member. Together they are the group bootstrapping flow — the
	// creator becomes the first member, then shares the invite code.
	r.POST("/groups", authRequired, groupHandler.CreateGroup)
	r.POST("/groups/join", authRequired, groupHandler.JoinGroup)
	r.GET("/groups", groupHandler.GetGroups)
	// "My groups" — authRequired only, same reasoning as /players/me/stats:
	// no single group to authorize against, and the answer is derived from
	// the JWT's own player. Must stay distinct from the public GET /groups,
	// which lists every group in the system.
	r.GET("/groups/me", authRequired, groupHandler.GetMyGroups)
	r.GET("/groups/:id", groupHandler.GetGroupByID)
	// The invite code is a shared secret, so it gets its own member-only
	// route rather than riding along in the (public) group JSON.
	r.GET("/groups/:id/invite-code", authRequired, requireGroupMemberByPathID, groupHandler.GetInviteCode)
	r.GET("/groups/:id/teams", authRequired, requireGroupMemberByPathID, teamHandler.GetTeamsByGroup)
	r.POST("/groups/:id/players", authRequired, requireGroupMemberByPathID, groupHandler.AddPlayerToGroup)
	r.GET("/groups/:id/players", authRequired, requireGroupMemberByPathID, groupHandler.GetGroupMembers)
	// Self-service "leave a group" — the caller can only ever remove their own
	// membership (from the JWT), never someone else's; removing another
	// member is handled by the owner-only route below instead.
	r.DELETE("/groups/:id/members/me", authRequired, requireGroupMemberByPathID, groupHandler.LeaveGroup)
	// Owner-only "remove a member" — unlike the self-service route above, this
	// targets another player's membership, so it's gated by
	// RequireGroupOwnerByPathParam rather than plain group membership.
	r.DELETE("/groups/:id/members/:playerId", authRequired, handlers.RequireGroupOwnerByPathParam(groupMembershipService, "id"), groupHandler.RemoveMember)

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
