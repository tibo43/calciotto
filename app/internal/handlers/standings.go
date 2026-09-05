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

// GetMotmStandings returns the resolved group's Man of the Match leaderboard,
// following the same shape as GetPointsStandings/GetScorers above.
func (h *StandingsHandler) GetMotmStandings(c *gin.Context) {
	groupID, ok := resolveGroupID(c, h.MembershipService)
	if !ok {
		return
	}
	rows, err := h.Service.GetMotmStandings(groupID, c.Query("season"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rows)
}

// GetPlayerProfile returns the authenticated caller's own stats across every
// group they belong to. The player is read from the JWT, never from the URL:
// which groups a player belongs to is itself privileged information, so a
// /players/:id/stats variant would leak another player's group list —
// including groups the caller isn't a member of — straight past
// RequireGroupMembership. Consulting someone else's profile needs a
// visibility model of its own and doesn't exist yet.
//
// This lives on StandingsHandler rather than PlayerHandler because it is
// backed by StandingsService, keeping the one-handler-per-service split.
func (h *StandingsHandler) GetPlayerProfile(c *gin.Context) {
	playerID, ok := playerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
		return
	}

	profile, err := h.Service.GetPlayerProfile(playerID, c.Query("season"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, profile)
}
