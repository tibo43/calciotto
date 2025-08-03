package services

import (
	"app/internal/models"
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PlayerService struct {
	DB *gorm.DB
}

func NewPlayerService(db *gorm.DB) *PlayerService {
	return &PlayerService{DB: db}
}

func (s *PlayerService) GetPlayers() ([]models.PlayerCustom, error) {
	var playersCustom []models.PlayerCustom
	var players []models.Player
	result := s.DB.Find(&players)

	for i := range players {
		playersCustom = append(playersCustom, models.PlayerCustom{
			ID:         players[i].ID,
			Name:       players[i].Name,
			GoalNumber: 0,
		})
	}

	if result.Error != nil {
		return nil, result.Error
	}
	return playersCustom, nil
}

func (s *PlayerService) CreatePlayer(player *models.Player) error {
	player.ID = uuid.New().String()
	result := s.DB.Create(player)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (s *PlayerService) SearchPlayer(name string) (*models.Player, error) {
	var player models.Player
	result := s.DB.First(&player, "name = ?", name)
	if result.Error != nil {
		return nil, result.Error
	}
	log.Println("Found player:", player)
	return &player, nil
}
