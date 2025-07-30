package services

import (
	"app/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TeamCompositionService struct {
	DB *gorm.DB
}

func NewTeamCompositionService(db *gorm.DB) *TeamCompositionService {
	return &TeamCompositionService{DB: db}
}

func (s *TeamCompositionService) CreateTeamComposition(teamComposition *models.TeamComposition) error {
	teamComposition.ID = uuid.New().String()
	result := s.DB.Create(teamComposition)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (s *TeamCompositionService) GetTeamCompositionAll() ([]models.TeamComposition, error) {
	var teamCompositions []models.TeamComposition
	result := s.DB.Find(&teamCompositions)
	if result.Error != nil {
		return nil, result.Error
	}
	return teamCompositions, nil
}

func (s *TeamCompositionService) GetTeamCompositionByMatchID(match_id string) ([]models.TeamComposition, error) {
	var teamCompositions []models.TeamComposition
	result := s.DB.Where("match_id = ?", match_id).Find(&teamCompositions)
	if result.Error != nil {
		return nil, result.Error
	}
	return teamCompositions, nil
}

func (s *TeamCompositionService) GetTeamCompositionByTeamID(team_id string) ([]models.TeamComposition, error) {
	var teamCompositions []models.TeamComposition
	result := s.DB.Where("team_id = ?", team_id).Find(&teamCompositions)
	if result.Error != nil {
		return nil, result.Error
	}
	return teamCompositions, nil
}
