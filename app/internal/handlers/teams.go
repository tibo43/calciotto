package handlers

import (
	"net/http"

	"app/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type TeamHandler struct {
	Service *services.TeamService
}

func NewTeamHandler(service *services.TeamService) *TeamHandler {
	return &TeamHandler{Service: service}
}

func (h *TeamHandler) GetTeamsByGroup(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
		return
	}

	teams, err := h.Service.GetTeamsByGroupID(groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, teams)
}
