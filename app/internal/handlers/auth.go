package handlers

import (
	"errors"
	"net/http"

	"app/internal/services"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	Service *services.AuthService
}

func NewAuthHandler(service *services.AuthService) *AuthHandler {
	return &AuthHandler{Service: service}
}

// Signup is the public "create your account" flow: it creates a brand-new
// Player from the submitted name and attaches email/password credentials to
// it in one step. It does not accept a player_id — there is no existing
// Player to pick from anymore, unlike the old "claim by name" flow that used
// to back this route (see AuthService.Signup / AuthService.SignupNewPlayer).
func (h *AuthHandler) Signup(c *gin.Context) {
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

	playerID, err := h.Service.SignupNewPlayer(req.Name, req.Email, req.Password, req.InviteCode)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrEmptyPlayerName),
			errors.Is(err, services.ErrEmailRequired),
			errors.Is(err, services.ErrPasswordRequired),
			errors.Is(err, services.ErrEmailAlreadyUsed),
			errors.Is(err, services.ErrInviteCodeRequired),
			errors.Is(err, services.ErrInviteCodeNotFound):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": playerID})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := h.Service.Login(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

// passwordResetRequestedMessage is returned by ForgotPassword whatever
// happened, so the response body can't be used to tell a registered email from
// an unregistered one.
const passwordResetRequestedMessage = "if an account exists for this email, a reset link has been sent"

func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// An unknown email is not an error here — AuthService.ForgotPassword
	// returns nil for it on purpose — so only a genuine server-side failure
	// (e.g. the DB being unreachable) can reach this branch.
	if err := h.Service.ForgotPassword(req.Email); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
			errors.Is(err, services.ErrPasswordRequired):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password updated"})
}
