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
	// TickSecret authenticates the scheduler calling Tick (see
	// notifications-tick.yml). The route isn't registered at all without it.
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
// 403 without the push-notifications feature: the card that calls this is
// hidden then, so reaching it means calling the API by hand.
func (h *NotificationHandler) Subscribe(c *gin.Context) {
	playerID, ok := playerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
		return
	}
	var input services.PushSubscriptionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.Service.Subscribe(playerID, input); err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidPushSubscription):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, services.ErrPushFeatureNotEnabled):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"subscribed": true})
}

// Unsubscribe forgets one of the caller's devices, named by its endpoint.
func (h *NotificationHandler) Unsubscribe(c *gin.Context) {
	playerID, ok := playerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
		return
	}
	var req struct {
		Endpoint string `json:"endpoint" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.Service.Unsubscribe(playerID, req.Endpoint); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"unsubscribed": true})
}

// Tick sends the time-driven notifications that are due. Called by the
// scheduler with `Authorization: Bearer <NOTIFICATIONS_TICK_SECRET>`, compared
// in constant time; anything else is a 401 that says nothing more.
func (h *NotificationHandler) Tick(c *gin.Context) {
	token, found := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
	if !found || h.TickSecret == "" || subtle.ConstantTimeCompare([]byte(token), []byte(h.TickSecret)) != 1 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	result, err := h.Service.Tick(time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
