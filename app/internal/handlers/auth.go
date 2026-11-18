package handlers

import (
	"crypto/sha256"
	"errors"
	"net/http"
	"strings"
	"time"

	"app/internal/services"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type AuthHandler struct {
	Service        *services.AuthService
	SessionOrigins []string
	SecureCookies  bool
	loginByAccount *ipRateLimiter
	resetByAccount *ipRateLimiter
}

func NewAuthHandler(service *services.AuthService) *AuthHandler {
	return &AuthHandler{
		Service:        service,
		loginByAccount: newIPRateLimiter(rate.Every(time.Minute), 12),
		resetByAccount: newIPRateLimiter(rate.Every(30*time.Minute), 3),
	}
}

func accountRateKey(email string) string {
	hash := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email))))
	return string(hash[:])
}

// Signup is the public "create your account" flow: it creates a brand-new
// Player from the submitted name and attaches email/password credentials to
// it in one step. It does not accept a player_id — there is no existing
// Player to pick from anymore, unlike the old "claim by name" flow that used
// to back this route (see AuthService.Signup / AuthService.SignupNewPlayer).
//
// The response includes a ready-to-use JWT (the same shape Login returns)
// alongside the player id: signing up and logging in are one action from the
// caller's point of view, so the frontend shouldn't have to resend the
// password through a second POST /auth/login just to get a token.
func (h *AuthHandler) Signup(c *gin.Context) {
	if c.ContentType() != "application/json" {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "expected application/json"})
		return
	}
	var req struct {
		Name       string `json:"name"`
		Email      string `json:"email"`
		Password   string `json:"password"`
		InviteCode string `json:"invite_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	playerID, credentials, err := h.Service.SignupSession(req.Name, req.Email, req.Password, req.InviteCode)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrEmptyPlayerName),
			errors.Is(err, services.ErrEmailRequired),
			errors.Is(err, services.ErrPasswordRequired),
			// Mapped alongside ErrPasswordRequired: both are the client's own
			// input failing validation, and ErrPasswordTooShort's message
			// already names the required length, so passing it through is what
			// makes the 400 actionable.
			errors.Is(err, services.ErrPasswordTooShort),
			errors.Is(err, services.ErrEmailAlreadyUsed),
			errors.Is(err, services.ErrInviteCodeRequired),
			errors.Is(err, services.ErrInviteCodeNotFound):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "unable to complete signup"})
		}
		return
	}

	h.writeSessionCookie(c, credentials)
	c.JSON(http.StatusOK, gin.H{"id": playerID, "token": credentials.AccessToken})
}

func (h *AuthHandler) Login(c *gin.Context) {
	if c.ContentType() != "application/json" {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "expected application/json"})
		return
	}
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !h.loginByAccount.allow(accountRateKey(req.Email)) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, please try again later"})
		return
	}
	credentials, err := h.Service.LoginSession(req.Email, req.Password)
	if errors.Is(err, services.ErrInvalidCredentials) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": services.ErrInvalidCredentials.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "unable to start the session"})
		return
	}

	h.writeSessionCookie(c, credentials)
	c.JSON(http.StatusOK, gin.H{"token": credentials.AccessToken})
}

// passwordResetRequestedMessage is the same for known and unknown accounts.
const passwordResetRequestedMessage = "if an account exists for this email, a reset link has been sent"

func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.Service.PasswordResetMailer != nil && !h.resetByAccount.allow(accountRateKey(req.Email)) {
		c.JSON(http.StatusOK, gin.H{"message": passwordResetRequestedMessage})
		return
	}

	if err := h.Service.ForgotPassword(req.Email); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, services.ErrPasswordResetUnavailable) {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, gin.H{"error": services.ErrPasswordResetUnavailable.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": passwordResetRequestedMessage})
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.Service.ResetPassword(req.Token, req.NewPassword); err != nil {
		switch {
		// ErrInvalidResetToken's own message is already generic (it covers
		// unknown/expired/used alike), so passing it through leaks nothing.
		case errors.Is(err, services.ErrInvalidResetToken),
			errors.Is(err, services.ErrPasswordRequired),
			errors.Is(err, services.ErrPasswordTooShort):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password updated"})
}

// DeleteAccount lets the authenticated player permanently delete their own
// account (DELETE /players/me). The current password is required in the body
// as a safety confirmation. See AuthService.DeleteAccount for what actually
// happens (the account is anonymized, not erased) and why.
//
// A wrong password answers 400, not 401 the way Login answers a rejected
// login attempt: the frontend's single axios instance treats *any* 401,
// system-wide, as "this session is no longer valid" and force-logs-out the
// caller (see api.js's response interceptor) — appropriate for a token the
// backend itself rejected, but a real bug here, since the caller is already
// authenticated and simply mistyped a confirmation password. 400 keeps that
// global interceptor from firing over what is a validation failure, not a
// session one.
func (h *AuthHandler) DeleteAccount(c *gin.Context) {
	playerID, ok := playerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.Service.DeleteAccount(playerID, req.Password); err != nil {
		switch {
		case errors.Is(err, services.ErrPasswordRequired), errors.Is(err, services.ErrInvalidCredentials):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": true})
}
