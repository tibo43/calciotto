package services

import (
	"errors"

	"app/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ErrLastMember is returned by LeaveGroup when the departing player is the
// group's only member (regardless of role): there would be no one left to
// hand the group off to, so the departure is refused rather than leaving the
// group adminless or deleting it outright.
var ErrLastMember = errors.New("cannot leave the group: no other members to hand it off to")

// ErrCannotRemoveSelf is returned by RemoveMember when the acting admin
// targets their own membership: removing yourself is voluntary departure,
// which goes through LeaveGroup (and its promotion logic) instead, not
// through an admin's "remove a member" power.
var ErrCannotRemoveSelf = errors.New("cannot remove yourself via this action: use the leave-group endpoint instead")

// ErrInvalidRole is returned by UpdateMemberRole when the requested role is
// neither models.RoleAdmin nor models.RoleMember — the only two values
// GroupMembership.Role is ever allowed to hold.
var ErrInvalidRole = errors.New("invalid role: must be \"admin\" or \"member\"")

// ErrCannotChangeOwnRole is returned by UpdateMemberRole when the acting
// admin targets their own membership. There is no self-service "step down"
// feature: changing a role always goes through this admin-only path, and
// self-targeting isn't supported — an admin who wants out of the group leaves
// it via LeaveGroup, which hands the role over on their behalf.
var ErrCannotChangeOwnRole = errors.New("cannot change your own role")

// ErrLastAdmin is returned by UpdateMemberRole when demoting the target would
// leave the group with members but no admin — the same invariant LeaveGroup
// maintains by promoting a successor, enforced here too since a role change
// is the second way it could be broken.
var ErrLastAdmin = errors.New("cannot demote the group's last admin")

// ErrDuplicatePlayerNameInGroup is surfaced by PlayerHandler.CreatePlayer when
// HasMemberNamed reports the target group already has a member with the
// requested name. It's a soft, per-group safety net against an admin
// accidentally creating the same "ghost" player twice (a typo, a double
// click) — not a global uniqueness rule: Player.Name is deliberately allowed
// to collide across unrelated players in unrelated groups (see
// AuthService.SignupNewPlayer).
var ErrDuplicatePlayerNameInGroup = errors.New("a player with this name already exists in this group")

type GroupMembershipService struct {
	DB *gorm.DB
}

func NewGroupMembershipService(db *gorm.DB) *GroupMembershipService {
	return &GroupMembershipService{DB: db}
}

// AddPlayerToGroup adds a player to a group as a plain member. Duplicate
// memberships are rejected by the DB-level unique index on
// (group_id, player_id). It's a thin wrapper over AddPlayerToGroupWithRole
// kept at this signature because too many call sites depend on it
// (PlayerHandler.CreatePlayer, GroupHandler.JoinGroup/AddPlayerToGroup,
// cmd/seed/main.go, several tests) — GroupHandler.CreateGroup is the only
// caller that needs a different role, and calls AddPlayerToGroupWithRole
// directly instead.
func (s *GroupMembershipService) AddPlayerToGroup(groupID, playerID uuid.UUID) error {
	return s.AddPlayerToGroupWithRole(groupID, playerID, models.RoleMember)
}

// AddPlayerToGroupWithRole adds a player to a group with the given role.
// Duplicate memberships are rejected by the DB-level unique index on
// (group_id, player_id).
func (s *GroupMembershipService) AddPlayerToGroupWithRole(groupID, playerID uuid.UUID, role string) error {
	membership := &models.GroupMembership{GroupID: groupID, PlayerID: playerID, Role: role}
	result := s.DB.Create(membership)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (s *GroupMembershipService) GetPlayersByGroupID(groupID uuid.UUID) ([]models.Player, error) {
	var players []models.Player
	result := s.DB.Joins("JOIN group_memberships ON group_memberships.player_id = players.id").
		Where("group_memberships.group_id = ?", groupID).
		Find(&players)
	if result.Error != nil {
		return nil, result.Error
	}
	return players, nil
}

// HasMemberNamed reports whether groupID already has a member whose
// Player.Name matches name case-insensitively. Used by PlayerHandler.CreatePlayer
// as the soft, per-group duplicate guard described by ErrDuplicatePlayerNameInGroup
// — two different groups can each have their own "Marco" with no conflict,
// only a collision within the same group is flagged. Follows the same
// players-joined-to-group-memberships join pattern as GetPlayersByGroupID.
func (s *GroupMembershipService) HasMemberNamed(groupID uuid.UUID, name string) (bool, error) {
	var count int64
	result := s.DB.Model(&models.Player{}).
		Joins("JOIN group_memberships ON group_memberships.player_id = players.id").
		Where("group_memberships.group_id = ? AND LOWER(players.name) = LOWER(?)", groupID, name).
		Count(&count)
	if result.Error != nil {
		return false, result.Error
	}
	return count > 0, nil
}

// IsMember reports whether playerID belongs to groupID — used by
// RequireGroupMembership to authorize access to group-scoped routes.
func (s *GroupMembershipService) IsMember(groupID, playerID uuid.UUID) (bool, error) {
	var count int64
	result := s.DB.Model(&models.GroupMembership{}).
		Where("group_id = ? AND player_id = ?", groupID, playerID).
		Count(&count)
	if result.Error != nil {
		return false, result.Error
	}
	return count > 0, nil
}

// GetRole returns playerID's role within groupID. It returns an empty string
// and gorm.ErrRecordNotFound if the player doesn't belong to the group at
// all — mirrors the query IsMember runs, but selects the role instead of
// just counting rows.
func (s *GroupMembershipService) GetRole(groupID, playerID uuid.UUID) (string, error) {
	var membership models.GroupMembership
	result := s.DB.
		Where("group_id = ? AND player_id = ?", groupID, playerID).
		First(&membership)
	if result.Error != nil {
		return "", result.Error
	}
	return membership.Role, nil
}

// GetFirstGroupForPlayer returns the group playerID joined first (by
// GroupMembership.CreatedAt), for use as that player's "default" group when a
// request doesn't specify a group_id. Unlike the old sort-by-random-UUID
// fallback this replaced (GroupService.GetDefaultGroup, since removed), this
// only ever returns a group the player actually belongs to, so it can't be
// knocked out from under them by someone else creating an unrelated group
// (see the "second group flips the default" incident this replaced).
func (s *GroupMembershipService) GetFirstGroupForPlayer(playerID uuid.UUID) (*models.Group, error) {
	var group models.Group
	result := s.DB.
		Joins("JOIN group_memberships ON group_memberships.group_id = groups.id").
		Where("group_memberships.player_id = ?", playerID).
		Order("group_memberships.created_at ASC").
		First(&group)
	if result.Error != nil {
		return nil, result.Error
	}
	return &group, nil
}

// LeaveGroup removes playerID's own membership in groupID — a player can only
// ever remove themselves through this method; removing someone else is a
// separate feature that doesn't exist yet.
//
// Rules:
//  1. The group's only member (whatever their role) cannot leave: a group
//     with zero members would have no one to hand it off to, so the
//     departure is refused with ErrLastMember instead of leaving the group
//     memberless or deleting it.
//  2. An admin leaving a group where at least one *other* admin remains
//     just has their membership row deleted — the group still has an admin,
//     so there is nothing to hand over. This is the case multiple admins per
//     group made possible.
//  3. The *last* admin leaving a group that still has other members first
//     promotes the longest-standing remaining member (by
//     GroupMembership.CreatedAt — same ordering GetFirstGroupForPlayer uses)
//     to admin, so the group is never left with members but no admin.
//  4. A plain member leaving a group that still has other members just has
//     their membership row deleted, nothing else changes.
//
// If playerID isn't a member of groupID at all, the lookup below returns
// gorm.ErrRecordNotFound, which is propagated as-is — this should normally
// never happen because the route is behind RequireGroupMembershipByPathParam,
// but the service stays safe on its own regardless.
//
// The promotion and the deletion run inside one transaction (same pattern as
// GroupService.CreateGroup) so the group can never end up, even transiently,
// with no admin at all.
func (s *GroupMembershipService) LeaveGroup(groupID, playerID uuid.UUID) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		var membership models.GroupMembership
		if err := tx.
			Where("group_id = ? AND player_id = ?", groupID, playerID).
			First(&membership).Error; err != nil {
			return err
		}

		var otherMembers []models.GroupMembership
		if err := tx.
			Where("group_id = ? AND player_id <> ?", groupID, playerID).
			Order("created_at ASC").
			Find(&otherMembers).Error; err != nil {
			return err
		}
		if len(otherMembers) == 0 {
			return ErrLastMember
		}

		if membership.Role == models.RoleAdmin && !containsAdmin(otherMembers) {
			successor := otherMembers[0]
			if err := tx.Model(&models.GroupMembership{}).
				Where("id = ?", successor.ID).
				Update("role", models.RoleAdmin).Error; err != nil {
				return err
			}
		}

		return tx.Delete(&membership).Error
	})
}

