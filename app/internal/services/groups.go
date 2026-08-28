package services

import (
	"app/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// defaultTeamColours are the two teams every group is given automatically.
var defaultTeamColours = []string{"black", "white"}

type GroupService struct {
	DB *gorm.DB
}

func NewGroupService(db *gorm.DB) *GroupService {
	return &GroupService{DB: db}
}

// CreateGroup creates a new group along with its two default teams
// (black/white), since every group must always have exactly these two teams.
func (s *GroupService) CreateGroup(name string) (*models.Group, error) {
	group := &models.Group{Name: name}
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if result := tx.Create(group); result.Error != nil {
			return result.Error
		}
		teamService := NewTeamService(tx)
		for _, colour := range defaultTeamColours {
			team := &models.Team{GroupID: group.ID, Colour: colour}
			if err := teamService.CreateTeam(team); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return group, nil
}

func (s *GroupService) GetGroups() ([]models.Group, error) {
	var groups []models.Group
	result := s.DB.Find(&groups)
	if result.Error != nil {
		return nil, result.Error
	}
	return groups, nil
}

func (s *GroupService) GetGroupByID(id uuid.UUID) (*models.Group, error) {
	var group models.Group
	result := s.DB.First(&group, "id = ?", id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &group, nil
}

// GetDefaultGroup returns whichever group's (random) UUID sorts first — it
// has no relation to any particular caller or player. That makes it unsafe
// as a fallback for anything authenticated: creating a second group can
// silently flip which group is "default" and, combined with a group
// membership check, lock out every existing user who never passes a
// group_id explicitly (see the incident this was pulled from PlayerHandler
// and MatchHandler's read/list paths for). Only use it where there is no
// authenticated player to resolve a real group for instead — currently just
// PlayerHandler.CreatePlayer, whose route is intentionally public. Anywhere
// behind AuthMiddleware, use GroupMembershipService.GetFirstGroupForPlayer.
func (s *GroupService) GetDefaultGroup() (*models.Group, error) {
	var group models.Group
	result := s.DB.Order("id").First(&group)
	if result.Error != nil {
		return nil, result.Error
	}
	return &group, nil
}
