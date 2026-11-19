package services

import (
	"errors"
	"fmt"
	"strings"

	"app/internal/models"

	"github.com/google/uuid"
)

// ErrInvalidPosition rejects a preferred position outside GK, DEF, MID, FWD.
var ErrInvalidPosition = errors.New("invalid position")

// NormalizePreferredPositions validates a player's position picks and returns
// them deduplicated, in canonical order (GK, DEF, MID, FWD), whatever order
// they came in. Surrounding spaces and letter case are forgiven ("gk " is GK);
// any other value is rejected with ErrInvalidPosition, naming it. An empty
// (or nil) list is valid and means "no preferred position".
func NormalizePreferredPositions(input []string) (models.PreferredPositions, error) {
	picked := map[string]bool{}
	for _, raw := range input {
		value := strings.ToUpper(strings.TrimSpace(raw))
		known := false
		for _, allowed := range models.CanonicalPositions {
			if value == allowed {
				known = true
				break
			}
		}
		if !known {
			return nil, fmt.Errorf("%w %q: use GK, DEF, MID or FWD", ErrInvalidPosition, raw)
		}
		picked[value] = true
	}
	out := models.PreferredPositions{}
	for _, allowed := range models.CanonicalPositions {
		if picked[allowed] {
			out = append(out, allowed)
		}
	}
	return out, nil
}

// GetPreferredPositions returns the player's own preferred positions (never
// nil: an empty list when they have none).
func (s *PlayerService) GetPreferredPositions(playerID uuid.UUID) (models.PreferredPositions, error) {
	var player models.Player
	if err := s.DB.Select("preferred_positions").Take(&player, "id = ?", playerID).Error; err != nil {
		return nil, err
	}
	if player.PreferredPositions == nil {
		return models.PreferredPositions{}, nil
	}
	return player.PreferredPositions, nil
}

// UpdatePreferredPositions replaces the player's preferred positions with the
// normalized form of input (see NormalizePreferredPositions) and returns what
// was stored.
func (s *PlayerService) UpdatePreferredPositions(playerID uuid.UUID, input []string) (models.PreferredPositions, error) {
	positions, err := NormalizePreferredPositions(input)
	if err != nil {
		return nil, err
	}
	if err := s.DB.Model(&models.Player{}).Where("id = ?", playerID).
		UpdateColumn("preferred_positions", positions).Error; err != nil {
		return nil, err
	}
	return positions, nil
}

// ListGroupPreferredPositions returns the preferred positions of every member
// of the group who has picked at least one. Admin-only data: the only route
// reading it (GET /matches/:id/positions) sits behind
// RequireGroupAdminByMatchPathParam, and the group is the one that middleware
// authorized, never one the caller supplies.
func (s *PlayerService) ListGroupPreferredPositions(groupID uuid.UUID) ([]models.PlayerPositionsEntry, error) {
	var entries []models.PlayerPositionsEntry
	if err := s.DB.Model(&models.Player{}).
		Select("players.id AS player_id, players.preferred_positions AS preferred_positions").
		Joins("JOIN group_memberships ON group_memberships.player_id = players.id").
		Where("group_memberships.group_id = ?", groupID).
		Where("cardinality(players.preferred_positions) > 0").
		Order("group_memberships.created_at ASC").
		Scan(&entries).Error; err != nil {
		return nil, err
	}
	if entries == nil {
		entries = []models.PlayerPositionsEntry{}
	}
	return entries, nil
}
