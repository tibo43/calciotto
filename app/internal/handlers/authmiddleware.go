package handlers

import (
	"log"
	"net/http"
	"strings"
	"time"

	"app/internal/services"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware validates the Authorization: Bearer <token> header and, on
// success, injects the token's player_id claim into the Gin context under
// the "player_id" key. Wired as authRequired in main.go, ahead of every
// route that needs a caller identity.
func AuthMiddleware(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(header, prefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, messageBody("missing or invalid Authorization header"))
			return
		}

		playerID, err := authService.ParseToken(strings.TrimPrefix(header, prefix))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, messageBody("invalid or expired token"))
			return
		}

		// Last activity is bookkeeping: a failed write must not fail the
		// request it rides on.
		if err := authService.TouchLastSeen(playerID, time.Now()); err != nil {
			log.Printf("[last seen] failed to record activity for player %s: %v", playerID, err)
		}

		c.Set("player_id", playerID)
		c.Next()
	}
}
