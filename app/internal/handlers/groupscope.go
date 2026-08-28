package handlers

import (
	"net/http"

	"app/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// resolveGroupID reads group_id from the query string, or falls back to the
// system's default group when absent (so existing callers, like the current
// frontend, keep working without passing a group_id explicitly). On failure
// it writes the error response itself and returns ok=false.
func resolveGroupID(c *gin.Context, groupService *services.GroupService) (id uuid.UUID, ok bool) {
	if raw := c.Query("group_id"); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group_id"})
			return uuid.Nil, false
		}
		return parsed, true
	}

	group, err := groupService.GetDefaultGroup()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no default group available: " + err.Error()})
		return uuid.Nil, false
	}
	return group.ID, true
}
