package handlers

import (
	"errors"
	"net/http"
	"strings"

	"app/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PlayerHandler struct {
	Service           *services.PlayerService
	GroupService      *services.GroupService
	MembershipService *services.GroupMembershipService
}

func NewPlayerHandler(service *services.PlayerService, groupService *services.GroupService, membershipService *services.GroupMembershipService) *PlayerHandler {
	return &PlayerHandler{Service: service, GroupService: groupService, MembershipService: membershipService}
}

// CreatePlayer creates a player (a "ghost" roster entry — see
// MatchDetails.vue's create-on-the-fly flow), then attaches it to a group —
// group_id in the body if provided, else the caller's own first group
// (GroupMembershipService.GetFirstGroupForPlayer), same fallback
// RequireGroupAdmin's own group resolution (resolveGroupIDForMembership)
// already uses. The route sits behind authRequired + requireGroupAdmin
// (main.go), which authorizes the request against whichever group it
// resolves via that same fallback — resolving to a *different* group here
// would let an admin of group A silently create the player in an unrelated
// group B, so this must stay in lockstep with the middleware.
// PlayerService itself stays group-agnostic; the membership (and the
// per-group duplicate-name check) is handled here in the handler.
func (h *PlayerHandler) CreatePlayer(c *gin.Context) {
	var req struct {
		Name    string    `json:"name"`
		GroupID uuid.UUID `json:"group_id"`
	}
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

	// Soft, per-group duplicate guard: Player.Name is no longer globally
	// unique (see AuthService.SignupNewPlayer), so this only rejects a name
	// collision within the target group's own roster — a safety net against
	// an admin accidentally creating the same "ghost" player twice.
	name := strings.TrimSpace(req.Name)
	hasMember, err := h.MembershipService.HasMemberNamed(groupID, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if hasMember {
		c.JSON(http.StatusBadRequest, gin.H{"error": services.ErrDuplicatePlayerNameInGroup.Error()})
		return
	}

	id, err := h.Service.CreatePlayer(name)
	if err != nil {
		if errors.Is(err, services.ErrEmptyPlayerName) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := h.MembershipService.AddPlayerToGroup(groupID, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, id)
}

// UpdateMyName lets the authenticated player rename themselves. Global
// uniqueness (unlike PlayerService.CreatePlayer, which allows name
// collisions on purpose — see CLAUDE.md) is enforced here specifically,
// per an explicit product decision: creation stays permissive, but a
// player actively renaming themselves into a name someone else already
// holds is rejected.
func (h *PlayerHandler) UpdateMyName(c *gin.Context) {
	playerID, ok := playerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	name := strings.TrimSpace(req.Name)
	if err := h.Service.UpdateName(playerID, name); err != nil {
		switch {
		case errors.Is(err, services.ErrEmptyPlayerName), errors.Is(err, services.ErrPlayerNameAlreadyUsed):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"name": name})
}
