package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"app/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrEmailRequired    = errors.New("email must not be empty")
	ErrPasswordRequired = errors.New("password must not be empty")
	// ErrPasswordTooShort rejects a password shorter than
	// minPasswordLength when one is being *chosen* — signing up, or resetting
	// a forgotten one. It is deliberately separate from
	// ErrPasswordRequired rather than folded into it: "you left it blank" and
	// "it needs to be longer" are different things to tell a caller, and the
	// second one has to name the length to be actionable.
	// The message is built from minPasswordLength rather than restating "8",
	// so raising the floor can never leave the wording behind.
	ErrPasswordTooShort     = fmt.Errorf("password must be at least %d characters long", minPasswordLength)
	ErrPlayerNotFound       = errors.New("player not found")
	ErrPlayerAlreadyClaimed = errors.New("player already has an account")
	ErrEmailAlreadyUsed     = errors.New("email already in use")
	ErrInvalidCredentials   = errors.New("invalid email or password")
	ErrInvalidToken         = errors.New("invalid or expired token")
	// ErrInvalidResetToken couvre volontairement *tous* les échecs de reset —
	// token inconnu, expiré, déjà consommé — sans les distinguer, exactement
	// comme ErrInvalidCredentials ne distingue pas "email inconnu" de "mauvais
	// mot de passe".
	ErrInvalidResetToken        = errors.New("invalid or expired reset token")
	ErrPasswordResetUnavailable = errors.New("password reset is temporarily unavailable")
	// ErrInviteCodeRequired is SignupNewPlayer's rejection when no invite code
	// is supplied at all: signing up without joining a group is no longer
	// allowed — every player must land in a group the moment they sign up, per
	// a deliberate business decision (self-service group creation/joining is
	// itself disabled, see GroupHandler.CreateGroup/JoinGroup).
	ErrInviteCodeRequired = errors.New("invite code is required")
)

const tokenTTL = 7 * 24 * time.Hour

// minPasswordLength is the floor for a password the caller is *choosing*.
// Login also limits attempts by IP and by account.
const minPasswordLength = 8

// validateNewPassword is the whole policy, in the one place both paths that
// let a caller choose a password go through (SignupNewPlayer and
// ResetPassword). Length is counted in *runes*, not bytes: a passphrase in a
// non-ASCII script would otherwise clear the bar on byte count alone while
// being shorter than it looks.
//
// Login is deliberately not a caller — it must only ever compare a hash, never
// judge the format of what it was given, or tightening this policy later would
// lock out every account whose password predates it. AuthService.Signup
// (attach credentials to an existing player) isn't one either: it is wired to
// no route and is only used by test/seed fixtures, so a policy there would buy
// nothing and break setup code that has no user behind it. DeleteAccount is
// the same case as Login — the password it takes is a confirmation of an
// existing one, not a new choice.
func validateNewPassword(password string) error {
	if password == "" {
		return ErrPasswordRequired
	}
	if utf8.RuneCountInString(password) < minPasswordLength {
		return ErrPasswordTooShort
	}
	return nil
}

const (
	// passwordResetTTL : durée de validité d'un lien de reset.
	passwordResetTTL = time.Hour
	// resetTokenBytes : entropie du token brut, avant encodage base64.
	resetTokenBytes = 32
	// defaultFrontendBaseURL sert de repli quand FRONTEND_BASE_URL n'est pas
	// dans l'environnement, dans le même esprit que le défaut d'API_BASE_URL
	// côté frontend.
	defaultFrontendBaseURL = "http://localhost:4000"
)

// playerClaims is the JWT payload used by AuthService — it carries the
// claimed Player's ID and, as of TokenVersion, the value of Player.TokenVersion
// at the moment this token was signed. ParseToken re-checks the latter against
// the database on every call, which is what makes revocation possible at all:
// identifying the caller still needs no DB round trip up front (the token
// itself carries player_id), but *trusting* the token now does.
type playerClaims struct {
	PlayerID     uuid.UUID  `json:"player_id"`
	TokenVersion int        `json:"token_version"`
	SessionID    *uuid.UUID `json:"session_id,omitempty"`
	jwt.RegisteredClaims
}

// AuthService covers the account lifecycle: SignupNewPlayer is the public
// signup flow (creates a new Player and attaches credentials in one step),
// Signup is a lower-level "attach credentials to an existing Player row" helper
// — not currently wired to any route; kept as a convenience for tests that need
// a player with real credentials without going through the full signup flow.
type AuthService struct {
	DB     *gorm.DB
	secret []byte
	// PasswordResetMailer delivers the reset link to its intended recipient.
	// Nil means recovery is unavailable for every account.
	PasswordResetMailer func(email, link string) error
}

