package services

import (
	"app/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PlayerService struct {
	DB *gorm.DB
}

func NewPlayerService(db *gorm.DB) *PlayerService {
	return &PlayerService{DB: db}
}

func (s *PlayerService) GetPlayers() ([]models.Player, error) {
	var players []models.Player
	result := s.DB.Find(&players)
	if result.Error != nil {
		return nil, result.Error
	}
	return players, nil
}

func (s *PlayerService) CreatePlayer(player *models.Player) error {
	player.ID = uuid.New().String()
	result := s.DB.Create(player)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (s *PlayerService) GetPlayerByID(id string) (*models.Player, error) {
	var player models.Player
	result := s.DB.First(&player, "id = ?", id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &player, nil
}
