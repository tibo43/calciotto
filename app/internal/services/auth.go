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
	// ErrInvalidResetToken couvre volontairement *tous* les échecs de reset —
	// token inconnu, expiré, déjà consommé — sans les distinguer, exactement
	// comme ErrInvalidCredentials ne distingue pas "email inconnu" de "mauvais
	// mot de passe".
	ErrInvalidResetToken = errors.New("invalid or expired reset token")
)

const tokenTTL = 7 * 24 * time.Hour

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
// claimed Player's ID so the middleware can identify the caller without a
// DB round trip.
type playerClaims struct {
	PlayerID uuid.UUID `json:"player_id"`
	jwt.RegisteredClaims
}

// AuthService covers the account lifecycle: SignupNewPlayer is the public
// signup flow (creates a new Player and attaches credentials in one step),
// Signup is the "claim by name" flow for a Player row that already exists
// (created via PlayerService) but has no credentials yet — not currently
// wired to any route, kept for an upcoming "claim an existing ghost player"
// feature.
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
// creates a new Player — the caller must already know the Player's ID. Not
// wired to any HTTP route today (see SignupNewPlayer for the public
// POST /auth/signup flow); kept for an upcoming "claim an existing ghost
// player" feature that will reuse this.
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
// check here, and do not route this through PlayerService.CreatePlayer,
// which enforces exactly that check for a different flow (admin-created
// "ghost" players).
//
// inviteCode is optional — pass "" to sign up without joining any group,
// the same behavior as before this parameter existed. When non-empty (after
// normalizeInviteCode trims/uppercases it), it must resolve to an existing
// group or the whole signup fails with ErrInviteCodeNotFound: the player row
// and the group membership are created inside a single s.DB.Transaction, so
// a typo'd or stale code can never leave behind an orphaned account with no
// group, and a valid code can never silently fail to attach the new player
// to it. This intentionally does not reuse GroupService.JoinByInviteCode —
// that helper's IsMember check is meaningless for a player created moments
// earlier in the same transaction, and it queries s.DB rather than the
// transaction's tx, which would break atomicity. Instead the group lookup
// and services.NewGroupMembershipService(tx).AddPlayerToGroupWithRole(...)
// call are inlined here, both scoped to tx.
func (s *AuthService) SignupNewPlayer(name, email, password, inviteCode string) (uuid.UUID, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return uuid.Nil, ErrEmptyPlayerName
	}

	email = normalizeEmail(email)
	if email == "" {
		return uuid.Nil, ErrEmailRequired
	}
	if password == "" {
		return uuid.Nil, ErrPasswordRequired
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
	normalizedInviteCode := normalizeInviteCode(inviteCode)
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&player).Error; err != nil {
			return err
		}
		if normalizedInviteCode == "" {
			return nil
		}

		var group models.Group
		if err := tx.First(&group, "invite_code = ?", normalizedInviteCode).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInviteCodeNotFound
			}
			return err
		}
		return NewGroupMembershipService(tx).AddPlayerToGroupWithRole(group.ID, player.ID, models.RoleMember)
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

// ForgotPassword issues a single-use reset link for the account registered
// under email, and "sends" it (see sendPasswordResetLink — for now, a server
// log line).
//
// An unknown email returns nil, not an error: answering differently would turn
// this endpoint into an email-enumeration oracle, the very thing Login avoids
// by returning one sentinel for both of its failure modes. For the same reason
// the unknown-email path still generates a token it throws away, so the two
// branches cost roughly the same time — a best effort, not a constant-time
// guarantee.
func (s *AuthService) ForgotPassword(email string) error {
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
	if err != nil {
		return err
	}

	sendPasswordResetLink(email, rawToken)
	return nil
}

