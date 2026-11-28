package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"app/internal/services"

	"github.com/gin-gonic/gin"
)

type PlayerHandler struct {
	Service *services.PlayerService
}

func NewPlayerHandler(service *services.PlayerService) *PlayerHandler {
	return &PlayerHandler{Service: service}
}

// UpdateMyName lets the authenticated player rename themselves. Global
// uniqueness (unlike PlayerService.CreatePlayer, which allows name
// collisions on purpose — see CLAUDE.md) is enforced here specifically,
// per an explicit product decision: creation stays permissive, but a
// player actively renaming themselves into a name someone else already
// holds is rejected.
func (h *PlayerHandler) UpdateMyName(c *gin.Context) {
	playerID, ok := playerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	name := strings.TrimSpace(req.Name)
	if err := h.Service.UpdateName(playerID, name); err != nil {
		switch {
		case errors.Is(err, services.ErrEmptyPlayerName), errors.Is(err, services.ErrPlayerNameAlreadyUsed):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"name": name})
}

// GetMyFeatures returns the caller's feature list and guide status.
func (h *PlayerHandler) GetMyFeatures(c *gin.Context) {
	playerID, ok := playerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
		return
	}
	features, err := h.Service.GetFeatures(playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	completed, err := h.Service.GuideCompleted(playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load guide status"})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"features": features, "guide_completed": completed})
}

func (h *PlayerHandler) CompleteGuide(c *gin.Context) {
	playerID, ok := playerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
		return
	}
	if err := h.Service.CompleteGuide(playerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save guide completion"})
		return
	}
	c.Status(http.StatusNoContent)
}

// positionsBody is the body of PUT /players/me/positions and the response of
// both /players/me/positions routes. A pointer, so that a body without the
// key (a typo, or `{}`) is refused instead of silently clearing every
// position; an explicit empty list is how a player clears them.
type positionsBody struct {
	PreferredPositions *[]string `json:"PreferredPositions"`
}

// GetMyPositions returns the caller's own preferred positions.
func (h *PlayerHandler) GetMyPositions(c *gin.Context) {
	playerID, ok := playerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
		return
	}
	positions, err := h.Service.GetPreferredPositions(playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load your positions"})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"PreferredPositions": positions})
}

// UpdateMyPositions replaces the caller's preferred positions. A dedicated
// route rather than a new field on PATCH /players/me: that route is the
// rename, which requires a non-empty name, while the Account page saves the
// positions on their own, at every tap, independently of the name field.
func (h *PlayerHandler) UpdateMyPositions(c *gin.Context) {
	playerID, ok := playerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
		return
	}
	var req positionsBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "PreferredPositions must be a list of positions"})
		return
	}
	if req.PreferredPositions == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "PreferredPositions is required"})
		return
	}
	positions, err := h.Service.UpdatePreferredPositions(playerID, *req.PreferredPositions)
	if err != nil {
		if errors.Is(err, services.ErrInvalidPosition) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save your positions"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"PreferredPositions": positions})
}

// GetMatchPositions lists the preferred positions of the match's group
// members, shown to every member in the line-ups and used by the admin
// composing the teams. A route of its own behind
// RequireGroupMembershipByMatchPathParam (a non-member gets 404) rather than a
// field on the sign-up list or the match details, so the shared payloads stay
// exactly as they were.
func (h *PlayerHandler) GetMatchPositions(c *gin.Context) {
	groupID, ok := authorizedGroupIDFromContext(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "route is missing its group authorization middleware"})
		return
	}
	entries, err := h.Service.ListGroupPreferredPositions(groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load positions"})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, entries)
}

// localeBody is the body of PUT /players/me/locale and the response of both
// /players/me/locale routes. "locale" is a pointer inside a presence check
// below: null (or "") is how a player goes back to following the phone, but a
// body without the key (a typo, or `{}`) is refused instead of silently
// resetting the choice.
type localeBody struct {
	Locale *string `json:"locale"`
}

// GetMyLocale returns the language the caller chose, null when they follow
// the phone.
func (h *PlayerHandler) GetMyLocale(c *gin.Context) {
	playerID, ok := playerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
		return
	}
	locale, err := h.Service.GetLocale(playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load your language"})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, localeBody{Locale: locale})
}

// UpdateMyLocale saves the caller's language, or null to follow the phone. A
// dedicated route, like positions: the Account page saves it on its own, at
// each pick, independently of the name field.
func (h *PlayerHandler) UpdateMyLocale(c *gin.Context) {
	playerID, ok := playerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
		return
	}
	var raw map[string]json.RawMessage
	if err := c.ShouldBindJSON(&raw); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "locale must be a language code or null"})
		return
	}
	if _, present := raw["locale"]; !present {
		c.JSON(http.StatusBadRequest, gin.H{"error": "locale is required (a language code, or null to follow the phone)"})
		return
	}
	var req localeBody
	if err := json.Unmarshal(raw["locale"], &req.Locale); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "locale must be a language code or null"})
		return
	}
	locale, err := h.Service.UpdateLocale(playerID, req.Locale)
	if err != nil {
		if errors.Is(err, services.ErrUnsupportedLocale) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save your language"})
		return
	}
	c.JSON(http.StatusOK, localeBody{Locale: locale})
}
