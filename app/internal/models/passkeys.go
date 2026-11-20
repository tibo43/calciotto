package models

import (
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

// Passkey holds the server-side public credential and its verification state.
type Passkey struct {
	BaseModel
	PlayerID     uuid.UUID           `gorm:"type:uuid;not null;index" json:"-"`
	Player       Player              `gorm:"foreignKey:PlayerID;constraint:OnDelete:CASCADE" json:"-"`
	RPID         string              `gorm:"type:text;not null;uniqueIndex:passkey_credential" json:"-"`
	CredentialID []byte              `gorm:"type:bytea;not null;uniqueIndex:passkey_credential" json:"-"`
	Credential   webauthn.Credential `gorm:"serializer:json;type:jsonb;not null" json:"-"`
	Name         string              `gorm:"type:text;not null" json:"name"`
	CreatedAt    time.Time           `json:"created_at"`
	LastUsedAt   *time.Time          `json:"last_used_at"`
}

// PasskeyChallenge is shared across backend instances. A player ID is set
// after password confirmation for enrollment and absent for sign-in.
type PasskeyChallenge struct {
	BaseModel
	PlayerID     *uuid.UUID           `gorm:"type:uuid;index"`
	Origin       string               `gorm:"type:text;not null"`
	TokenVersion int                  `gorm:"not null"`
	Name         string               `gorm:"type:text;not null"`
	Session      webauthn.SessionData `gorm:"serializer:json;type:jsonb;not null"`
	ExpiresAt    time.Time            `gorm:"not null;index"`
}
