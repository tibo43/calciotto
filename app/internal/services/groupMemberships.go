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

// AddPlayerToGroup adds a player to a group. Duplicate memberships are
// rejected by the DB-level unique index on (group_id, player_id).
func (s *GroupMembershipService) AddPlayerToGroup(groupID, playerID uuid.UUID) error {
	membership := &models.GroupMembership{GroupID: groupID, PlayerID: playerID}
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
