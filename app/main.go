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
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
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
	authService := services.NewAuthService(db, jwtSecret)
	playerHandler := handlers.NewPlayerHandler(services.NewPlayerService(db), groupService, groupMembershipService)
	groupHandler := handlers.NewGroupHandler(groupService, groupMembershipService, authService)
	teamHandler := handlers.NewTeamHandler(services.NewTeamService(db))
	matchHandler := handlers.NewMatchHandler(services.NewMatchService(db), groupMembershipService)
	standingsHandler := handlers.NewStandingsHandler(services.NewStandingsService(db, groupMembershipService), groupMembershipService)
	authHandler := handlers.NewAuthHandler(authService)

	// Auth middleware chains, reused across the group-scoped routes below.
	authRequired := handlers.AuthMiddleware(authService)
	requireGroupMember := handlers.RequireGroupMembership(groupMembershipService)
	requireGroupMemberByPathID := handlers.RequireGroupMembershipByPathParam(groupMembershipService, "id")
	requireGroupAdmin := handlers.RequireGroupAdmin(groupMembershipService)
	requireGroupAdminByPathID := handlers.RequireGroupAdminByPathParam(groupMembershipService, "id")

	// Setup routes
	// Players — creating a player ("ghost" roster entry, e.g. from
	// MatchDetails.vue's create-on-the-fly flow) is admin-only, gated to the
	// admin of the target group the same way POST /matches is: group_id
	// travels in the JSON body, so this reuses the body/query-resolving
	// requireGroupAdmin rather than a path-param variant. See CLAUDE.md.
	r.POST("/players", authRequired, requireGroupAdmin, playerHandler.CreatePlayer)
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
	// member is handled by the admin-only route below instead.
	r.DELETE("/groups/:id/members/me", authRequired, requireGroupMemberByPathID, groupHandler.LeaveGroup)
	// Admin-only "remove a member" — unlike the self-service route above, this
	// targets another player's membership, so it's gated by
	// RequireGroupAdminByPathParam rather than plain group membership.
	r.DELETE("/groups/:id/members/:playerId", authRequired, requireGroupAdminByPathID, groupHandler.RemoveMember)
	// Admin-only "change a member's role" — the only way a group gains an
	// admin besides its creator, so it has to be reachable by any existing
	// admin, and by no one else.
	r.PATCH("/groups/:id/members/:playerId/role", authRequired, requireGroupAdminByPathID, groupHandler.UpdateMemberRole)
	// Admin-only "invite a ghost member": attaches an email to a member who has
	// none (a roster entry created via POST /players) and sends them a link to
	// set their own password. Same admin gate as the two routes above; the
	// handler additionally checks the target is a member of *this* group.
	r.POST("/groups/:id/members/:playerId/invite", authRequired, requireGroupAdminByPathID, groupHandler.InvitePlayer)

	// Matches
	// Creating a match and editing its scores are admin-only
	// (requireGroupAdmin); reading them stays open to any member
	// (requireGroupMember). Both write routes carry the group_id in the body,
	// not the path, hence the body/query-resolving middleware rather than the
	// ByPathID one.
	r.POST("/matches", authRequired, requireGroupAdmin, matchHandler.CreateMatch)
	r.GET("/matches/details", authRequired, requireGroupMember, matchHandler.GetMatchesDetails)
	r.GET("/matches/:id/details", authRequired, requireGroupMember, matchHandler.GetMatchDetailsByID)
	r.PUT("/matches/:id", authRequired, requireGroupAdmin, matchHandler.UpdateMatch)

	// Standings
	r.GET("/standings/points", authRequired, requireGroupMember, standingsHandler.GetPointsStandings)
	r.GET("/standings/scorers", authRequired, requireGroupMember, standingsHandler.GetScorers)
	r.GET("/standings/seasons", authRequired, requireGroupMember, standingsHandler.GetSeasons)

	// Auth
	r.POST("/auth/signup", authHandler.Signup)
	r.POST("/auth/login", authHandler.Login)
	// Public like signup/login above: someone who forgot their password has by
	// definition no token to authenticate the request with.
	r.POST("/auth/forgot-password", authHandler.ForgotPassword)
	r.POST("/auth/reset-password", authHandler.ResetPassword)
	// Add more routes as needed

	// Start server
	r.Run(":8080")
}
