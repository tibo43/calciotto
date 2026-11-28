package services

import (
	"app/internal/models"
	"errors"
	"slices"
	"strings"
	"time"

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

// GetPlayerByID returns a player as a DTO, without Email — Player.Email is
// account data and must not leak through endpoints that only need a name.
func (s *PlayerService) GetPlayerByID(id uuid.UUID) (*models.PlayerCustom, error) {
	var player models.Player
	result := s.DB.First(&player, "id = ?", id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &models.PlayerCustom{ID: player.ID, Name: player.Name, GoalsScored: 0}, nil
}

// GetEmailByID returns the player's own email address, or "" for a player
// who has no account yet. It exists for GET /players/me/stats only: the
// caller sees their own address there, and nowhere else is it looked up.
func (s *PlayerService) GetEmailByID(id uuid.UUID) (string, error) {
	var player models.Player
	if err := s.DB.Select("email").First(&player, "id = ?", id).Error; err != nil {
		return "", err
	}
	if player.Email == nil {
		return "", nil
	}
	return *player.Email, nil
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

// GetFeatures lists the features enabled for a player: every launched one
// (models.LaunchedFeatures) plus the hidden ones they have a row for (see
// models.PlayerFeature), sorted and without duplicates.
func (s *PlayerService) GetFeatures(playerID uuid.UUID) ([]string, error) {
	var rows []string
	if err := s.DB.Model(&models.PlayerFeature{}).
		Where("player_id = ?", playerID).
		Pluck("feature", &rows).Error; err != nil {
		return nil, err
	}
	features := append(slices.Clone(models.LaunchedFeatures), rows...)
	slices.Sort(features)
	return slices.Compact(features), nil
}

// HasFeature reports whether a feature is enabled for a player, launched or
// through their own row, by the same rule as GetFeatures.
func (s *PlayerService) HasFeature(playerID uuid.UUID, feature string) (bool, error) {
	features, err := s.GetFeatures(playerID)
	if err != nil {
		return false, err
	}
	return slices.Contains(features, feature), nil
}

func (s *PlayerService) GuideCompleted(playerID uuid.UUID) (bool, error) {
	var player models.Player
	err := s.DB.Select("guide_completed_at").Take(&player, "id = ?", playerID).Error
	return player.GuideCompletedAt != nil, err
}

func (s *PlayerService) CompleteGuide(playerID uuid.UUID) error {
	return s.DB.Model(&models.Player{}).
		Where("id = ? AND email IS NOT NULL AND guide_completed_at IS NULL", playerID).
		UpdateColumn("guide_completed_at", time.Now()).Error
}
