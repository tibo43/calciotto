package services

import (
	"app/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GoalService struct {
	DB *gorm.DB
}

func NewGoalService(db *gorm.DB) *GoalService {
	return &GoalService{DB: db}
}

func (s *GoalService) CreateGoal(goal *models.Goal) error {
	goal.ID = uuid.New().String()
	result := s.DB.Create(goal)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (s *GoalService) GetGoalByMatchID(match_id string) ([]models.Goal, error) {
	var goals []models.Goal
	result := s.DB.Where("match_id = ?", match_id).Find(&goals)
	if result.Error != nil {
		return nil, result.Error
	}
	return goals, nil
}

func (s *GoalService) GetGoalByPlayerID(player_id string) ([]models.Goal, error) {
	var goals []models.Goal
	result := s.DB.Where("player_id = ?", player_id).Find(&goals)
	if result.Error != nil {
		return nil, result.Error
	}
	return goals, nil
}
