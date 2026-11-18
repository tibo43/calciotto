package handlers

import (
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"app/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const refreshCookieName = "calciotto-refresh"

func (h *AuthHandler) writeSessionCookie(c *gin.Context, credentials services.SessionCredentials) {
	c.Header("Cache-Control", "no-store")
	http.SetCookie(c.Writer, &http.Cookie{
		Name: refreshCookieName, Value: credentials.RefreshToken,
		Path: "/api/auth", Expires: credentials.ExpiresAt,
		HttpOnly: true, Secure: h.SecureCookies, SameSite: http.SameSiteStrictMode,
	})
}

func (h *AuthHandler) RefreshSession(c *gin.Context) {
	cookie, ok := h.sessionCookie(c)
	if !ok {
		return
	}
	credentials, err := h.Service.RefreshSession(cookie)
	switch {
	case errors.Is(err, services.ErrRefreshRotated):
		c.JSON(http.StatusConflict, gin.H{"error": "session just renewed, retry"})
	case errors.Is(err, services.ErrInvalidToken):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired session"})
	case err != nil:
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "unable to renew the session"})
	default:
		h.writeSessionCookie(c, credentials)
		c.JSON(http.StatusOK, gin.H{"token": credentials.AccessToken})
	}
}

func (h *AuthHandler) Logout(c *gin.Context) {
	cookie, ok := h.sessionCookie(c)
	if !ok {
		return
	}
	if err := h.Service.EndSession(cookie); err != nil && !errors.Is(err, services.ErrInvalidToken) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "unable to end the session"})
		return
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name: refreshCookieName, Path: "/api/auth", MaxAge: -1, Expires: time.Unix(1, 0),
		HttpOnly: true, Secure: h.SecureCookies, SameSite: http.SameSiteStrictMode,
	})
	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) sessionCookie(c *gin.Context) (string, bool) {
	c.Header("Cache-Control", "no-store")
	origin := c.GetHeader("Origin")
	if origin == "" || !slices.Contains(h.SessionOrigins, origin) {
		c.JSON(http.StatusForbidden, gin.H{"error": "invalid session request origin"})
		return "", false
	}
	expected, err := uuid.Parse(c.GetHeader("X-Session-ID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing or invalid session ID"})
		return "", false
	}
	cookie, _ := c.Cookie(refreshCookieName)
	rawID, _, _ := strings.Cut(cookie, ".")
	if actual, err := uuid.Parse(rawID); err == nil && actual != expected {
		// Another login may have replaced the cookie in a different tab.
		c.JSON(http.StatusConflict, gin.H{"error": "the active session changed"})
		return "", false
	}
	return cookie, true
}
