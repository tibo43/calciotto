package services

import (
	"app/internal/models"
	"app/pkg/common"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type LoginService struct {
	DB *gorm.DB
}

func NewLoginService(db *gorm.DB) *LoginService {
	return &LoginService{DB: db}
}

func (s *LoginService) CreateToken(userLogin models.UserLogin) (string, error) {
	var userDB models.UserLogin
	result := s.DB.Raw(`SELECT user_name as user_name, password as password, is_admin as is_admin FROM users WHERE user_name=?`, userLogin.UserName).Scan(&userDB)
	if result.Error != nil {
		return "", result.Error
	}

	if !common.CheckPassword(userLogin.Password, userDB.Password) {
		return "Password didn't match", nil
	}

	// Replace this with your actual secret key (e.g., from environment variables)
	var jwtSecret = []byte("truc-much")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"username": userDB.UserName,
			"is_admin": userDB.IsAdmin,
			"exp":      time.Now().Add(time.Hour).Unix(),
		})
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return err.Error(), nil
	}
	return tokenString, nil
}