// NewAuthService takes the signing secret as a parameter rather than reading
// JWT_SECRET itself, so callers (main.go) control the fail-fast behavior
// when the env var is missing, and tests can pass a fixed secret without
// depending on the environment.
func NewAuthService(db *gorm.DB, secret string) *AuthService {
	return &AuthService{DB: db, secret: []byte(secret), PasswordResetMailer: newPasswordResetMailer()}
}

// Signup attaches email/password credentials to an existing Player. It never
// creates a new Player — the caller must already know the Player's ID. Not
// wired to any HTTP route today (see SignupNewPlayer for the public
// POST /auth/signup flow).
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

// SignupNewPlayer is the public signup flow: it creates a brand-new Player
// and attaches email/password credentials to it in a single step, for
// someone who has never been added to a group before and has no existing
// Player row to claim.
//
// Unlike Signup, it never checks Player.Name for uniqueness — a display
// name may legitimately collide with another unrelated player's, and since
// a player only ever has one account name shared across every group they
// belong to, per-group uniqueness wouldn't make sense either. That's a
// deliberate product decision, not an oversight: do not add a duplicate-name
// check here.
//
// The account and its membership are created in one transaction after the
// invitation is rechecked under the group lock.
func (s *AuthService) SignupNewPlayer(name, email, password, inviteCode string) (uuid.UUID, error) {
	return s.signupNewPlayer(name, email, password, inviteCode, nil)
}

func (s *AuthService) signupNewPlayer(name, email, password, inviteCode string, complete func(*gorm.DB, models.Player) error) (uuid.UUID, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return uuid.Nil, ErrEmptyPlayerName
	}

	email = normalizeEmail(email)
	if email == "" {
		return uuid.Nil, ErrEmailRequired
	}
	if err := validateNewPassword(password); err != nil {
		return uuid.Nil, err
	}

	normalizedInviteCode := normalizeInviteCode(inviteCode)
	if normalizedInviteCode == "" {
		return uuid.Nil, ErrInviteCodeRequired
	}
	if _, err := groupForActiveInvite(s.DB, normalizedInviteCode); err != nil {
		return uuid.Nil, err
	}

	var existing models.Player
	result := s.DB.Where("email = ?", email).First(&existing)
	if result.Error == nil {
		return uuid.Nil, ErrEmailAlreadyUsed
	}
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return uuid.Nil, result.Error
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return uuid.Nil, err
	}

	player := models.Player{
		Name:         name,
		Email:        &email,
		PasswordHash: string(hash),
	}
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&player).Error; err != nil {
			return err
		}

		group, err := groupForActiveInvite(tx.Clauses(clause.Locking{Strength: "UPDATE"}), normalizedInviteCode)
		if err != nil {
			return err
		}
		if err := addGroupMembership(tx, group.ID, player.ID, models.RoleMember); err != nil {
			return err
		}
		if complete != nil {
			return complete(tx, player)
		}
		return nil
	})
	if err != nil {
		return uuid.Nil, err
	}
	return player.ID, nil
}

// Login checks email/password and returns a signed JWT on success. The error
// is intentionally the same sentinel whether the email is unknown or the
// password is wrong, so callers can't use it to enumerate registered emails.
func (s *AuthService) Login(email, password string) (string, error) {
	player, err := s.authenticate(email, password)
	if err != nil {
		return "", err
	}
	now := time.Now()
	return s.signAccessToken(player.ID, player.TokenVersion, nil, now, now.Add(tokenTTL))
}

func (s *AuthService) authenticate(email, password string) (models.Player, error) {
	var player models.Player
	err := s.DB.Where("email = ?", normalizeEmail(email)).First(&player).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.Player{}, ErrInvalidCredentials
	}
	if err != nil {
		return models.Player{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(player.PasswordHash), []byte(password)); err != nil {
		return models.Player{}, ErrInvalidCredentials
	}
	return player, nil
}

