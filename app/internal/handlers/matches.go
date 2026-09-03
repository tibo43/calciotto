package handlers

import (
	"errors"
	"net/http"
	"time"

	"app/internal/models"
	"app/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MatchHandler struct {
	Service           *services.MatchService
	MembershipService *services.GroupMembershipService
}

func NewMatchHandler(service *services.MatchService, membershipService *services.GroupMembershipService) *MatchHandler {
	return &MatchHandler{Service: service, MembershipService: membershipService}
}

// createMatchRequest is the wire shape of POST /matches, deliberately a
// dedicated struct (as the auth handlers already do) rather than models.Match.
//
// Binding straight into the model would now let a client set
// registrations_closed_at while creating the match — sign-ups closed before
// anyone could make one — which is not an admin action that exists. A request
// struct that simply has no such field makes that unrepresentable rather than
// something a later filter has to remember to strip.
//
// The three scheduling fields are pointers so that "absent" and "zero" stay
// distinguishable: absent means an ordinary unscheduled match, which is what
// every existing caller sends. MatchSpec validates them as all-or-nothing.
//
// The timestamps are plain *time.Time, so encoding/json parses them as RFC3339
// and keeps the offset the client sent. That is load-bearing and must not be
// "normalized" to UTC here: MatchService.CreateMatch derives the match's
// calendar Date from ScheduledAt *in the timestamp's own location*, so a 21:00
// Paris kick-off keeps the day the client meant — converting first would silently
// move some matches to the previous day.
type createMatchRequest struct {
	Date                models.Date `json:"date"`
	GroupID             uuid.UUID   `json:"group_id"`
	ScheduledAt         *time.Time  `json:"scheduled_at"`
	RegistrationOpensAt *time.Time  `json:"registration_opens_at"`
	MaxPlayers          *int        `json:"max_players"`
}

// CreateMatch requires authentication (see main.go), so when the payload
// carries no group_id it falls back to a group the caller actually belongs
// to — the same resolution RequireGroupMembership already authorized against
// — rather than the old sort-by-random-UUID default (GroupService.GetDefaultGroup,
// since removed), which had no relation to the caller and would let the match
// get created in a group the caller was never even checked against.
func (h *MatchHandler) CreateMatch(c *gin.Context) {
	var req createMatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	groupID := req.GroupID
	if groupID == uuid.Nil {
		playerID, ok := playerIDFromContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
			return
		}
		group, err := h.MembershipService.GetFirstGroupForPlayer(playerID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "authenticated player does not belong to any group"})
			return
		}
		groupID = group.ID
	}

	id, err := h.Service.CreateMatch(services.MatchSpec{
		Date:                req.Date,
		ScheduledAt:         req.ScheduledAt,
		RegistrationOpensAt: req.RegistrationOpensAt,
		MaxPlayers:          req.MaxPlayers,
	}, groupID)
	if err != nil {
		switch {
		// An incoherent schedule is the client's mistake, not the server's:
		// half a schedule, a window opening after kick-off, or a roster size
		// that would bench everyone. Mapped to 400 the way every other handler
		// maps its service's validation sentinels.
		case errors.Is(err, services.ErrIncompleteSchedule),
			errors.Is(err, services.ErrRegistrationOpensAfterKickoff),
			errors.Is(err, services.ErrInvalidMaxPlayers):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, id)
}

func (h *MatchHandler) GetMatchesDetails(c *gin.Context) {
	groupID, ok := resolveGroupID(c, h.MembershipService)
	if !ok {
		return
	}
	// season is optional and read the same way StandingsHandler reads it: an
	// absent/empty value means "every season", the behaviour before the
	// matches list gained a season filter.
	matches, err := h.Service.GetMatchesDetails(groupID, c.Query("season"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// A group with no matches yet is a normal state (e.g. a brand-new group),
	// not an error — but GORM leaves the slice nil, which would serialize as
	// `null` and force every caller to special-case it (see GroupHandler.GetMyGroups
	// for the same fix applied to GET /groups/me).
	if matches == nil {
		matches = []models.MatchWithDetails{}
	}
	c.JSON(http.StatusOK, matches)
}

func (h *MatchHandler) GetMatchDetailsByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid match id"})
		return
	}
	groupID, ok := resolveGroupID(c, h.MembershipService)
	if !ok {
		return
	}
	matches, err := h.Service.GetMatchDetailsByID(id, groupID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrMatchNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, matches)
}

// DeleteMatch is gated by requireGroupAdmin in main.go, which resolves its
// group the same way this handler does (query param group_id, else the
// caller's first group) — reusing that exact resolution, rather than a
// different one, is what stops an admin of group A from deleting a match in
// group B: the group the middleware authorized against must be the same one
// the delete is scoped to.
func (h *MatchHandler) DeleteMatch(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid match id"})
		return
	}
	groupID, ok := resolveGroupID(c, h.MembershipService)
	if !ok {
		return
	}

	if err := h.Service.DeleteMatch(id, groupID); err != nil {
		switch {
		case errors.Is(err, services.ErrMatchNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": true})
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
