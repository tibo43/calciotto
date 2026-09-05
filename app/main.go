package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"app/internal/handlers"
	"app/internal/services"

	"app/pkg/database"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
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
		AllowOrigins:     allowedOrigins(),
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
	playerHandler := handlers.NewPlayerHandler(services.NewPlayerService(db))
	groupHandler := handlers.NewGroupHandler(groupService, groupMembershipService)
	teamHandler := handlers.NewTeamHandler(services.NewTeamService(db))
	matchService := services.NewMatchService(db)
	matchHandler := handlers.NewMatchHandler(matchService, groupMembershipService)
	matchRegistrationHandler := handlers.NewMatchRegistrationHandler(services.NewMatchRegistrationService(db))
	matchVoteHandler := handlers.NewMatchVoteHandler(services.NewMatchVoteService(db))
	standingsHandler := handlers.NewStandingsHandler(services.NewStandingsService(db, groupMembershipService), groupMembershipService)
	authHandler := handlers.NewAuthHandler(authService)

	// Auth middleware chains, reused across the group-scoped routes below.
	authRequired := handlers.AuthMiddleware(authService)
	requireGroupMember := handlers.RequireGroupMembership(groupMembershipService)
	requireGroupMemberByPathID := handlers.RequireGroupMembershipByPathParam(groupMembershipService, "id")
	requireGroupAdmin := handlers.RequireGroupAdmin(groupMembershipService)
	requireGroupAdminByPathID := handlers.RequireGroupAdminByPathParam(groupMembershipService, "id")
	// The /matches/:id/registrations routes name a *match*, not a group, so
	// they need the pair that derives the group from the match rather than
	// trusting a group id the caller supplies (see matchscope.go).
	requireGroupMemberByMatchID := handlers.RequireGroupMembershipByMatchPathParam(matchService, groupMembershipService, "id")
	requireGroupAdminByMatchID := handlers.RequireGroupAdminByMatchPathParam(matchService, groupMembershipService, "id")

	// Setup routes
	// Players — there is no admin-only "create a player" route any more:
	// ghost (credential-less) players were removed, so every player row now
	// comes from POST /auth/signup instead. See CLAUDE.md.
	// Cross-group profile of the caller themselves: authRequired only, with no
	// requireGroupMember — there is no single group_id to authorize against
	// here, and the handler only ever reports on the groups the JWT's own
	// player belongs to (see StandingsHandler.GetPlayerProfile).
	r.GET("/players/me/stats", authRequired, standingsHandler.GetPlayerProfile)
	// Same reasoning as /players/me/stats: this only ever acts on the JWT's
	// own player id, so no group-scoping middleware is needed.
	r.PATCH("/players/me", authRequired, playerHandler.UpdateMyName)

	// Groups
	// POST /groups and POST /groups/join are temporarily disabled (both
	// handlers always answer 403 — see GroupHandler.CreateGroup/JoinGroup): a
	// player can no longer self-service creating or joining a group.
	// authRequired stays so an unauthenticated caller still gets 401 before
	// reaching that 403. The only way into a group now is an admin sharing the
	// invite-code link (GroupSettingsModal.vue) that pre-fills the mandatory
	// invite code on POST /auth/signup — see CLAUDE.md.
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
	// Admin-only "rename/recolour a team" — same admin gate as the member-role
	// and invite routes above; TeamService.UpdateTeam additionally scopes the
	// lookup to :id so :teamId can't reach into another group.
	r.PATCH("/groups/:id/teams/:teamId", authRequired, requireGroupAdminByPathID, teamHandler.UpdateTeam)
	r.POST("/groups/:id/players", authRequired, requireGroupMemberByPathID, groupHandler.AddPlayerToGroup)
	r.GET("/groups/:id/players", authRequired, requireGroupMemberByPathID, groupHandler.GetGroupMembers)
	// Self-service "leave a group" — the caller can only ever remove their own
	// membership (from the JWT), never someone else's; removing another
	// member is handled by the admin-only route below instead.
	r.DELETE("/groups/:id/members/me", authRequired, requireGroupMemberByPathID, groupHandler.LeaveGroup)
	r.PATCH("/groups/:id/favorite", authRequired, requireGroupMemberByPathID, groupHandler.SetFavoriteGroup)
	// Admin-only "remove a member" — unlike the self-service route above, this
	// targets another player's membership, so it's gated by
	// RequireGroupAdminByPathParam rather than plain group membership.
	r.DELETE("/groups/:id/members/:playerId", authRequired, requireGroupAdminByPathID, groupHandler.RemoveMember)
	// Admin-only "change a member's role" — the only way a group gains an
	// admin besides its creator, so it has to be reachable by any existing
	// admin, and by no one else.
	r.PATCH("/groups/:id/members/:playerId/role", authRequired, requireGroupAdminByPathID, groupHandler.UpdateMemberRole)

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
	// Deleting a match is admin-only too — group_id travels in the query
	// string (a DELETE has no body), which requireGroupAdmin's
	// resolveGroupIDForMembership already handles the same way resolveGroupID
	// does in the handler itself.
	r.DELETE("/matches/:id", authRequired, requireGroupAdmin, matchHandler.DeleteMatch)

	// Sign-ups for a scheduled match. All five are gated by the match-scoped
	// middlewares: the path carries a match id, so the group is derived from
	// the match itself — a member of another group gets 404, not 403, so match
	// ids stay unenumerable (see matchscope.go).
	//
	// Signing up, withdrawing and reading the list are open to any member: the
	// player always comes from the JWT, so a member can only ever add or remove
	// *themselves*. The DELETE has no /me suffix for that reason — there is no
	// other sign-up it could target.
	r.POST("/matches/:id/registrations", authRequired, requireGroupMemberByMatchID, matchRegistrationHandler.Register)
	r.DELETE("/matches/:id/registrations", authRequired, requireGroupMemberByMatchID, matchRegistrationHandler.Unregister)
	r.GET("/matches/:id/registrations", authRequired, requireGroupMemberByMatchID, matchRegistrationHandler.ListRegistrations)
	// Freezing the roster (in order to compose the teams) and undoing a
	// mis-clicked close are admin actions, like every other write on a match.
	r.POST("/matches/:id/registrations/close", authRequired, requireGroupAdminByMatchID, matchRegistrationHandler.CloseRegistrations)
	r.POST("/matches/:id/registrations/reopen", authRequired, requireGroupAdminByMatchID, matchRegistrationHandler.ReopenRegistrations)

	// Man of the Match voting. Same match-scoped middleware pair as the
	// sign-up routes above, but every route is open to any member: voting has
	// no admin-only action at all, since eligibility to vote is deliberately
	// broader than "played in the match" and there is no close/reopen concept
	// for it (see MatchVoteService).
	r.POST("/matches/:id/votes", authRequired, requireGroupMemberByMatchID, matchVoteHandler.Vote)
	r.DELETE("/matches/:id/votes", authRequired, requireGroupMemberByMatchID, matchVoteHandler.Unvote)
	r.GET("/matches/:id/votes", authRequired, requireGroupMemberByMatchID, matchVoteHandler.ListVotes)

	// Standings
	r.GET("/standings/points", authRequired, requireGroupMember, standingsHandler.GetPointsStandings)
	r.GET("/standings/scorers", authRequired, requireGroupMember, standingsHandler.GetScorers)
	r.GET("/standings/motm", authRequired, requireGroupMember, standingsHandler.GetMotmStandings)
	r.GET("/standings/seasons", authRequired, requireGroupMember, standingsHandler.GetSeasons)

	// Auth — all four are unauthenticated by necessity (a caller signing up or
	// resetting a lost password has no token yet), which also makes them the
	// only routes in this app an anonymous script can hit repeatedly for free.
	// Each gets its own per-IP rate limiter (see ratelimit.go): bcrypt already
	// slows down one guess against Login, but not a script making thousands: a
	// generous burst absorbs a legitimate user mistyping their password a few
	// times in a row, then throttles hard.
	loginRateLimit := handlers.RateLimit(rate.Every(6*time.Second), 10)          // ~10/min, burst 10
	signupRateLimit := handlers.RateLimit(rate.Every(12*time.Minute), 5)         // ~5/hour, burst 5
	forgotPasswordRateLimit := handlers.RateLimit(rate.Every(12*time.Minute), 5) // ~5/hour, burst 5 — also caps Brevo email spend, see CLAUDE.md
	resetPasswordRateLimit := handlers.RateLimit(rate.Every(3*time.Minute), 20)  // ~20/hour, burst 20 — tokens are unguessable 32-byte values, this is a courtesy cap, not the real defense

	r.POST("/auth/signup", signupRateLimit, authHandler.Signup)
	r.POST("/auth/login", loginRateLimit, authHandler.Login)
	// Public like signup/login above: someone who forgot their password has by
	// definition no token to authenticate the request with.
	r.POST("/auth/forgot-password", forgotPasswordRateLimit, authHandler.ForgotPassword)
	r.POST("/auth/reset-password", resetPasswordRateLimit, authHandler.ResetPassword)
	// Add more routes as needed

	// Liveness probe for the platform's healthcheck. It deliberately does NOT
	// touch the database: the host restarts an instance whose healthcheck
	// fails, so reporting unhealthy on a transient database hiccup (Neon
	// waking from scale-to-zero, a pooler blip) would turn a few seconds of
	// slow queries into a restart loop. InitDB already fails fast at startup
	// if the database is unreachable, which is the check that matters.
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Start server. The host assigns the port through PORT (Koyeb does), so it
	// can't be hardcoded; 8080 keeps docker-compose and `go run .` unchanged.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}

// allowedOrigins lists the browser origins allowed to call this API
// cross-origin. In production the frontend is served from its own domain
// (Vercel), so the list has to be configurable rather than hardcoded:
// CORS_ALLOWED_ORIGINS holds it as a comma-separated list. An unset — or
// entirely blank — value falls back to the local dev origins, so
// `docker compose up` keeps working with no extra configuration. Note the
// fallback rather than an empty slice: gin-contrib/cors with no allowed
// origin and AllowAllOrigins false rejects every cross-origin request, which
// would be a silent, hard-to-diagnose outage rather than a startup failure.
func allowedOrigins() []string {
	raw := os.Getenv("CORS_ALLOWED_ORIGINS")

	origins := make([]string, 0, 2)
	for _, origin := range strings.Split(raw, ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			origins = append(origins, origin)
		}
	}

	if len(origins) == 0 {
		return []string{"http://localhost:4000", "http://127.0.0.1:4000"}
	}
	return origins
}
