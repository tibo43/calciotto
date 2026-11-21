package services

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"app/internal/models"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const passkeyTimeout = 5 * time.Minute

var (
	ErrPasskeyOrigin    = errors.New("passkeys are unavailable on this origin")
	ErrPasskeysDisabled = errors.New("passkeys are unavailable for this account")
	ErrPasskeyName      = errors.New("passkey names must be at most 60 characters")
	ErrPasskeyProof     = errors.New("could not verify the passkey")
)

type PasskeyService struct {
	auth  *AuthService
	sites map[string]*webauthn.WebAuthn
}

// NewPasskeyService binds each ceremony to one configured frontend origin.
// The RP ID is the frontend hostname, including for the same-origin auth proxy.
func NewPasskeyService(auth *AuthService, origins []string) (*PasskeyService, error) {
	sites := make(map[string]*webauthn.WebAuthn)
	for _, origin := range origins {
		site, err := url.Parse(origin)
		// A skipped origin would otherwise only show up as a 403 on every
		// passkey request from it, with nothing saying why.
		if err != nil || site.Hostname() == "" || origin != site.Scheme+"://"+site.Host {
			log.Printf("passkeys are not available on %q: an allowed origin must be a bare scheme://host[:port], with no path or trailing slash", origin)
			continue
		}
		host := site.Hostname()
		if (site.Scheme != "https" && !(site.Scheme == "http" && host == "localhost")) || protocol.ValidateRPID(host) != nil {
			log.Printf("passkeys are not available on %q: WebAuthn needs https (or http://localhost) and a domain name", origin)
			continue
		}
		timeout := webauthn.TimeoutConfig{Enforce: true, Timeout: passkeyTimeout}
		rp, err := webauthn.New(&webauthn.Config{
			RPID: host, RPDisplayName: "Agoran", RPOrigins: []string{origin},
			AttestationPreference: protocol.PreferNoAttestation,
			AuthenticatorSelection: protocol.AuthenticatorSelection{
				ResidentKey: protocol.ResidentKeyRequirementRequired, RequireResidentKey: new(true),
				UserVerification: protocol.VerificationRequired,
			},
			Timeouts: webauthn.TimeoutsConfig{Registration: timeout, Login: timeout},
		})
		if err != nil {
			return nil, fmt.Errorf("configure passkeys for %s: %w", origin, err)
		}
		sites[origin] = rp
	}
	return &PasskeyService{auth: auth, sites: sites}, nil
}

type passkeyUser struct {
	player      models.Player
	credentials []webauthn.Credential
}

func (u passkeyUser) WebAuthnID() []byte                         { return u.player.ID[:] }
func (u passkeyUser) WebAuthnName() string                       { return *u.player.Email }
func (u passkeyUser) WebAuthnDisplayName() string                { return u.player.Name }
func (u passkeyUser) WebAuthnCredentials() []webauthn.Credential { return u.credentials }

func requirePasskeyFeatures(db *gorm.DB, playerID uuid.UUID) error {
	var features []string
	// In a write transaction, keep the flag in place until the write commits.
	// design-v2 used to be required too; it is launched now.
	if err := db.Model(&models.PlayerFeature{}).Clauses(clause.Locking{Strength: "SHARE"}).
		Where("player_id = ? AND feature = ?", playerID, models.FeaturePasskeys).
		Pluck("feature", &features).Error; err != nil {
		return err
	}
	if len(features) != 1 {
		return ErrPasskeysDisabled
	}
	return nil
}

func (s *PasskeyService) List(playerID uuid.UUID) ([]models.Passkey, error) {
	if err := requirePasskeyFeatures(s.auth.DB, playerID); err != nil {
		return nil, err
	}
	keys := []models.Passkey{}
	err := s.auth.DB.Select("id", "name", "created_at", "last_used_at").
		Where("player_id = ?", playerID).Order("created_at DESC").Find(&keys).Error
	return keys, err
}

