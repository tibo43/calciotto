package handlers

import (
	"net/http"

	"app/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// resolveGroupID reads group_id from the query string, or falls back to a
// group the authenticated caller (AuthMiddleware sets "player_id" in the
// context) actually belongs to — so existing callers, like the current
// frontend, keep working without passing a group_id explicitly. This must
// only be used behind AuthMiddleware; it deliberately does NOT fall back to
// the old sort-by-random-UUID default (GroupService.GetDefaultGroup, since
// removed), which had no relation to who the caller is — that mismatch let
// anyone flip every existing user's default group (and thus their access)
// just by creating an unrelated second group. On failure it
// writes the error response itself and returns ok=false.
func resolveGroupID(c *gin.Context, membershipService *services.GroupMembershipService) (id uuid.UUID, ok bool) {
	if raw := c.Query("group_id"); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group_id"})
			return uuid.Nil, false
		}
		return parsed, true
	}

	playerID, ok := playerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
		return uuid.Nil, false
	}

	group, err := membershipService.GetFirstGroupForPlayer(playerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "authenticated player does not belong to any group"})
		return uuid.Nil, false
	}
	return group.ID, true
}
