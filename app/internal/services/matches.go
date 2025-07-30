package services

import (
	"app/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MatchService struct {
	DB *gorm.DB
}

func NewMatchService(db *gorm.DB) *MatchService {
	return &MatchService{DB: db}
}

func (s *MatchService) GetMatches() ([]models.Match, error) {
	var matches []models.Match
	result := s.DB.Find(&matches)
	if result.Error != nil {
		return nil, result.Error
	}
	return matches, nil
}

func (s *MatchService) CreateMatch(match *models.Match) error {
	match.ID = uuid.New().String()
	result := s.DB.Create(match)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (s *MatchService) GetMatchByID(id string) (*models.Match, error) {
	var match models.Match
	result := s.DB.First(&match, "id = ?", id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &match, nil
}
