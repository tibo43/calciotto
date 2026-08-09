package handlers

import (
	"net/http"

	"app/internal/models"
	"app/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MatchPlayerHandler struct {
	Service *services.MatchPlayerService
}

func NewMatchPlayerHandler(service *services.MatchPlayerService) *MatchPlayerHandler {
	return &MatchPlayerHandler{Service: service}
}

func (h *MatchPlayerHandler) CreateMatchPlayer(c *gin.Context) {
	var teamComposition models.MatchPlayer
	if err := c.ShouldBindJSON(&teamComposition); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.Service.CreateMatchPlayer(&teamComposition); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, teamComposition)
}

func (h *MatchPlayerHandler) GetMatchPlayerAll(c *gin.Context) {
	teamComposition, err := h.Service.GetMatchPlayerAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, teamComposition)
}

func (h *MatchPlayerHandler) GetMatchPlayerByMatchID(c *gin.Context) {
	match_id, err := uuid.Parse(c.Param("match_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid match id"})
		return
	}
	teamComposition, err := h.Service.GetMatchPlayerByMatchID(match_id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, teamComposition)
}

func (h *MatchPlayerHandler) GetMatchPlayerByTeamID(c *gin.Context) {
	team_id, err := uuid.Parse(c.Param("team_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team id"})
		return
	}
	teamComposition, err := h.Service.GetMatchPlayerByTeamID(team_id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, teamComposition)
}
