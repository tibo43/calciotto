package handlers

import (
	"net/http"

	"app/internal/models"
	"app/internal/services"

	"github.com/gin-gonic/gin"
)

type TeamCompositionHandler struct {
	Service *services.TeamCompositionService
}

func NewTeamCompositionHandler(service *services.TeamCompositionService) *TeamCompositionHandler {
	return &TeamCompositionHandler{Service: service}
}

func (h *TeamCompositionHandler) CreateTeamComposition(c *gin.Context) {
	var teamComposition models.TeamComposition
	if err := c.ShouldBindJSON(&teamComposition); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.Service.CreateTeamComposition(&teamComposition); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, teamComposition)
}

func (h *TeamCompositionHandler) GetTeamCompositionAll(c *gin.Context) {
	teamComposition, err := h.Service.GetTeamCompositionAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, teamComposition)
}

func (h *TeamCompositionHandler) GetTeamCompositionByMatchID(c *gin.Context) {
	match_id := c.Param("match_id")
	teamComposition, err := h.Service.GetTeamCompositionByMatchID(match_id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, teamComposition)
}

func (h *TeamCompositionHandler) GetTeamCompositionByTeamID(c *gin.Context) {
	team_id := c.Param("team_id")
	teamComposition, err := h.Service.GetTeamCompositionByTeamID(team_id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, teamComposition)
}
