package handlers

import (
	"net/http"

	"app/internal/services"

	"github.com/gin-gonic/gin"
)

type StandingsHandler struct {
	Service *services.StandingsService
}

func NewStandingsHandler(service *services.StandingsService) *StandingsHandler {
	return &StandingsHandler{Service: service}
}

func (h *StandingsHandler) GetPointsStandings(c *gin.Context) {
	rows, err := h.Service.GetPointsStandings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *StandingsHandler) GetScorers(c *gin.Context) {
	rows, err := h.Service.GetScorers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rows)
}
