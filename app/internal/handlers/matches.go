package handlers

import (
	"net/http"

	"app/internal/models"
	"app/internal/services"

	"github.com/gin-gonic/gin"
)

type MatchHandler struct {
	Service *services.MatchService
}

func NewMatchHandler(service *services.MatchService) *MatchHandler {
	return &MatchHandler{Service: service}
}

func (h *MatchHandler) CreateMatch(c *gin.Context) {
	var match models.Match
	if err := c.ShouldBindJSON(&match); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, err := h.Service.CreateMatch(match.Date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, id)
}

func (h *MatchHandler) GetMatchesDetails(c *gin.Context) {
	matches, err := h.Service.GetMatchesDetails()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, matches)
}

func (h *MatchHandler) GetMatchDetailsByID(c *gin.Context) {
	id := c.Param("id")
	matches, err := h.Service.GetMatchDetailsByID(id)
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