func (s *PasskeyService) Remove(playerID, keyID uuid.UUID) error {
	return s.auth.DB.Transaction(func(tx *gorm.DB) error {
		if _, err := lockPlayer(tx, playerID); err != nil {
			return err
		}
		if err := requirePasskeyFeatures(tx, playerID); err != nil {
			return err
		}
		return tx.Where("player_id = ? AND id = ?", playerID, keyID).Delete(&models.Passkey{}).Error
	})
}

func (s *PasskeyService) BeginRegistration(playerID uuid.UUID, origin, password, name string) (uuid.UUID, *protocol.CredentialCreation, error) {
	rp := s.sites[origin]
	if rp == nil {
		return uuid.Nil, nil, ErrPasskeyOrigin
	}
	if err := requirePasskeyFeatures(s.auth.DB, playerID); err != nil {
		return uuid.Nil, nil, err
	}
	name = strings.TrimSpace(name)
	if utf8.RuneCountInString(name) > 60 {
		return uuid.Nil, nil, ErrPasskeyName
	}
	if name == "" {
		name = "Passkey"
	}
	var player models.Player
	err := s.auth.DB.Take(&player, "id = ?", playerID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && (player.Email == nil || !validPassword(player, password))) {
		return uuid.Nil, nil, ErrInvalidCredentials
	}
	if err != nil {
		return uuid.Nil, nil, err
	}
	var keys []models.Passkey
	if err := s.auth.DB.Select("credential").Where("player_id = ? AND rp_id = ?", playerID, rp.Config.RPID).Find(&keys).Error; err != nil {
		return uuid.Nil, nil, err
	}
	var excluded []protocol.CredentialDescriptor
	for _, key := range keys {
		excluded = append(excluded, key.Credential.Descriptor())
	}
	options, session, err := rp.BeginRegistration(passkeyUser{player: player}, webauthn.WithExclusions(excluded))
	if err != nil {
		return uuid.Nil, nil, err
	}
	id, err := s.saveChallenge(models.PasskeyChallenge{
		PlayerID: &playerID, Origin: origin, TokenVersion: player.TokenVersion,
		Name: name, Session: *session, ExpiresAt: session.Expires,
	})
	return id, options, err
}

func (s *PasskeyService) FinishRegistration(playerID uuid.UUID, origin string, challengeID uuid.UUID, proof []byte) (models.Passkey, error) {
	rp := s.sites[origin]
	if rp == nil {
		return models.Passkey{}, ErrPasskeyOrigin
	}
	if err := requirePasskeyFeatures(s.auth.DB, playerID); err != nil {
		return models.Passkey{}, err
	}
	challenge, err := s.takeChallenge(challengeID, origin, &playerID)
	if err != nil {
		return models.Passkey{}, err
	}
	parsed, err := protocol.ParseCredentialCreationResponseBytes(proof)
	if err != nil {
		return models.Passkey{}, ErrPasskeyProof
	}
	var key models.Passkey
	err = s.auth.DB.Transaction(func(tx *gorm.DB) error {
		player, err := lockPlayer(tx, playerID)
		if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && (player.Email == nil || player.TokenVersion != challenge.TokenVersion)) {
			return ErrPasskeyProof
		}
		if err != nil {
			return err
		}
		if err := requirePasskeyFeatures(tx, playerID); err != nil {
			return err
		}
		credential, err := rp.CreateCredential(passkeyUser{player: player}, challenge.Session, parsed)
		if err != nil {
			return ErrPasskeyProof
		}
		key = models.Passkey{
			PlayerID: playerID, RPID: rp.Config.RPID, CredentialID: credential.ID,
			Credential: *credential, Name: challenge.Name,
		}
		created := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&key)
		if created.Error == nil && created.RowsAffected != 1 {
			return ErrPasskeyProof
		}
		return created.Error
	})
	return key, err
}

