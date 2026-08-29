package services

import (
	"app/internal/models"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrEmptyPlayerName = errors.New("player name must not be empty")

// ErrPlayerNameAlreadyUsed is UpdateName's rejection when another player
// already holds the requested name (case-insensitively), anywhere in the
// system. This is deliberately stricter than CreatePlayer's total
// permissiveness on names — see UpdateName's own doc comment for why a
// self-service rename is held to a different bar than player creation.
var ErrPlayerNameAlreadyUsed = errors.New("player name is already in use")

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

// UpdateName lets a player rename themselves (PATCH /players/me). Unlike
// CreatePlayer, which allows name collisions on purpose (see the package doc
// comment on ErrEmptyPlayerName / AuthService.SignupNewPlayer), a rename is
// rejected if another player already holds the requested name anywhere in the
// system, case-insensitively — an explicit product decision to hold a
// player's own deliberate choice of name to a stricter bar than account
// creation, even though it's inconsistent with the rest of this codebase's
// stance on names. The uniqueness check excludes the caller's own row
// (`id != ?`) so renaming to your current name, or the same name in a
// different case, is a no-op success rather than a false collision.
func (s *PlayerService) UpdateName(playerID uuid.UUID, newName string) error {
	newName = strings.TrimSpace(newName)
	if newName == "" {
		return ErrEmptyPlayerName
	}

	var count int64
	if err := s.DB.Model(&models.Player{}).
		Where("LOWER(name) = LOWER(?) AND id != ?", newName, playerID).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrPlayerNameAlreadyUsed
	}

	return s.DB.Model(&models.Player{}).Where("id = ?", playerID).Update("name", newName).Error
}
