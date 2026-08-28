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