// ForgotPassword issues and emails a reset link for the registered account.
//
// An unknown email returns nil, not an error: answering differently would turn
// this endpoint into an email-enumeration oracle, the very thing Login avoids
// by returning one sentinel for both of its failure modes. For the same reason
// the unknown-email path still generates a token it throws away, so the two
// branches cost roughly the same time — a best effort, not a constant-time
// guarantee.
func (s *AuthService) ForgotPassword(email string) error {
	if s.PasswordResetMailer == nil {
		return ErrPasswordResetUnavailable
	}
	email = normalizeEmail(email)

	var player models.Player
	if err := s.DB.Where("email = ?", email).First(&player).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if _, _, genErr := generateResetToken(); genErr != nil {
			return genErr
		}
		return nil
	}

	rawToken, err := issuePasswordResetToken(s.DB, player.ID)
	if errors.Is(err, ErrPlayerNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	base := os.Getenv("FRONTEND_BASE_URL")
	if base == "" {
		base = defaultFrontendBaseURL
	}
	link := fmt.Sprintf("%s/reset-password?token=%s", strings.TrimSuffix(base, "/"), url.QueryEscape(rawToken))
	// Keep the same caller-visible response when delivery to a registered
	// account fails. Mail transport errors may contain the recipient or link.
	if err := s.PasswordResetMailer(email, link); err != nil {
		log.Print("[password reset] email delivery failed")
	}
	return nil
}

// issuePasswordResetToken persists a fresh single-use reset token for playerID
// and returns the raw value to put in the link — only its digest is stored.
func issuePasswordResetToken(db *gorm.DB, playerID uuid.UUID) (string, error) {
	rawToken, tokenHash, err := generateResetToken()
	if err != nil {
		return "", err
	}

	resetToken := models.PasswordResetToken{
		PlayerID:  playerID,
		TokenHash: tokenHash,
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		player, err := lockPlayer(tx, playerID)
		if err != nil {
			return err
		}
		if player.Email == nil {
			return ErrPlayerNotFound
		}
		resetToken.ExpiresAt = time.Now().Add(passwordResetTTL)
		return tx.Create(&resetToken).Error
	}); err != nil {
		return "", err
	}
	return rawToken, nil
}

func lockPlayer(tx *gorm.DB, playerID uuid.UUID) (models.Player, error) {
	var player models.Player
	err := tx.Clauses(clause.Locking{Strength: "NO KEY UPDATE"}).Take(&player, "id = ?", playerID).Error
	return player, err
}

// ResetPassword consumes a reset token and replaces the player's password.
//
// Every way this can fail on the token itself — unknown, expired, already used,
// or pointing at a player that no longer exists — collapses into
// ErrInvalidResetToken, so the caller learns nothing about which links exist.
// A successful reset also burns every other outstanding link for that player:
// requesting a second reset email must not leave the first one usable.
//
// It also bumps TokenVersion, which is the actual point of resetting a
// password after a suspected compromise: without this, every JWT already
// issued to this player — including one an attacker stole — would stay valid
// for the rest of its 7-day life regardless of the password changing under
// it. The increment is expressed as a SQL expression (token_version + 1)
// rather than read-then-write, so two concurrent resets can't race and lose
// one of the bumps.
func (s *AuthService) ResetPassword(token, newPassword string) error {
	if err := validateNewPassword(newPassword); err != nil {
		return err
	}

	var resetToken models.PasswordResetToken
	if err := s.DB.Where("token_hash = ?", hashResetToken(token)).First(&resetToken).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrInvalidResetToken
		}
		return err
	}

	if resetToken.UsedAt != nil || !resetToken.ExpiresAt.After(time.Now()) {
		return ErrInvalidResetToken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return s.DB.Transaction(func(tx *gorm.DB) error {
		player, err := lockPlayer(tx, resetToken.PlayerID)
		if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && player.Email == nil) {
			return ErrInvalidResetToken
		}
		if err != nil {
			return err
		}
		now := time.Now()
		consumed := tx.Model(&models.PasswordResetToken{}).
			Where("id = ? AND used_at IS NULL AND expires_at > ?", resetToken.ID, now).
			Update("used_at", now)
		if consumed.Error != nil {
			return consumed.Error
		}
		if consumed.RowsAffected != 1 {
			return ErrInvalidResetToken
		}
		if err := tx.Model(&models.Player{}).
			Where("id = ?", player.ID).
			Updates(map[string]interface{}{
				"password_hash": string(hash),
				"token_version": gorm.Expr("token_version + 1"),
			}).Error; err != nil {
			return err
		}
		// One statement marks the token just used *and* every other unused
		// token of that player — the already-expired ones are swept along
		// harmlessly, they were unusable anyway.
		return tx.Model(&models.PasswordResetToken{}).
			Where("player_id = ? AND used_at IS NULL", player.ID).
			Update("used_at", now).Error
	})
}

