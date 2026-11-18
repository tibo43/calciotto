package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"app/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/time/rate"
)

type PasskeyHandler struct {
	service          *services.PasskeyService
	auth             *AuthHandler
	enrollmentLimits *ipRateLimiter
}

func NewPasskeyHandler(service *services.PasskeyService, auth *AuthHandler) *PasskeyHandler {
	return &PasskeyHandler{service: service, auth: auth, enrollmentLimits: newIPRateLimiter(rate.Every(time.Minute), 10)}
}

func (h *PasskeyHandler) RegistrationOptions(c *gin.Context) {
	playerID, ok := playerIDFromContext(c)
	if !ok {
		c.Status(http.StatusUnauthorized)
		return
	}
	var request struct {
		Password string `json:"password" binding:"required"`
		Name     string `json:"name"`
	}
	if !bindPasskeyJSON(c, &request) {
		return
	}
	if !h.enrollmentLimits.allow(playerID.String()) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many attempts, try again later"})
		return
	}
	id, options, err := h.service.BeginRegistration(playerID, c.GetHeader("Origin"), request.Password, request.Name)
	if err != nil {
		passkeyError(c, err, http.StatusBadRequest)
		return
	}
	c.JSON(http.StatusOK, gin.H{"challenge_id": id, "publicKey": options.Response})
}

type passkeyResponse struct {
	ChallengeID uuid.UUID       `json:"challenge_id" binding:"required"`
	Credential  json.RawMessage `json:"credential" binding:"required"`
}

func (h *PasskeyHandler) Register(c *gin.Context) {
	playerID, ok := playerIDFromContext(c)
	if !ok {
		c.Status(http.StatusUnauthorized)
		return
	}
	var request passkeyResponse
	if !bindPasskeyJSON(c, &request) {
		return
	}
	key, err := h.service.FinishRegistration(playerID, c.GetHeader("Origin"), request.ChallengeID, request.Credential)
	if err != nil {
		passkeyError(c, err, http.StatusBadRequest)
		return
	}
	c.JSON(http.StatusCreated, key)
}

func (h *PasskeyHandler) LoginOptions(c *gin.Context) {
	if !bindPasskeyJSON(c, nil) {
		return
	}
	id, options, err := h.service.BeginLogin(c.GetHeader("Origin"))
	if err != nil {
		passkeyError(c, err, http.StatusUnauthorized)
		return
	}
	c.JSON(http.StatusOK, gin.H{"challenge_id": id, "publicKey": options.Response})
}

func (h *PasskeyHandler) Login(c *gin.Context) {
	var request passkeyResponse
	if !bindPasskeyJSON(c, &request) {
		return
	}
	session, err := h.service.FinishLogin(c.GetHeader("Origin"), request.ChallengeID, request.Credential)
	if err != nil {
		passkeyError(c, err, http.StatusUnauthorized)
		return
	}
	h.auth.writeSessionCookie(c, session)
	c.JSON(http.StatusOK, gin.H{"token": session.AccessToken})
}

func (h *PasskeyHandler) List(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	playerID, ok := playerIDFromContext(c)
	if !ok {
		c.Status(http.StatusUnauthorized)
		return
	}
	keys, err := h.service.List(playerID)
	if err != nil {
		passkeyError(c, err, http.StatusBadRequest)
		return
	}
	c.JSON(http.StatusOK, keys)
}

func (h *PasskeyHandler) Remove(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	playerID, ok := playerIDFromContext(c)
	if !ok {
		c.Status(http.StatusUnauthorized)
		return
	}
	keyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid passkey ID"})
		return
	}
	if err := h.service.Remove(playerID, keyID); err != nil {
		passkeyError(c, err, http.StatusBadRequest)
		return
	}
	c.Status(http.StatusNoContent)
}

func bindPasskeyJSON(c *gin.Context, request any) bool {
	c.Header("Cache-Control", "no-store")
	if c.ContentType() != "application/json" {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "expected application/json"})
		return false
	}
	if request != nil && c.ShouldBindJSON(request) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid passkey request"})
		return false
	}
	return true
}

func passkeyError(c *gin.Context, err error, proofStatus int) {
	switch {
	case errors.Is(err, services.ErrPasskeyOrigin), errors.Is(err, services.ErrPasskeysDisabled):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrPasskeyName):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrInvalidCredentials):
		c.JSON(http.StatusBadRequest, gin.H{"error": "incorrect password"})
	case errors.Is(err, services.ErrPasskeyProof):
		c.JSON(proofStatus, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "passkeys are temporarily unavailable"})
	}
}
