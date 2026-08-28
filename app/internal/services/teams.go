package services

import (
	"app/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TeamService struct {
	DB *gorm.DB
}

func NewTeamService(db *gorm.DB) *TeamService {
	return &TeamService{DB: db}
}

func (s *TeamService) GetTeams() ([]models.Team, error) {
	var teams []models.Team
	result := s.DB.Find(&teams)
	if result.Error != nil {
		return nil, result.Error
	}
	return teams, nil
}

func (s *TeamService) CreateTeam(team *models.Team) error {
	team.ID = uuid.New()
	result := s.DB.Create(team)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (s *TeamService) GetTeamByID(id uuid.UUID) (*models.Team, error) {
	var team models.Team
	result := s.DB.First(&team, "id = ?", id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &team, nil
}

// GetTeamsByGroupID returns the teams belonging to a given group (normally
// exactly the 2 default teams created alongside the group).
func (s *TeamService) GetTeamsByGroupID(groupID uuid.UUID) ([]models.Team, error) {
	var teams []models.Team
	result := s.DB.Where("group_id = ?", groupID).Find(&teams)
	if result.Error != nil {
		return nil, result.Error
	}
	return teams, nil
}
