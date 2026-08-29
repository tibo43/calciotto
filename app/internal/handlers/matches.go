package handlers

import (
	"errors"
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
// — rather than the old sort-by-random-UUID default (GroupService.GetDefaultGroup,
// since removed), which had no relation to the caller and would let the match
// get created in a group the caller was never even checked against.
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
	// A group with no matches yet is a normal state (e.g. a brand-new group),
	// not an error — but GORM leaves the slice nil, which would serialize as
	// `null` and force every caller to special-case it (see GroupHandler.GetMyGroups
	// for the same fix applied to GET /groups/me).
	if matches == nil {
		matches = []models.MatchWithDetails{}
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
		switch {
		case errors.Is(err, services.ErrMatchNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, matches)
}

// DeleteMatch is gated by requireGroupAdmin in main.go, which resolves its
// group the same way this handler does (query param group_id, else the
// caller's first group) — reusing that exact resolution, rather than a
// different one, is what stops an admin of group A from deleting a match in
// group B: the group the middleware authorized against must be the same one
// the delete is scoped to.
func (h *MatchHandler) DeleteMatch(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid match id"})
		return
	}
	groupID, ok := resolveGroupID(c, h.MembershipService)
	if !ok {
		return
	}

	if err := h.Service.DeleteMatch(id, groupID); err != nil {
		switch {
		case errors.Is(err, services.ErrMatchNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": true})
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
