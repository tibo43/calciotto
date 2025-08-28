package services

import (
	"app/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MatchPlayerService struct {
	DB *gorm.DB
}

func NewMatchPlayerService(db *gorm.DB) *MatchPlayerService {
	return &MatchPlayerService{DB: db}
}

func (s *MatchPlayerService) CreateMatchPlayer(teamComposition *models.MatchPlayer) error {
	teamComposition.ID = uuid.New().String()
	result := s.DB.Create(teamComposition)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (s *MatchPlayerService) GetMatchPlayerAll() ([]models.MatchPlayer, error) {
	var teamCompositions []models.MatchPlayer
	result := s.DB.Find(&teamCompositions)
	if result.Error != nil {
		return nil, result.Error
	}
	return teamCompositions, nil
}

func (s *MatchPlayerService) GetMatchPlayerByMatchID(match_id string) ([]models.MatchPlayer, error) {
	var teamCompositions []models.MatchPlayer
	result := s.DB.Where("match_id = ?", match_id).Find(&teamCompositions)
	if result.Error != nil {
		return nil, result.Error
	}
	return teamCompositions, nil
}

func (s *MatchPlayerService) GetMatchPlayerByTeamID(team_id string) ([]models.MatchPlayer, error) {
	var teamCompositions []models.MatchPlayer
	result := s.DB.Where("team_id = ?", team_id).Find(&teamCompositions)
	if result.Error != nil {
		return nil, result.Error
	}
	return teamCompositions, nil
}