// containsAdmin reports whether any of the given memberships carries
// models.RoleAdmin — LeaveGroup's test for "does the group still have an
// admin once this player is gone?".
func containsAdmin(memberships []models.GroupMembership) bool {
	for _, m := range memberships {
		if m.Role == models.RoleAdmin {
			return true
		}
	}
	return false
}

// UpdateMemberRole promotes targetPlayerID to admin or demotes them back to
// plain member within groupID, on behalf of actingPlayerID, one of the
// group's admins (callers are expected to already be authorized via
// RequireGroupAdminByPathParam). It is the only way a group gains an admin
// besides its creator, and therefore the only way "several admins per group"
// is reachable at all.
//
// Rules:
//  1. newRole must be models.RoleAdmin or models.RoleMember; anything else is
//     refused with ErrInvalidRole before touching the database.
//  2. actingPlayerID cannot target itself — refused with
//     ErrCannotChangeOwnRole. There is no self-service "step down"; an admin
//     who wants out leaves the group via LeaveGroup instead.
//  3. If targetPlayerID isn't a member of groupID at all, the lookup below
//     returns gorm.ErrRecordNotFound, which is propagated as-is — same
//     handling as RemoveMember.
//  4. Setting the role the target already has is a successful no-op: the
//     caller asked for a state the group is already in, and reporting an
//     error would make a retried request fail for no reason.
//  5. Demoting the group's last admin is refused with ErrLastAdmin, so a
//     group with members always keeps at least one admin — the same invariant
//     LeaveGroup upholds by promoting a successor. Through the HTTP route
//     this can't actually trigger (the acting admin is themselves an admin
//     and, by rule 2, not the target, so another admin always remains); it's
//     a service-level safety net for any other caller.
//
// The check and the update run inside one transaction so a concurrent
// demotion of another admin can't slip between them and strip the group's
// last admin.
func (s *GroupMembershipService) UpdateMemberRole(groupID, actingPlayerID, targetPlayerID uuid.UUID, newRole string) error {
	if newRole != models.RoleAdmin && newRole != models.RoleMember {
		return ErrInvalidRole
	}
	if actingPlayerID == targetPlayerID {
		return ErrCannotChangeOwnRole
	}

	return s.DB.Transaction(func(tx *gorm.DB) error {
		var membership models.GroupMembership
		if err := tx.
			Where("group_id = ? AND player_id = ?", groupID, targetPlayerID).
			First(&membership).Error; err != nil {
			return err
		}

		if membership.Role == newRole {
			return nil
		}

		if newRole == models.RoleMember {
			var otherAdmins int64
			if err := tx.Model(&models.GroupMembership{}).
				Where("group_id = ? AND player_id <> ? AND role = ?", groupID, targetPlayerID, models.RoleAdmin).
				Count(&otherAdmins).Error; err != nil {
				return err
			}
			if otherAdmins == 0 {
				return ErrLastAdmin
			}
		}

		return tx.Model(&models.GroupMembership{}).
			Where("id = ?", membership.ID).
			Update("role", newRole).Error
	})
}

