package handlers

import (
	"errors"
	"net/http"

	"app/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
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

// UpdateTeam renames a team and/or changes its colour. The route
// (PATCH /groups/:id/teams/:teamId) sits behind RequireGroupAdminByPathParam,
// so the caller is already confirmed to be an admin of :id by the time this
// runs — but that says nothing about whether :teamId actually belongs to
// :id, which is why TeamService.UpdateTeam re-checks group_id itself and
// this handler maps a resulting gorm.ErrRecordNotFound to 404 rather than
// treating admin-of-:id as license to touch any team by guessable UUID.
//
// The body is a full replacement (name + colour), not a partial patch, the
// same convention PATCH /groups/:id/members/:playerId/role uses for its
// role field.
func (h *TeamHandler) UpdateTeam(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
		return
	}

	teamID, err := uuid.Parse(c.Param("teamId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team id"})
		return
	}

	var body struct {
		Name   string `json:"name"`
		Colour string `json:"colour"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	team, err := h.Service.UpdateTeam(teamID, groupID, body.Name, body.Colour)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrTeamNameRequired), errors.Is(err, services.ErrTeamColourRequired):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, gorm.ErrRecordNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "team not found in this group"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, team)
}
