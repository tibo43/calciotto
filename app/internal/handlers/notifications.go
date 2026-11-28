package handlers

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"
	"time"

	"app/internal/services"

	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	Service *services.NotificationService
	// PublicKey is the VAPID public key the browser subscribes with. Empty
	// when push isn't configured, which the frontend reads as "unavailable".
	PublicKey string
	// TickSecret authenticates the external scheduler (cron-job.org) calling
	// Tick every minute: POST
	// https://api.agoran.app/internal/notifications/tick with
	// Authorization: Bearer <NOTIFICATIONS_TICK_SECRET>. The route isn't
	// registered at all without it.
	TickSecret string
}

func NewNotificationHandler(service *services.NotificationService, publicKey, tickSecret string) *NotificationHandler {
	return &NotificationHandler{Service: service, PublicKey: publicKey, TickSecret: tickSecret}
}

// GetPublicKey answers {"public_key": "..."} — not a secret, but only useful
// to a signed-in player, hence behind authRequired like the rest.
func (h *NotificationHandler) GetPublicKey(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"public_key": h.PublicKey})
}

// Subscribe records the caller's device (the browser's PushSubscription).
func (h *NotificationHandler) Subscribe(c *gin.Context) {
	playerID, ok := playerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, messageBody("missing authentication"))
		return
	}
	var input services.PushSubscriptionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, errorBody(err))
		return
	}
	if err := h.Service.Subscribe(playerID, input); err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidPushSubscription):
			c.JSON(http.StatusBadRequest, errorBody(err))
		default:
			c.JSON(http.StatusInternalServerError, errorBody(err))
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"subscribed": true})
}

// Unsubscribe forgets one of the caller's devices, named by its endpoint.
func (h *NotificationHandler) Unsubscribe(c *gin.Context) {
	playerID, ok := playerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, messageBody("missing authentication"))
		return
	}
	var req struct {
		Endpoint string `json:"endpoint" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody(err))
		return
	}
	if err := h.Service.Unsubscribe(playerID, req.Endpoint); err != nil {
		c.JSON(http.StatusInternalServerError, errorBody(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"unsubscribed": true})
}

// GetPreferences returns all six notification types for the caller, each
// true unless they have turned that one off.
func (h *NotificationHandler) GetPreferences(c *gin.Context) {
	playerID, ok := playerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, messageBody("missing authentication"))
		return
	}
	prefs, err := h.Service.GetPreferences(playerID)
	if err != nil {
		writePreferenceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"preferences": prefs})
}

// UpdatePreferences stores the subset of types in the body and answers the
// full map. A missing preferences object is 400; an unknown type is 400 and
// changes nothing.
func (h *NotificationHandler) UpdatePreferences(c *gin.Context) {
	playerID, ok := playerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, messageBody("missing authentication"))
		return
	}
	var req struct {
		Preferences map[string]bool `json:"preferences"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody(err))
		return
	}
	if req.Preferences == nil {
		c.JSON(http.StatusBadRequest, messageBody("preferences is required"))
		return
	}
	prefs, err := h.Service.UpdatePreferences(playerID, req.Preferences)
	if err != nil {
		writePreferenceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"preferences": prefs})
}

func writePreferenceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrUnknownNotificationType):
		c.JSON(http.StatusBadRequest, errorBody(err))
	default:
		c.JSON(http.StatusInternalServerError, errorBody(err))
	}
}

// Tick sends the time-driven notifications that are due. cron-job.org calls
// POST https://api.agoran.app/internal/notifications/tick every
// minute with `Authorization: Bearer <NOTIFICATIONS_TICK_SECRET>`, compared
// in constant time; anything else is a 401 that says nothing more.
func (h *NotificationHandler) Tick(c *gin.Context) {
	token, found := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
	if !found || h.TickSecret == "" || subtle.ConstantTimeCompare([]byte(token), []byte(h.TickSecret)) != 1 {
		c.JSON(http.StatusUnauthorized, messageBody("unauthorized"))
		return
	}
	result, err := h.Service.Tick(time.Now())
	if err != nil {
		// Still a 500, so the scheduler reports it, but every other step ran:
		// sent says what went out anyway.
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "sent": result})
		return
	}
	c.JSON(http.StatusOK, result)
}
