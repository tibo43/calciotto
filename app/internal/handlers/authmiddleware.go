package handlers

import (
	"net/http"
	"strings"

	"app/internal/services"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware validates the Authorization: Bearer <token> header and, on
// success, injects the token's player_id claim into the Gin context under
// the "player_id" key. Not wired to any route yet — see CLAUDE.md.
func AuthMiddleware(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(header, prefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid Authorization header"})
			return
		}

		playerID, err := authService.ParseToken(strings.TrimPrefix(header, prefix))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Set("player_id", playerID)
		c.Next()
	}
}
