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

// RequireGroupOwner is RequireGroupMembership's stricter sibling: it must run
// after AuthMiddleware and resolves the target group the same way
// (resolveGroupIDForMembership — query param, then JSON body, then the
// caller's own group), but rejects with 403 unless the authenticated player
// is that group's owner, not merely a member. Use it on routes that carry
// the group_id in the query string or the request body — nothing currently
// does; this exists ahead of the "leave a group" / "remove a member"
// features that will need it.
func RequireGroupOwner(membershipService *services.GroupMembershipService) gin.HandlerFunc {
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

		authorizeGroupOwner(c, membershipService, groupID, playerID)
	}
}

// RequireGroupOwnerByPathParam is RequireGroupOwner for routes where the
// group id is the URL param named paramName (e.g. "id" in
// /groups/:id/players), rather than a query param or body field.
func RequireGroupOwnerByPathParam(membershipService *services.GroupMembershipService, paramName string) gin.HandlerFunc {
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

		authorizeGroupOwner(c, membershipService, groupID, playerID)
	}
}

// authorizeGroupOwner rejects with 403 both when the player isn't a member of
// the group at all (GetRole returns gorm.ErrRecordNotFound) and when the
// player is a member but not the owner — from the caller's point of view
// both are just "not authorized to act as owner here", not a server error.
// Any other error from GetRole is a genuine 500.
func authorizeGroupOwner(c *gin.Context, membershipService *services.GroupMembershipService, groupID, playerID uuid.UUID) {
	role, err := membershipService.GetRole(groupID, playerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "not the owner of this group"})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if role != models.RoleOwner {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "not the owner of this group"})
		return
	}
	c.Next()
}
