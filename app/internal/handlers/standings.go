package handlers

import (
	"net/http"

	"app/internal/services"

	"github.com/gin-gonic/gin"
)

type StandingsHandler struct {
	Service      *services.StandingsService
	GroupService *services.GroupService
}

func NewStandingsHandler(service *services.StandingsService, groupService *services.GroupService) *StandingsHandler {
	return &StandingsHandler{Service: service, GroupService: groupService}
}

func (h *StandingsHandler) GetPointsStandings(c *gin.Context) {
	groupID, ok := resolveGroupID(c, h.GroupService)
	if !ok {
		return
	}
	rows, err := h.Service.GetPointsStandings(groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *StandingsHandler) GetScorers(c *gin.Context) {
	groupID, ok := resolveGroupID(c, h.GroupService)
	if !ok {
		return
	}
	rows, err := h.Service.GetScorers(groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rows)
}
