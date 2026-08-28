package services

import (
	"app/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

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