// deletedAccountName is what Player.Name is replaced with when the player
// deletes their own account (DeleteAccount) — the account is anonymized, not
// erased: MatchPlayer and MatchVote both hold a real foreign key to
// players(id) with no ON DELETE CASCADE (the same kind of FK
// MatchService.DeleteMatch already has to work around for matches(id) — see
// CLAUDE.md), so a genuine row delete would either fail outright or, if
// forced first, corrupt every other player's shared match history (goals
// scored, Man of the Match awards) in every match this player ever took part
// in. Stripping the name/email/password instead is what actually satisfies
// "this account no longer exists" without touching a single match record.
const deletedAccountName = "Deleted account"

// DeleteAccount lets a player permanently delete their own account (DELETE
// /players/me). The current password is required as a safety confirmation —
// this is the one truly irreversible self-service action in the whole app,
// unlike e.g. reopening sign-ups or demoting an admin, both of which can be
// undone — verified with the same bcrypt.CompareHashAndPassword Login uses.
// A wrong password and an unknown player both collapse into
// ErrInvalidCredentials, the same "don't tell the caller which check failed"
// posture Login already takes with its own two failure modes; in practice
// only "wrong password" is reachable through the route, since the player
// always comes from an already-validated JWT.
//
// The Player row itself is never deleted, only anonymized — see
// deletedAccountName's own comment for why a hard delete isn't an option.
// What *is* deleted outright, all inside one transaction:
//   - Every GroupMembership. If this player was a group's only admin and the
//     group still has other members, the oldest remaining member is promoted
//     first — the same successor rule GroupMembershipService.LeaveGroup
//     applies — except a group where this player was the *only* member is
//     deliberately left with none, rather than refused the way LeaveGroup's
//     own ErrLastMember would refuse a voluntary departure: there is no
//     "hand the group off" option left once the account itself is gone.
//   - Every PasswordResetToken (nothing left to reset a password on).
//   - Every MatchRegistration (future sign-ups this player will no longer
//     attend — safe to drop, unlike a MatchPlayer row, which is historical
//     record rather than a pending action).
//   - Every MatchVote *cast by* this player. Votes *for* this player as Man
//     of the Match are deliberately left alone: like a MatchPlayer's goals,
//     they're a fact about a match that already happened, not something tied
//     to the voter's account still existing.
//
// TokenVersion is also bumped, in the same statement as the anonymization
// update below — without it, a JWT issued before the deletion would still
// pass ParseToken (nothing about player_id, the only claim it checked before
// TokenVersion existed, becomes invalid just because the row was anonymized)
// and could keep reading/acting through routes that only need a valid token,
// for the rest of its 7-day life.
func (s *AuthService) DeleteAccount(playerID uuid.UUID, password string) error {
	if password == "" {
		return ErrPasswordRequired
	}

	var player models.Player
	if err := s.DB.First(&player, "id = ?", playerID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrInvalidCredentials
		}
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(player.PasswordHash), []byte(password)); err != nil {
		return ErrInvalidCredentials
	}

	return s.DB.Transaction(func(tx *gorm.DB) error {
		current, err := lockPlayer(tx, playerID)
		if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && current.PasswordHash != player.PasswordHash) {
			return ErrInvalidCredentials
		}
		if err != nil {
			return err
		}
		var groupIDs []uuid.UUID
		if err := tx.Model(&models.GroupMembership{}).Where("player_id = ?", playerID).
			Order("group_id ASC").Pluck("group_id", &groupIDs).Error; err != nil {
			return err
		}
		for _, groupID := range groupIDs {
			if err := lockGroup(tx, groupID); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}
		var memberships []models.GroupMembership
		if err := tx.Where("player_id = ?", playerID).Find(&memberships).Error; err != nil {
			return err
		}

		for _, membership := range memberships {
			if membership.Role != models.RoleAdmin {
				continue
			}

			var otherAdmins int64
			if err := tx.Model(&models.GroupMembership{}).
				Where("group_id = ? AND player_id <> ? AND role = ?", membership.GroupID, playerID, models.RoleAdmin).
				Count(&otherAdmins).Error; err != nil {
				return err
			}
			if otherAdmins > 0 {
				continue
			}

			var successor models.GroupMembership
			err := tx.Where("group_id = ? AND player_id <> ?", membership.GroupID, playerID).
				Order("created_at ASC").
				First(&successor).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					// An empty group has no administrator to manage an invitation.
					if err := tx.Model(&models.Group{}).Where("id = ?", membership.GroupID).
						Update("invite_expires_at", time.Now()).Error; err != nil {
						return err
					}
					continue
				}
				return err
			}
			if err := tx.Model(&models.GroupMembership{}).
				Where("id = ?", successor.ID).
				Update("role", models.RoleAdmin).Error; err != nil {
				return err
			}
		}

		if err := tx.Where("player_id = ?", playerID).Delete(&models.GroupMembership{}).Error; err != nil {
			return err
		}
		if err := tx.Where("player_id = ?", playerID).Delete(&models.PasswordResetToken{}).Error; err != nil {
			return err
		}
		if err := tx.Where("player_id = ?", playerID).Delete(&models.AuthSession{}).Error; err != nil {
			return err
		}
		if err := tx.Where("player_id = ?", playerID).Delete(&models.MatchRegistration{}).Error; err != nil {
			return err
		}
		if err := tx.Where("voter_id = ?", playerID).Delete(&models.MatchVote{}).Error; err != nil {
			return err
		}
		// Predictions are the caller's own guesses, like a vote cast — kept,
		// they would leave a "Deleted account" row on the prediction
		// leaderboard.
		if err := tx.Where("player_id = ?", playerID).Delete(&models.MatchPrediction{}).Error; err != nil {
			return err
		}
		// Hidden features enabled for this account: nothing left to show them to.
		if err := tx.Where("player_id = ?", playerID).Delete(&models.PlayerFeature{}).Error; err != nil {
			return err
		}
		// And the devices it received push notifications on, plus which of
		// those notifications the player had turned off.
		if err := tx.Where("player_id = ?", playerID).Delete(&models.PushSubscription{}).Error; err != nil {
			return err
		}
		if err := tx.Where("player_id = ?", playerID).Delete(&models.NotificationPreference{}).Error; err != nil {
			return err
		}

		return tx.Model(&models.Player{}).
			Where("id = ?", playerID).
			Updates(map[string]interface{}{
				"name":               deletedAccountName,
				"email":              nil,
				"password_hash":      "",
				"token_version":      gorm.Expr("token_version + 1"),
				"guide_completed_at": nil,
			}).Error
	})
}

