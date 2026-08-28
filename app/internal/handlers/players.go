package handlers

import (
	"errors"
	"net/http"

	"app/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PlayerHandler struct {
	Service           *services.PlayerService
	GroupService      *services.GroupService
	MembershipService *services.GroupMembershipService
}

func NewPlayerHandler(service *services.PlayerService, groupService *services.GroupService, membershipService *services.GroupMembershipService) *PlayerHandler {
	return &PlayerHandler{Service: service, GroupService: groupService, MembershipService: membershipService}
}

func (h *PlayerHandler) GetPlayers(c *gin.Context) {
	players, err := h.Service.GetPlayers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, players)
}

// CreatePlayer creates a player, then attaches it to a group — group_id in
// the body if provided, else the default group (same fallback CreateMatch
// uses, see resolveGroupID). PlayerService itself stays group-agnostic; the
// membership is created here in the handler.
func (h *PlayerHandler) CreatePlayer(c *gin.Context) {
	var req struct {
		Name    string    `json:"name"`
		GroupID uuid.UUID `json:"group_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id, err := h.Service.CreatePlayer(req.Name)
	if err != nil {
		if errors.Is(err, services.ErrEmptyPlayerName) || errors.Is(err, services.ErrPlayerAlreadyExists) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	groupID := req.GroupID
	if groupID == uuid.Nil {
		group, err := h.GroupService.GetDefaultGroup()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no default group available: " + err.Error()})
			return
		}
		groupID = group.ID
	}

	if err := h.MembershipService.AddPlayerToGroup(groupID, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, id)
}

func (h *PlayerHandler) SearchPlayer(c *gin.Context) {
	name := c.Query("name")
	player, err := h.Service.SearchPlayer(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Player not found"})
		return
	}
	c.JSON(http.StatusOK, player)
}
