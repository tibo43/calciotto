package handlers

import (
	"errors"
	"net/http"

	"app/internal/models"
	"app/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RequireGroupAdmin is RequireGroupMembership's stricter sibling: it must run
// after AuthMiddleware and resolves the target group the same way
// (resolveGroupIDForMembership — query param, then JSON body, then the
// caller's own group), but rejects with 403 unless the authenticated player
// is one of that group's admins, not merely a member. Use it on routes that
// carry the group_id in the query string or the request body: POST /matches
// and PUT /matches/:id are gated with it, so only an admin can create a match
// or edit its scores. Reading matches and standings deliberately stays open
// to any member and keeps RequireGroupMembership.
func RequireGroupAdmin(membershipService *services.GroupMembershipService) gin.HandlerFunc {
	return func(c *gin.Context) {
		playerID, ok := playerIDFromContext(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
			return
		}

		groupID, ok := resolveGroupIDForMembership(c, membershipService, playerID)
		if !ok {
			return
		}

		authorizeGroupAdmin(c, membershipService, groupID, playerID)
	}
}

// RequireGroupAdminByPathParam is RequireGroupAdmin for routes where the
// group id is the URL param named paramName (e.g. "id" in
// /groups/:id/members/:playerId), rather than a query param or body field.
func RequireGroupAdminByPathParam(membershipService *services.GroupMembershipService, paramName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		playerID, ok := playerIDFromContext(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
			return
		}

		groupID, err := uuid.Parse(c.Param(paramName))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
			return
		}

		authorizeGroupAdmin(c, membershipService, groupID, playerID)
	}
}

// authorizeGroupAdmin rejects with 403 both when the player isn't a member of
// the group at all (GetRole returns gorm.ErrRecordNotFound) and when the
// player is a member but not an admin — from the caller's point of view both
// are just "not authorized to act as an admin here", not a server error.
// Any other error from GetRole is a genuine 500.
func authorizeGroupAdmin(c *gin.Context, membershipService *services.GroupMembershipService, groupID, playerID uuid.UUID) {
	role, err := membershipService.GetRole(groupID, playerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "not an admin of this group"})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if role != models.RoleAdmin {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "not an admin of this group"})
		return
	}
	c.Next()
}
