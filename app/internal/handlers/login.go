package handlers

import (
	"app/internal/models"
	"app/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type LoginHandler struct {
	Service *services.LoginService
}

func NewLoginHandler(service *services.LoginService) *LoginHandler {
	return &LoginHandler{Service: service}
}

func (h *LoginHandler) TokenHandler(c *gin.Context) {
	var user models.UserLogin
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	token, err := h.Service.CreateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.SetCookie("token", token, 3600, "/", "localhost", false, true)
	c.Redirect(http.StatusSeeOther, "/")

	c.JSON(http.StatusOK, token)
}
