package handlers

import (
	"errors"
	"net/http"
	"strings"

	"app/internal/services"

	"github.com/gin-gonic/gin"
)

type PlayerHandler struct {
	Service *services.PlayerService
}

func NewPlayerHandler(service *services.PlayerService) *PlayerHandler {
	return &PlayerHandler{Service: service}
}

// UpdateMyName lets the authenticated player rename themselves. Global
// uniqueness (unlike PlayerService.CreatePlayer, which allows name
// collisions on purpose — see CLAUDE.md) is enforced here specifically,
// per an explicit product decision: creation stays permissive, but a
// player actively renaming themselves into a name someone else already
// holds is rejected.
func (h *PlayerHandler) UpdateMyName(c *gin.Context) {
	playerID, ok := playerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	name := strings.TrimSpace(req.Name)
	if err := h.Service.UpdateName(playerID, name); err != nil {
		switch {
		case errors.Is(err, services.ErrEmptyPlayerName), errors.Is(err, services.ErrPlayerNameAlreadyUsed):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"name": name})
}

// GetMyFeatures returns the caller's feature list and guide status.
func (h *PlayerHandler) GetMyFeatures(c *gin.Context) {
	playerID, ok := playerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
		return
	}
	features, err := h.Service.GetFeatures(playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	completed, err := h.Service.GuideCompleted(playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load guide status"})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"features": features, "guide_completed": completed})
}

func (h *PlayerHandler) CompleteGuide(c *gin.Context) {
	playerID, ok := playerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
		return
	}
	if err := h.Service.CompleteGuide(playerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save guide completion"})
		return
	}
	c.Status(http.StatusNoContent)
}
