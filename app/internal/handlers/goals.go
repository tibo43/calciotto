package handlers

import (
	"log"
	"net/http"

	"app/internal/models"
	"app/internal/services"

	"github.com/gin-gonic/gin"
)

type GoalHandler struct {
	Service *services.GoalService
}

func NewGoalHandler(service *services.GoalService) *GoalHandler {
	return &GoalHandler{Service: service}
}

func (h *GoalHandler) CreateGoal(c *gin.Context) {
	var goal models.Goal
	if err := c.ShouldBindJSON(&goal); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.Service.CreateGoal(&goal); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	log.Println(goal)
	c.JSON(http.StatusOK, goal)
}

func (h *GoalHandler) GetGoalByMatchID(c *gin.Context) {
	match_id := c.Param("match_id")
	goal, err := h.Service.GetGoalByMatchID(match_id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, goal)
}

func (h *GoalHandler) GetGoalByPlayerID(c *gin.Context) {
	player_id := c.Param("player_id")
	goal, err := h.Service.GetGoalByPlayerID(player_id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, goal)
}
