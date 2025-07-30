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
	team.ID = uuid.New().String()
	result := s.DB.Create(team)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (s *TeamService) GetTeamByID(id string) (*models.Team, error) {
	var team models.Team
	result := s.DB.First(&team, "id = ?", id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &team, nil
}
