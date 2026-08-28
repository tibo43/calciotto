package handlers

import (
	"net/http"

	"app/internal/models"
	"app/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MatchHandler struct {
	Service           *services.MatchService
	MembershipService *services.GroupMembershipService
}

func NewMatchHandler(service *services.MatchService, membershipService *services.GroupMembershipService) *MatchHandler {
	return &MatchHandler{Service: service, MembershipService: membershipService}
}

// CreateMatch requires authentication (see main.go), so when the payload
// carries no group_id it falls back to a group the caller actually belongs
// to — the same resolution RequireGroupMembership already authorized against
// — rather than GroupService.GetDefaultGroup(), which has no relation to the
// caller and would let the match get created in a group the caller was never
// even checked against.
func (h *MatchHandler) CreateMatch(c *gin.Context) {
	var match models.Match
	if err := c.ShouldBindJSON(&match); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	groupID := match.GroupID
	if groupID == uuid.Nil {
		playerID, ok := playerIDFromContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
			return
		}
		group, err := h.MembershipService.GetFirstGroupForPlayer(playerID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "authenticated player does not belong to any group"})
			return
		}
		groupID = group.ID
	}

	id, err := h.Service.CreateMatch(match.Date, groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, id)
}

func (h *MatchHandler) GetMatchesDetails(c *gin.Context) {
	groupID, ok := resolveGroupID(c, h.MembershipService)
	if !ok {
		return
	}
	matches, err := h.Service.GetMatchesDetails(groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, matches)
}

func (h *MatchHandler) GetMatchDetailsByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid match id"})
		return
	}
	groupID, ok := resolveGroupID(c, h.MembershipService)
	if !ok {
		return
	}
	matches, err := h.Service.GetMatchDetailsByID(id, groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, matches)
}

func (h *MatchHandler) UpdateMatch(c *gin.Context) {
	var match models.MatchWithDetails
	if err := c.ShouldBindJSON(&match); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.Service.UpdateMatch(match); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, match)
}
