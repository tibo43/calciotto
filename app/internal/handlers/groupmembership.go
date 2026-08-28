package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"app/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequireGroupMembership must run after AuthMiddleware (which sets
// "player_id" in the context). It resolves the target group the same way
// resolveGroupID does for read handlers — a group_id query param, else the
// JSON body's group_id/GroupID field, else a group the authenticated player
// actually belongs to — and rejects with 403 unless the authenticated player
// belongs to that group. Use it on routes that carry the group_id in the
// query string or the request body (matches, standings). For routes carrying
// it as a URL param instead (the /groups/:id/* routes), use
// RequireGroupMembershipByPathParam.
func RequireGroupMembership(membershipService *services.GroupMembershipService) gin.HandlerFunc {
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

		authorizeGroupMembership(c, membershipService, groupID, playerID)
	}
}

// RequireGroupMembershipByPathParam is RequireGroupMembership for routes
// where the group id is the URL param named paramName (e.g. "id" in
// /groups/:id/players), rather than a query param or body field.
func RequireGroupMembershipByPathParam(membershipService *services.GroupMembershipService, paramName string) gin.HandlerFunc {
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

		authorizeGroupMembership(c, membershipService, groupID, playerID)
	}
}

func authorizeGroupMembership(c *gin.Context, membershipService *services.GroupMembershipService, groupID, playerID uuid.UUID) {
	isMember, err := membershipService.IsMember(groupID, playerID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !isMember {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "not a member of this group"})
		return
	}
	c.Next()
}

func playerIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	val, exists := c.Get("player_id")
	if !exists {
		return uuid.Nil, false
	}
	id, ok := val.(uuid.UUID)
	return id, ok
}

// resolveGroupIDForMembership mirrors resolveGroupID's query-param-or-own-group
// fallback, but also checks the JSON body in between — needed because
// POST /matches and PUT /matches/:id carry group_id in the body, not the
// query string, under two different JSON keys (models.Match uses "group_id",
// models.MatchWithDetails uses "GroupID"). See resolveGroupID for why this
// deliberately never falls back to GroupService.GetDefaultGroup().
func resolveGroupIDForMembership(c *gin.Context, membershipService *services.GroupMembershipService, playerID uuid.UUID) (uuid.UUID, bool) {
	if raw := c.Query("group_id"); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid group_id"})
			return uuid.Nil, false
		}
		return parsed, true
	}

	if id, ok := peekGroupIDFromBody(c); ok {
		return id, true
	}

	group, err := membershipService.GetFirstGroupForPlayer(playerID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "authenticated player does not belong to any group"})
		return uuid.Nil, false
	}
	return group.ID, true
}

// peekGroupIDFromBody reads the request body to look for a group_id, then
// restores it so the handler's own ShouldBindJSON can still read it.
func peekGroupIDFromBody(c *gin.Context) (uuid.UUID, bool) {
	body, err := c.GetRawData()
	if err != nil {
		return uuid.Nil, false
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		return uuid.Nil, false
	}

	for _, key := range []string{"group_id", "GroupID"} {
		raw, ok := payload[key]
		if !ok {
			continue
		}
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			continue
		}
		id, err := uuid.Parse(s)
		if err != nil {
			continue
		}
		return id, true
	}
	return uuid.Nil, false
}
