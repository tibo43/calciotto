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

// GetDefaultGroup returns the first group in the system. It is used as an
// implicit fallback for callers (e.g. the current frontend) that don't pass a
// group_id explicitly — this app is expected to operate with a single group
// until multi-group UI support lands.
func (s *GroupService) GetDefaultGroup() (*models.Group, error) {
	var group models.Group
	result := s.DB.Order("id").First(&group)
	if result.Error != nil {
		return nil, result.Error
	}
	return &group, nil
}