// generateResetToken returns the raw token that goes into the emailed link and
// the SHA-256 hex digest that goes into the database — only the digest is ever
// persisted. The raw token is base64 URL-safe without padding so it can sit in
// a query string untouched.
func generateResetToken() (rawToken string, tokenHash string, err error) {
	buf := make([]byte, resetTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	rawToken = base64.RawURLEncoding.EncodeToString(buf)
	return rawToken, hashResetToken(rawToken), nil
}

func hashResetToken(rawToken string) string {
	digest := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(digest[:])
}

// ParseToken validates both the signed access token and its revocation state.
func (s *AuthService) ParseToken(tokenString string) (uuid.UUID, error) {
	claims := &playerClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(_ *jwt.Token) (interface{}, error) {
		return s.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil || !token.Valid || claims.PlayerID == uuid.Nil {
		return uuid.Nil, ErrInvalidToken
	}
	if claims.SessionID != nil {
		var session models.AuthSession
		err := s.DB.Select("auth_sessions.id").
			Joins("JOIN players ON players.id = auth_sessions.player_id AND players.token_version = auth_sessions.token_version").
			Where("auth_sessions.id = ? AND auth_sessions.player_id = ? AND auth_sessions.token_version = ? AND auth_sessions.expires_at > ?",
				*claims.SessionID, claims.PlayerID, claims.TokenVersion, time.Now()).
			Take(&session).Error
		if err != nil {
			return uuid.Nil, ErrInvalidToken
		}
		return claims.PlayerID, nil
	}
	// Previously issued access tokens keep their original expiry.
	var player models.Player
	if err := s.DB.Select("token_version").Take(&player, "id = ?", claims.PlayerID).Error; err != nil {
		return uuid.Nil, ErrInvalidToken
	}
	if player.TokenVersion != claims.TokenVersion {
		return uuid.Nil, ErrInvalidToken
	}
	return claims.PlayerID, nil
}

// GenerateToken issues a standalone token for callers using the legacy contract.
func (s *AuthService) GenerateToken(playerID uuid.UUID) (string, error) {
	var player models.Player
	if err := s.DB.Select("token_version").Take(&player, "id = ?", playerID).Error; err != nil {
		return "", err
	}
	now := time.Now()
	return s.signAccessToken(playerID, player.TokenVersion, nil, now, now.Add(tokenTTL))
}

func (s *AuthService) signAccessToken(playerID uuid.UUID, version int, sessionID *uuid.UUID, now, expiresAt time.Time) (string, error) {
	claims := playerClaims{
		PlayerID: playerID, TokenVersion: version, SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