// InviteExistingPlayer attaches email to a "ghost" Player — one created by an
// admin via PlayerService.CreatePlayer, with a Name but no credentials — and
// issues them a link to set their own password, turning the roster entry into
// a real account. It is the counterpart of SignupNewPlayer for someone who is
// already on a group's roster: the Player row (and its match history) already
// exists, only the credentials are missing.
//
// It reuses the password-reset token machinery rather than inventing a
// claim-specific token type, because the two are the same operation seen from
// two angles: hand someone a single-use, short-lived, hashed-at-rest secret
// that authorizes setting Player.PasswordHash once. ResetPassword already
// overwrites PasswordHash unconditionally — it never asks whether one was
// there before — so it consumes an invite token exactly as it consumes a
// reset token, and the frontend needs no new page: the existing
// /reset-password?token=... link works verbatim.
//
// A player who already has an email is refused with ErrPlayerAlreadyClaimed:
// this path must not silently rewrite a live account's email (that would be a
// takeover vector, and "change my email" is an unrelated feature that doesn't
// exist). Authorizing *who* may invite whom is the caller's job — see
// GroupHandler.InvitePlayer, gated by RequireGroupAdminByPathParam plus an
// explicit IsMember check on the target.
func (s *AuthService) InviteExistingPlayer(playerID uuid.UUID, email string) error {
	email = normalizeEmail(email)
	if email == "" {
		return ErrEmailRequired
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

	// Attaching the email and issuing the token have to succeed or fail
	// together: a player left with an email but no outstanding invite could
	// neither log in nor be re-invited (the email is now set, so a retry would
	// hit ErrPlayerAlreadyClaimed) — a dead end worse than failing outright.
	var rawToken string
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Player{}).
			Where("id = ?", player.ID).
			Update("email", email).Error; err != nil {
			return err
		}
		var err error
		rawToken, err = issuePasswordResetToken(tx, player.ID)
		return err
	}); err != nil {
		return err
	}

	// Sent only once the transaction has committed — a link pointing at a
	// token that was rolled back would be worse than no link at all.
	sendPasswordResetLink(email, rawToken)
	return nil
}

// issuePasswordResetToken persists a fresh single-use reset token for playerID
// and returns the raw value to put in the link — only its digest is stored.
// Shared by ForgotPassword and InviteExistingPlayer, which differ solely in
// how they establish that the caller may set that player's password (knowing
// the email vs. being an admin of one of their groups). db is a parameter
// rather than s.DB so it can run inside a caller's transaction.
func issuePasswordResetToken(db *gorm.DB, playerID uuid.UUID) (string, error) {
	rawToken, tokenHash, err := generateResetToken()
	if err != nil {
		return "", err
	}

	resetToken := models.PasswordResetToken{
		PlayerID:  playerID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(passwordResetTTL),
	}
	if err := db.Create(&resetToken).Error; err != nil {
		return "", err
	}
	return rawToken, nil
}

// ResetPassword consumes a reset token and replaces the player's password.
//
// Every way this can fail on the token itself — unknown, expired, already used,
// or pointing at a player that no longer exists — collapses into
// ErrInvalidResetToken, so the caller learns nothing about which links exist.
// A successful reset also burns every other outstanding link for that player:
// requesting a second reset email must not leave the first one usable.
func (s *AuthService) ResetPassword(token, newPassword string) error {
	if newPassword == "" {
		return ErrPasswordRequired
	}

	var resetToken models.PasswordResetToken
	if err := s.DB.Where("token_hash = ?", hashResetToken(token)).First(&resetToken).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrInvalidResetToken
		}
		return err
	}

	now := time.Now()
	if resetToken.UsedAt != nil || now.After(resetToken.ExpiresAt) {
		return ErrInvalidResetToken
	}

	var player models.Player
	if err := s.DB.First(&player, "id = ?", resetToken.PlayerID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrInvalidResetToken
		}
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Player{}).
			Where("id = ?", player.ID).
			Update("password_hash", string(hash)).Error; err != nil {
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

// sendPasswordResetLink "delivers" the reset link. With no BREVO_API_KEY set
// it falls back to the original dev-only behavior — the link is only written
// to the server log, the same thing Rails and Django do with their
// development mailers — so nothing changes for local dev or CI. With
// BREVO_API_KEY set, it sends a real email through Brevo's transactional
// email API instead.
//
// Either way this is fire-and-forget from the caller's point of view: a
// failed send (including a quota breach — Brevo's free plan caps at 300
// emails/day) is only logged, never returned. ForgotPassword/
// InviteExistingPlayer have already committed the token to the DB by the
// time this runs, and neither must let mail delivery change what the caller
// sees — a player can always fall back to asking an admin to re-invite them,
// or requesting another reset, once delivery is actually working again.
func sendPasswordResetLink(email, rawToken string) {
	base := os.Getenv("FRONTEND_BASE_URL")
	if base == "" {
		base = defaultFrontendBaseURL
	}
	link := fmt.Sprintf("%s/reset-password?token=%s", strings.TrimSuffix(base, "/"), url.QueryEscape(rawToken))

	apiKey := os.Getenv("BREVO_API_KEY")
	if apiKey == "" {
		log.Printf("[password reset] would email %s: %s", email, link)
		return
	}

	subject := "Reset your Calciotto password"
	htmlContent := fmt.Sprintf(
		`<p>Click the link below to reset your Calciotto password:</p><p><a href="%s">%s</a></p><p>This link expires in one hour. If you didn't request this, you can ignore this email.</p>`,
		link, link,
	)
	if err := sendViaBrevo(apiKey, email, subject, htmlContent); err != nil {
		log.Printf("[password reset] failed to email %s via Brevo: %v", email, err)
	}
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
