package handlers

import (
	"errors"
	"net/http"

	"app/internal/models"
	"app/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GroupHandler struct {
	Service           *services.GroupService
	MembershipService *services.GroupMembershipService
}

func NewGroupHandler(service *services.GroupService, membershipService *services.GroupMembershipService) *GroupHandler {
	return &GroupHandler{Service: service, MembershipService: membershipService}
}

// CreateGroup creates a group and makes the authenticated caller its first
// member. That membership is what closes the bootstrapping hole: a group with
// no members can never gain one, since POST /groups/:id/players itself
// requires membership of the target group. The service stays group-agnostic
// about who created what — same split as PlayerHandler.CreatePlayer.
//
// The response deliberately doesn't include the invite code (Group carries
// json:"-" on it); the creator reads it back from GET /groups/:id/invite-code
// like any other member.
func (h *GroupHandler) CreateGroup(c *gin.Context) {
	playerID, ok := playerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
		return
	}

	var group models.Group
	if err := c.ShouldBindJSON(&group); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	created, err := h.Service.CreateGroup(group.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := h.MembershipService.AddPlayerToGroup(created.ID, playerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, created)
}

// JoinGroup adds the authenticated caller to the group matching the invite
// code in the body. Unlike every other group route it can't be behind
// RequireGroupMembership — joining a group you don't belong to yet is the
// entire point — so the invite code is the only thing authorizing the join.
func (h *GroupHandler) JoinGroup(c *gin.Context) {
	playerID, ok := playerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
		return
	}

	var body struct {
		InviteCode string `json:"invite_code"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group, err := h.Service.JoinByInviteCode(playerID, body.InviteCode)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInviteCodeNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case errors.Is(err, services.ErrAlreadyMember):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, group)
}

// GetInviteCode returns the group's invite code so a member can share it. It
// is the one endpoint that exposes the code, and its route is guarded by
// RequireGroupMembershipByPathParam — anywhere else the code stays hidden
// behind Group's json:"-".
func (h *GroupHandler) GetInviteCode(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
		return
	}

	group, err := h.Service.GetGroupByID(groupID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"invite_code": group.InviteCode})
}

func (h *GroupHandler) GetGroups(c *gin.Context) {
	groups, err := h.Service.GetGroups()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, groups)
}

func (h *GroupHandler) GetGroupByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
		return
	}
	group, err := h.Service.GetGroupByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
		return
	}
	c.JSON(http.StatusOK, group)
}

func (h *GroupHandler) AddPlayerToGroup(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
		return
	}

	var body struct {
		PlayerID uuid.UUID `json:"player_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.MembershipService.AddPlayerToGroup(groupID, body.PlayerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"group_id": groupID, "player_id": body.PlayerID})
}

func (h *GroupHandler) GetGroupMembers(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
		return
	}

	players, err := h.MembershipService.GetPlayersByGroupID(groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, players)
}
