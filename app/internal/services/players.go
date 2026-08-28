package services

import (
	"app/internal/models"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrEmptyPlayerName     = errors.New("player name must not be empty")
	ErrPlayerAlreadyExists = errors.New("player already exists")
)

type PlayerService struct {
	DB *gorm.DB
}

func NewPlayerService(db *gorm.DB) *PlayerService {
	return &PlayerService{DB: db}
}

func (s *PlayerService) CreatePlayer(name string) (uuid.UUID, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return uuid.Nil, ErrEmptyPlayerName
	}

	var existing models.Player
	result := s.DB.Where("LOWER(name) = LOWER(?)", name).First(&existing)
	if result.Error == nil {
		return uuid.Nil, ErrPlayerAlreadyExists
	}
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return uuid.Nil, result.Error
	}

	player := &models.Player{Name: name}
	if result := s.DB.Create(player); result.Error != nil {
		return uuid.Nil, result.Error
	}
	return player.ID, nil
}

func (s *PlayerService) GetPlayers() ([]models.PlayerCustom, error) {
	var players []models.Player
	if result := s.DB.Find(&players); result.Error != nil {
		return nil, result.Error
	}

	var playersCustom []models.PlayerCustom
	for i := range players {
		playersCustom = append(playersCustom, models.PlayerCustom{
			ID:          players[i].ID,
			Name:        players[i].Name,
			GoalsScored: 0,
		})
	}
	return playersCustom, nil
}

// SearchPlayer returns a DTO without Email — this backs a public endpoint
// (GET /players/search), and Player.Email now holds account data that must
// not leak to anyone who merely knows a player's name.
func (s *PlayerService) SearchPlayer(name string) (*models.PlayerCustom, error) {
	var player models.Player
	result := s.DB.First(&player, "name = ?", name)
	if result.Error != nil {
		return nil, result.Error
	}
	return &models.PlayerCustom{ID: player.ID, Name: player.Name, GoalsScored: 0}, nil
}

// GetPlayerByID returns a player as a DTO, without Email — same reasoning as
// SearchPlayer: Player.Email is account data and must not leak through
// endpoints that only need a name.
func (s *PlayerService) GetPlayerByID(id uuid.UUID) (*models.PlayerCustom, error) {
	var player models.Player
	result := s.DB.First(&player, "id = ?", id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &models.PlayerCustom{ID: player.ID, Name: player.Name, GoalsScored: 0}, nil
}
