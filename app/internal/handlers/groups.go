package handlers

import (
	"errors"
	"net/http"

	"app/internal/models"
	"app/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GroupHandler struct {
	Service           *services.GroupService
	MembershipService *services.GroupMembershipService
}

func NewGroupHandler(service *services.GroupService, membershipService *services.GroupMembershipService) *GroupHandler {
	return &GroupHandler{Service: service, MembershipService: membershipService}
}

// CreateGroup is temporarily disabled — a deliberate, reversible business
// decision, not a technical limitation: players must not self-service a new
// group right now. The underlying logic (create the group, its two teams,
// and make the caller its first admin) is untouched in GroupService.CreateGroup
// and GroupMembershipService.AddPlayerToGroupWithRole; re-enabling this route
// is a matter of restoring this handler's old body, not rebuilding anything.
func (h *GroupHandler) CreateGroup(c *gin.Context) {
	c.JSON(http.StatusForbidden, gin.H{"error": "group creation is not available yet"})
}

// JoinGroup is temporarily disabled for the same reason as CreateGroup above
// — players must not self-service joining a group by invite code right now.
// GroupService.JoinByInviteCode is untouched.
func (h *GroupHandler) JoinGroup(c *gin.Context) {
	c.JSON(http.StatusForbidden, gin.H{"error": "joining a group is not available yet"})
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

// GetGroupMembers lists the group's members, each tagged with their role
// (models.PlayerWithRole) so a client can show who's an admin and, if the
// caller is one themselves, offer role-change/remove controls — open to any
// member (route sits behind RequireGroupMembershipByPathParam only, no admin
// gate: seeing the roster and who administers it isn't privileged, only
// acting on it is, and that's enforced by UpdateMemberRole/RemoveMember's own
// admin-only routes).
func (h *GroupHandler) GetGroupMembers(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
		return
	}

	players, err := h.MembershipService.GetPlayersWithRoleByGroupID(groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, players)
}

// LeaveGroup lets the authenticated caller remove their own membership from
// the group in the URL. It's deliberately self-service only — there is no way
// to target another player's membership here, that's a separate "remove a
// member" feature that doesn't exist yet. The route sits behind
// RequireGroupMembershipByPathParam, so the caller is already known to be a
// member by the time this runs; the actual leave/promote/last-member logic
// lives in GroupMembershipService.LeaveGroup.
func (h *GroupHandler) LeaveGroup(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
		return
	}

	playerID, ok := playerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
		return
	}

	if err := h.MembershipService.LeaveGroup(groupID, playerID); err != nil {
		if errors.Is(err, services.ErrLastMember) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"left": true})
}

// SetFavoriteGroup marks the group in the URL as the caller's favorite — a
// personal preference, not an admin action, so it's self-service the same
// way LeaveGroup is: RequireGroupMembershipByPathParam already establishes
// the caller belongs to this group by the time this runs.
func (h *GroupHandler) SetFavoriteGroup(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
		return
	}

	playerID, ok := playerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
		return
	}

	if err := h.MembershipService.SetFavoriteGroup(playerID, groupID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"favorite": true})
}

// RemoveMember lets a group admin remove another member's membership from the
// group in the URL. The route sits behind RequireGroupAdminByPathParam, so the
// caller is already confirmed to be one of that group's admins by the time
// this runs.
//
// It deliberately cannot be used to remove the caller's own membership — an
// admin who wants to leave has to go through DELETE /groups/:id/members/me
// (LeaveGroup), which handles promoting a successor when they were the last
// admin. Targeting yourself here is rejected with
// services.ErrCannotRemoveSelf before any deletion happens.
func (h *GroupHandler) RemoveMember(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
		return
	}

	targetPlayerID, err := uuid.Parse(c.Param("playerId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid player id"})
		return
	}

	actingPlayerID, ok := playerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
		return
	}

	if err := h.MembershipService.RemoveMember(groupID, actingPlayerID, targetPlayerID); err != nil {
		switch {
		case errors.Is(err, services.ErrCannotRemoveSelf):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, gorm.ErrRecordNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "player is not a member of this group"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"removed": true})
}

// UpdateMemberRole lets a group admin promote another member to admin, or
// demote another admin back to plain member, in the group in the URL. The
// route (PATCH /groups/:id/members/:playerId/role) sits behind
// RequireGroupAdminByPathParam, so the caller is already confirmed to be one
// of that group's admins by the time this runs; every other rule — valid role
// value, no self-targeting, target must be a member, never demote the last
// admin — lives in GroupMembershipService.UpdateMemberRole.
func (h *GroupHandler) UpdateMemberRole(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
		return
	}

	targetPlayerID, err := uuid.Parse(c.Param("playerId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid player id"})
		return
	}

	actingPlayerID, ok := playerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
		return
	}

	var body struct {
		Role string `json:"role"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.MembershipService.UpdateMemberRole(groupID, actingPlayerID, targetPlayerID, body.Role); err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidRole),
			errors.Is(err, services.ErrCannotChangeOwnRole),
			errors.Is(err, services.ErrLastAdmin):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, gorm.ErrRecordNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "player is not a member of this group"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"group_id": groupID, "player_id": targetPlayerID, "role": body.Role})
}

// GetMyGroups returns the groups the authenticated caller belongs to, each
// tagged with the caller's own role in it (models.GroupWithRole). The public
// GET /groups lists every group in the system, which is useless to a client
// asking "which groups are mine" — and it can't be narrowed, being
// unauthenticated by design, nor carry a role, having no caller.
//
// The role is what lets a client know where it may act as an admin (create a
// match, edit scores, change roles) without one request per group. The DTO
// embeds Group, which keeps InviteCode behind json:"-", so the code stays
// exclusive to GET /groups/:id/invite-code.
//
// authRequired alone, no requireGroupMember: like GET /players/me/stats there
// is no single group_id to authorize against, and the handler only ever
// reports on the JWT's own player.
func (h *GroupHandler) GetMyGroups(c *gin.Context) {
	playerID, ok := playerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
		return
	}

	groups, err := h.MembershipService.GetGroupsWithRoleByPlayerID(playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Belonging to no group is a normal state (a freshly signed-up player),
	// not an error — but GORM leaves the slice nil, which would serialize as
	// `null` and force every caller to special-case it.
	if groups == nil {
		groups = []models.GroupWithRole{}
	}

	c.JSON(http.StatusOK, groups)
}
