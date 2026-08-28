package handlers

import (
	"net/http"

	"app/internal/services"

	"github.com/gin-gonic/gin"
)

type StandingsHandler struct {
	Service           *services.StandingsService
	MembershipService *services.GroupMembershipService
}

func NewStandingsHandler(service *services.StandingsService, membershipService *services.GroupMembershipService) *StandingsHandler {
	return &StandingsHandler{Service: service, MembershipService: membershipService}
}

func (h *StandingsHandler) GetPointsStandings(c *gin.Context) {
	groupID, ok := resolveGroupID(c, h.MembershipService)
	if !ok {
		return
	}
	rows, err := h.Service.GetPointsStandings(groupID, c.Query("season"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *StandingsHandler) GetScorers(c *gin.Context) {
	groupID, ok := resolveGroupID(c, h.MembershipService)
	if !ok {
		return
	}
	rows, err := h.Service.GetScorers(groupID, c.Query("season"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rows)
}

// GetSeasons returns the season labels the resolved group has matches in, so
// the frontend can populate its season selector without knowing the
// September-to-September rule itself.
func (h *StandingsHandler) GetSeasons(c *gin.Context) {
	groupID, ok := resolveGroupID(c, h.MembershipService)
	if !ok {
		return
	}
	seasons, err := h.Service.GetSeasons(groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, seasons)
}