// RemoveMember removes targetPlayerID's membership in groupID on behalf of
// actingPlayerID, one of the group's admins (callers are expected to already
// be authorized via RequireGroupAdminByPathParam).
//
// Unlike LeaveGroup, there is no promotion or "last member" logic here: the
// acting admin isn't leaving, so the group always keeps at least that one
// admin regardless of who else is removed.
//
// Rules:
//  1. actingPlayerID cannot target itself — that's voluntary departure and
//     belongs to LeaveGroup, which hands the admin role over. Refused
//     with ErrCannotRemoveSelf before touching the database.
//  2. If targetPlayerID isn't a member of groupID at all, the lookup below
//     returns gorm.ErrRecordNotFound, which is propagated as-is — same
//     handling as LeaveGroup.
//  3. Otherwise the target's membership row is simply deleted.
func (s *GroupMembershipService) RemoveMember(groupID, actingPlayerID, targetPlayerID uuid.UUID) error {
	if actingPlayerID == targetPlayerID {
		return ErrCannotRemoveSelf
	}

	var membership models.GroupMembership
	if err := s.DB.
		Where("group_id = ? AND player_id = ?", groupID, targetPlayerID).
		First(&membership).Error; err != nil {
		return err
	}

	return s.DB.Delete(&membership).Error
}

func (s *GroupMembershipService) GetGroupsByPlayerID(playerID uuid.UUID) ([]models.Group, error) {
	var groups []models.Group
	result := s.DB.Joins("JOIN group_memberships ON group_memberships.group_id = groups.id").
		Where("group_memberships.player_id = ?", playerID).
		Find(&groups)
	if result.Error != nil {
		return nil, result.Error
	}
	return groups, nil
}

// GetGroupsWithRoleByPlayerID is GetGroupsByPlayerID plus the player's role in
// each group, for GET /groups/me — a client needs to know which of its groups
// it can act as an admin in (create a match, edit scores, change roles)
// without asking group by group. GetGroupsByPlayerID stays as it is: its other
// caller, StandingsService.GetPlayerProfile, has no use for the role.
func (s *GroupMembershipService) GetGroupsWithRoleByPlayerID(playerID uuid.UUID) ([]models.GroupWithRole, error) {
	var groups []models.GroupWithRole
	result := s.DB.Model(&models.Group{}).
		Select("groups.*, group_memberships.role AS role").
		Joins("JOIN group_memberships ON group_memberships.group_id = groups.id").
		Where("group_memberships.player_id = ?", playerID).
		Scan(&groups)
	if result.Error != nil {
		return nil, result.Error
	}
	return groups, nil
}
