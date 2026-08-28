package handlers

import (
	"net/http"

	"app/internal/models"
	"app/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MatchHandler struct {
	Service      *services.MatchService
	GroupService *services.GroupService
}

func NewMatchHandler(service *services.MatchService, groupService *services.GroupService) *MatchHandler {
	return &MatchHandler{Service: service, GroupService: groupService}
}

func (h *MatchHandler) CreateMatch(c *gin.Context) {
	var match models.Match
	if err := c.ShouldBindJSON(&match); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	groupID := match.GroupID
	if groupID == uuid.Nil {
		group, err := h.GroupService.GetDefaultGroup()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no default group available: " + err.Error()})
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
	groupID, ok := resolveGroupID(c, h.GroupService)
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
	groupID, ok := resolveGroupID(c, h.GroupService)
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
