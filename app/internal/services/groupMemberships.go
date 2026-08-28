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
// group ownerless or deleting it outright.
var ErrLastMember = errors.New("cannot leave the group: no other members to hand it off to")

// ErrCannotRemoveSelf is returned by RemoveMember when the acting owner
// targets their own membership: removing yourself is voluntary departure,
// which goes through LeaveGroup (and its promotion logic) instead, not
// through the owner's "remove a member" power.
var ErrCannotRemoveSelf = errors.New("cannot remove yourself via this action: use the leave-group endpoint instead")

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
// request doesn't specify a group_id. Unlike GroupService.GetDefaultGroup —
// which just returns whichever group's random UUID happens to sort first,
// with no relation to who belongs to it — this only ever returns a group the
// player actually belongs to, so it can't be knocked out from under them by
// someone else creating an unrelated group (see the "second group flips the
// default" incident this replaced).
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
//  2. An owner leaving a group that still has other members first promotes
//     the longest-standing remaining member (by GroupMembership.CreatedAt —
//     same ordering GetFirstGroupForPlayer uses) to owner, so the group is
//     never left without one.
//  3. A plain member leaving a group that still has other members just has
//     their membership row deleted, nothing else changes.
//
// If playerID isn't a member of groupID at all, the lookup below returns
// gorm.ErrRecordNotFound, which is propagated as-is — this should normally
// never happen because the route is behind RequireGroupMembershipByPathParam,
// but the service stays safe on its own regardless.
//
// The promotion and the deletion run inside one transaction (same pattern as
// GroupService.CreateGroup) so the group can never end up, even transiently,
// with two owners or none.
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

		if membership.Role == models.RoleOwner {
			successor := otherMembers[0]
			if err := tx.Model(&models.GroupMembership{}).
				Where("id = ?", successor.ID).
				Update("role", models.RoleOwner).Error; err != nil {
				return err
			}
		}

		return tx.Delete(&membership).Error
	})
}

// RemoveMember removes targetPlayerID's membership in groupID on behalf of
// actingPlayerID, the group's owner (callers are expected to already be
// authorized via RequireGroupOwnerByPathParam, which guarantees the group has
// exactly one owner and actingPlayerID is it).
//
// Unlike LeaveGroup, there is no promotion or "last member" logic here: the
// owner isn't leaving, so the group always keeps its owner regardless of who
// else is removed.
//
// Rules:
//  1. actingPlayerID cannot target itself — that's voluntary departure and
//     belongs to LeaveGroup, which handles handing off ownership. Refused
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