func (s *PasskeyService) BeginLogin(origin string) (uuid.UUID, *protocol.CredentialAssertion, error) {
	rp := s.sites[origin]
	if rp == nil {
		return uuid.Nil, nil, ErrPasskeyOrigin
	}
	options, session, err := rp.BeginDiscoverableLogin()
	if err != nil {
		return uuid.Nil, nil, err
	}
	id, err := s.saveChallenge(models.PasskeyChallenge{Origin: origin, Session: *session, ExpiresAt: session.Expires})
	return id, options, err
}

func (s *PasskeyService) FinishLogin(origin string, challengeID uuid.UUID, proof []byte) (SessionCredentials, error) {
	rp := s.sites[origin]
	if rp == nil {
		return SessionCredentials{}, ErrPasskeyOrigin
	}
	challenge, err := s.takeChallenge(challengeID, origin, nil)
	if err != nil {
		return SessionCredentials{}, err
	}
	parsed, err := protocol.ParseCredentialRequestResponseBytes(proof)
	if err != nil {
		return SessionCredentials{}, ErrPasskeyProof
	}
	playerID, err := uuid.FromBytes(parsed.Response.UserHandle)
	if err != nil {
		return SessionCredentials{}, ErrPasskeyProof
	}
	var session SessionCredentials
	err = s.auth.DB.Transaction(func(tx *gorm.DB) error {
		player, err := lockPlayer(tx, playerID)
		if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && player.Email == nil) {
			return ErrPasskeyProof
		}
		if err != nil {
			return err
		}
		if err := requirePasskeyFeatures(tx, playerID); err != nil {
			if errors.Is(err, ErrPasskeysDisabled) {
				return ErrPasskeyProof
			}
			return err
		}
		var key models.Passkey
		err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Take(&key, "player_id = ? AND rp_id = ? AND credential_id = ?", playerID, rp.Config.RPID, parsed.RawID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPasskeyProof
		}
		if err != nil {
			return err
		}
		user := func(_, _ []byte) (webauthn.User, error) {
			return passkeyUser{player: player, credentials: []webauthn.Credential{key.Credential}}, nil
		}
		_, credential, err := rp.ValidatePasskeyLogin(user, challenge.Session, parsed)
		if err != nil || credential.Authenticator.CloneWarning {
			return ErrPasskeyProof
		}
		now := time.Now()
		key.Credential = *credential
		key.LastUsedAt = &now
		if err := tx.Model(&key).Select("Credential", "LastUsedAt").Updates(&key).Error; err != nil {
			return err
		}
		session, err = s.auth.createLoginSession(tx, player)
		return err
	})
	if err != nil {
		return SessionCredentials{}, err
	}
	return session, nil
}

func (s *PasskeyService) saveChallenge(challenge models.PasskeyChallenge) (uuid.UUID, error) {
	if err := s.auth.DB.Where("expires_at <= ?", time.Now()).Delete(&models.PasskeyChallenge{}).Error; err != nil {
		return uuid.Nil, err
	}
	err := s.auth.DB.Create(&challenge).Error
	return challenge.ID, err
}

// Consume the challenge even when verification fails. The delete returns at
// most one row to concurrent requests, across every backend instance.
func (s *PasskeyService) takeChallenge(id uuid.UUID, origin string, playerID *uuid.UUID) (models.PasskeyChallenge, error) {
	var challenge models.PasskeyChallenge
	query := s.auth.DB.Clauses(clause.Returning{}).Where("id = ? AND origin = ? AND expires_at > ?", id, origin, time.Now())
	if playerID == nil {
		query = query.Where("player_id IS NULL")
	} else {
		query = query.Where("player_id = ?", *playerID)
	}
	result := query.Delete(&challenge)
	if result.Error == nil && result.RowsAffected != 1 {
		return challenge, ErrPasskeyProof
	}
	return challenge, result.Error
}

func removePlayerPasskeys(tx *gorm.DB, playerID uuid.UUID) error {
	if err := tx.Where("player_id = ?", playerID).Delete(&models.PasskeyChallenge{}).Error; err != nil {
		return err
	}
	return tx.Where("player_id = ?", playerID).Delete(&models.Passkey{}).Error
}
