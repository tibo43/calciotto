package services

import (
	"errors"
	"strings"
	"time"

	"app/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrEmailRequired        = errors.New("email must not be empty")
	ErrPasswordRequired     = errors.New("password must not be empty")
	ErrPlayerNotFound       = errors.New("player not found")
	ErrPlayerAlreadyClaimed = errors.New("player already has an account")
	ErrEmailAlreadyUsed     = errors.New("email already in use")
	ErrInvalidCredentials   = errors.New("invalid email or password")
	ErrInvalidToken         = errors.New("invalid or expired token")
)

const tokenTTL = 7 * 24 * time.Hour

// playerClaims is the JWT payload used by AuthService — it carries the
// claimed Player's ID so the middleware can identify the caller without a
// DB round trip.
type playerClaims struct {
	PlayerID uuid.UUID `json:"player_id"`
	jwt.RegisteredClaims
}

// AuthService implements the "claim by name" flow: a Player row must already
// exist (created via PlayerService) before it can be attached to
// email/password credentials through Signup.
type AuthService struct {
	DB     *gorm.DB
	secret []byte
}

// NewAuthService takes the signing secret as a parameter rather than reading
// JWT_SECRET itself, so callers (main.go) control the fail-fast behavior
// when the env var is missing, and tests can pass a fixed secret without
// depending on the environment.
func NewAuthService(db *gorm.DB, secret string) *AuthService {
	return &AuthService{DB: db, secret: []byte(secret)}
}

// Signup attaches email/password credentials to an existing Player. It never
// creates a new Player — the caller must already know the Player's ID (e.g.
// picked from a list in the UI).
func (s *AuthService) Signup(playerID uuid.UUID, email, password string) error {
	email = normalizeEmail(email)
	if email == "" {
		return ErrEmailRequired
	}
	if password == "" {
		return ErrPasswordRequired
	}

	var player models.Player
	if err := s.DB.First(&player, "id = ?", playerID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPlayerNotFound
		}
		return err
	}
	if player.Email != nil {
		return ErrPlayerAlreadyClaimed
	}

	var existing models.Player
	result := s.DB.Where("email = ?", email).First(&existing)
	if result.Error == nil {
		return ErrEmailAlreadyUsed
	}
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return result.Error
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	player.Email = &email
	player.PasswordHash = string(hash)
	return s.DB.Save(&player).Error
}

// Login checks email/password and returns a signed JWT on success. The error
// is intentionally the same sentinel whether the email is unknown or the
// password is wrong, so callers can't use it to enumerate registered emails.
func (s *AuthService) Login(email, password string) (string, error) {
	email = normalizeEmail(email)

	var player models.Player
	if err := s.DB.Where("email = ?", email).First(&player).Error; err != nil {
		return "", ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(player.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	return s.generateToken(player.ID)
}

// ParseToken validates a JWT and returns the player_id claim it carries. Used
// by the auth middleware.
func (s *AuthService) ParseToken(tokenString string) (uuid.UUID, error) {
	claims := &playerClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.secret, nil
	})
	if err != nil || !token.Valid {
		return uuid.Nil, ErrInvalidToken
	}
	return claims.PlayerID, nil
}

func (s *AuthService) generateToken(playerID uuid.UUID) (string, error) {
	now := time.Now()
	claims := playerClaims{
		PlayerID: playerID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(tokenTTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
