package services

import (
	"errors"
	"slices"
	"strings"

	"app/internal/models"

	"github.com/google/uuid"
)

// ErrUnsupportedLocale rejects a language the app has no bundle for.
var ErrUnsupportedLocale = errors.New("unsupported language")

// NormalizeLocale validates a language code and returns its canonical form
// (lower case, no surrounding space). nil, or an empty string, means "follow
// the phone" and stays nil. Anything outside models.SupportedLocales is
// refused with ErrUnsupportedLocale: regional tags such as "fr-CA" are the
// client's to reduce to "fr", not stored as they come.
func NormalizeLocale(input *string) (*string, error) {
	if input == nil {
		return nil, nil
	}
	code := strings.ToLower(strings.TrimSpace(*input))
	if code == "" {
		return nil, nil
	}
	if !slices.Contains(models.SupportedLocales, code) {
		return nil, ErrUnsupportedLocale
	}
	return &code, nil
}

// GetLocale returns the player's saved language, nil when they follow the
// phone.
func (s *PlayerService) GetLocale(playerID uuid.UUID) (*string, error) {
	var player models.Player
	if err := s.DB.Select("locale").Take(&player, "id = ?", playerID).Error; err != nil {
		return nil, err
	}
	return player.Locale, nil
}

// UpdateLocale saves the player's language (nil = follow the phone) and
// returns what was stored.
func (s *PlayerService) UpdateLocale(playerID uuid.UUID, input *string) (*string, error) {
	locale, err := NormalizeLocale(input)
	if err != nil {
		return nil, err
	}
	// UpdateColumn with an explicit nil writes NULL (Updates would skip it).
	if err := s.DB.Model(&models.Player{}).Where("id = ?", playerID).
		UpdateColumn("locale", locale).Error; err != nil {
		return nil, err
	}
	return locale, nil
}
