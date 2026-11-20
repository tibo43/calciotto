package services

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"app/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	accessTokenTTL = 15 * time.Minute
	sessionTTL     = 7 * 24 * time.Hour
	refreshOverlap = time.Minute
)

// ErrRefreshRotated lets a second browser tab retry with the updated cookie.
var ErrRefreshRotated = errors.New("this session was just renewed")

type SessionCredentials struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

func (s *AuthService) LoginSession(email, password string) (SessionCredentials, error) {
	authenticated, err := s.authenticate(email, password)
	if err != nil {
		return SessionCredentials{}, err
	}
	var credentials SessionCredentials
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		player, err := lockPlayer(tx, authenticated.ID)
		if err != nil {
			return err
		}
		if player.Email == nil || player.TokenVersion != authenticated.TokenVersion {
			return ErrInvalidCredentials
		}
		credentials, err = s.createLoginSession(tx, player)
		return err
	})
	if err != nil {
		return SessionCredentials{}, err
	}
	return credentials, nil
}

func (s *AuthService) createLoginSession(tx *gorm.DB, player models.Player) (SessionCredentials, error) {
	if err := tx.Where("player_id = ? AND (expires_at <= ? OR token_version <> ?)", player.ID, time.Now(), player.TokenVersion).
		Delete(&models.AuthSession{}).Error; err != nil {
		return SessionCredentials{}, err
	}
	return s.createSession(tx, player)
}

// SignupSession commits the player, membership and session together.
func (s *AuthService) SignupSession(name, email, password, inviteCode string) (uuid.UUID, SessionCredentials, error) {
	var credentials SessionCredentials
	playerID, err := s.signupNewPlayer(name, email, password, inviteCode, func(tx *gorm.DB, player models.Player) error {
		var err error
		credentials, err = s.createSession(tx, player)
		return err
	})
	if err != nil {
		return uuid.Nil, SessionCredentials{}, err
	}
	return playerID, credentials, nil
}

func (s *AuthService) createSession(tx *gorm.DB, player models.Player) (SessionCredentials, error) {
	proof, hash, err := generateResetToken()
	if err != nil {
		return SessionCredentials{}, err
	}
	now := time.Now()
	session := models.AuthSession{
		PlayerID: player.ID, TokenVersion: player.TokenVersion, RefreshHash: hash, ExpiresAt: now.Add(sessionTTL),
	}
	if err := tx.Create(&session).Error; err != nil {
		return SessionCredentials{}, err
	}
	return s.sessionCredentials(session, proof, now)
}

func (s *AuthService) RefreshSession(cookie string) (SessionCredentials, error) {
	var credentials SessionCredentials
	err := s.withRefreshSession(cookie, func(tx *gorm.DB, session models.AuthSession, hash string) error {
		now := time.Now()
		if !session.ExpiresAt.After(now) {
			return ErrInvalidToken
		}
		if !refreshHashMatches(session.RefreshHash, hash) {
			if isRecentRefresh(session, hash, now) {
				return ErrRefreshRotated
			}
			return ErrInvalidToken
		}
		var player models.Player
		err := tx.Select("token_version").Take(&player, "id = ?", session.PlayerID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && player.TokenVersion != session.TokenVersion) {
			return ErrInvalidToken
		}
		if err != nil {
			return err
		}
		proof, nextHash, err := generateResetToken()
		if err != nil {
			return err
		}
		if err := tx.Model(&session).Updates(map[string]interface{}{
			"previous_hash": session.RefreshHash, "refresh_hash": nextHash, "rotated_at": now,
		}).Error; err != nil {
			return err
		}
		credentials, err = s.sessionCredentials(session, proof, now)
		return err
	})
	if err != nil {
		return SessionCredentials{}, err
	}
	return credentials, nil
}

func (s *AuthService) EndSession(cookie string) error {
	return s.withRefreshSession(cookie, func(tx *gorm.DB, session models.AuthSession, hash string) error {
		if !refreshHashMatches(session.RefreshHash, hash) && !isRecentRefresh(session, hash, time.Now()) {
			return ErrInvalidToken
		}
		return tx.Delete(&session).Error
	})
}

func (s *AuthService) withRefreshSession(cookie string, use func(*gorm.DB, models.AuthSession, string) error) error {
	rawID, proof, found := strings.Cut(cookie, ".")
	id, err := uuid.Parse(rawID)
	if !found || err != nil || len(proof) != base64.RawURLEncoding.EncodedLen(resetTokenBytes) {
		return ErrInvalidToken
	}
	hash := hashResetToken(proof)
	return s.DB.Transaction(func(tx *gorm.DB) error {
		var session models.AuthSession
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Take(&session, "id = ?", id).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrInvalidToken
		}
		if err != nil {
			return err
		}
		return use(tx, session, hash)
	})
}

func refreshHashMatches(stored, presented string) bool {
	return subtle.ConstantTimeCompare([]byte(stored), []byte(presented)) == 1
}

func isRecentRefresh(session models.AuthSession, hash string, now time.Time) bool {
	return session.RotatedAt.Add(refreshOverlap).After(now) && refreshHashMatches(session.PreviousHash, hash)
}

func (s *AuthService) sessionCredentials(session models.AuthSession, proof string, now time.Time) (SessionCredentials, error) {
	expires := now.Add(accessTokenTTL)
	if session.ExpiresAt.Before(expires) {
		expires = session.ExpiresAt
	}
	token, err := s.signAccessToken(session.PlayerID, session.TokenVersion, &session.ID, now, expires)
	if err != nil {
		return SessionCredentials{}, err
	}
	return SessionCredentials{
		AccessToken: token, RefreshToken: session.ID.String() + "." + proof, ExpiresAt: session.ExpiresAt,
	}, nil
}
