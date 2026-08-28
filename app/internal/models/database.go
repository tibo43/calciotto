package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Modèle de base pour les champs communs
type BaseModel struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
}

// BeforeCreate est un hook GORM qui s'exécute avant la création d'un enregistrement
func (bm *BaseModel) BeforeCreate(tx *gorm.DB) error {
	bm.ID = uuid.New() // Génération d'un nouvel UUID
	return nil
}

// Player représente un joueur. Email/PasswordHash restent nuls tant que le
// joueur n'a pas "réclamé" son compte via AuthService.Signup — un Player
// peut donc exister sans jamais être associé à des identifiants de connexion.
type Player struct {
	BaseModel
	Name             string        `gorm:"type:string;uniqueIndex" json:"name"`
	Email            *string       `gorm:"type:string;uniqueIndex" json:"email,omitempty"`
	PasswordHash     string        `gorm:"type:string" json:"-"`
	TeamCompositions []MatchPlayer `gorm:"foreignKey:PlayerID"`
}

// PasswordResetToken matérialise un lien "mot de passe oublié" à usage unique,
// émis par AuthService.ForgotPassword et consommé par ResetPassword.
//
// TokenHash contient le SHA-256 (hex) du token brut, jamais le token lui-même :
// une fuite de cette table ne permet donc pas de forger un lien valide. Le hash
// est volontairement rapide, contrairement à Player.PasswordHash qui utilise
// bcrypt — un token de reset est généré aléatoirement (32 octets d'entropie),
// donc hors de portée d'une attaque par dictionnaire ou par force brute, là où
// un mot de passe choisi par un humain a besoin du coût délibéré de bcrypt.
// bcrypt n'apporterait rien ici et rendrait chaque vérification inutilement
// lente.
//
// UsedAt reste nil tant que le lien n'a pas servi : un token est valide s'il
// existe, qu'il n'est pas expiré (ExpiresAt) et qu'il n'a pas déjà été
// consommé.
type PasswordResetToken struct {
	BaseModel
	PlayerID  uuid.UUID  `gorm:"type:uuid;index" json:"player_id"`
	TokenHash string     `gorm:"type:string;uniqueIndex" json:"-"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
}

// Group représente un groupe de joueurs (ex: un club, une salle). Team et
// Match sont scopés par groupe ; un Player peut appartenir à plusieurs
// groupes via GroupMembership.
//
// InviteCode est le secret partagé qui permet à un joueur de rejoindre le
// groupe (POST /groups/join) : il porte donc `json:"-"` pour ne jamais
// s'échapper via les routes publiques GET /groups et GET /groups/:id — seul
// GET /groups/:id/invite-code, réservé aux membres, le renvoie explicitement.
type Group struct {
	BaseModel
	Name       string `gorm:"type:string" json:"name"`
	InviteCode string `gorm:"type:string;uniqueIndex" json:"-"`
}

// Team représente une équipe. Chaque groupe a exactement deux équipes
// (noir/blanc), créées automatiquement à la création du groupe — voir
// GroupService.CreateGroup.
type Team struct {
	BaseModel
	GroupID          uuid.UUID     `gorm:"type:uuid;index" json:"group_id"`
	Colour           string        `gorm:"type:string" json:"colour"`
	TeamCompositions []MatchPlayer `gorm:"foreignKey:TeamID"`
}

// Match représente un match, scopé à un groupe.
type Match struct {
	BaseModel
	GroupID          uuid.UUID     `gorm:"type:uuid;index" json:"group_id"`
	Date             Date          `gorm:"type:date" json:"date"`
	TeamCompositions []MatchPlayer `gorm:"foreignKey:MatchID"`
}

// MatchPlayer représente la composition d'une équipe pour un match.
type MatchPlayer struct {
	BaseModel
	MatchID     uuid.UUID `gorm:"type:uuid" json:"match_id"`
	TeamID      uuid.UUID `gorm:"type:uuid" json:"team_id"`
	PlayerID    uuid.UUID `gorm:"type:uuid" json:"player_id"`
	GoalsScored int       `gorm:"type:int" json:"goals_scored"`
}

// GroupMembership est la table de jointure many-to-many entre Player et
// Group : un joueur peut appartenir à plusieurs groupes. CreatedAt (auto-rempli
// par GORM à la création) permet de déterminer quel groupe un joueur a
// rejoint en premier — l'ID d'un groupe est un UUID aléatoire (v4), donc trié
// par ID ne dit rien sur l'ordre d'appartenance ou de création.
//
// Role distingue le créateur d'un groupe (RoleOwner) du reste des membres
// (RoleMember) — c'est une simple string plutôt qu'un type enum Go dédié,
// comme Team.Colour ci-dessus : le repo n'a pas de convention pour les enums
// et deux valeurs constantes suffisent ici. Le rôle n'a aujourd'hui aucun
// consommateur câblé sur une route (voir RequireGroupOwner dans
// internal/handlers/groupowner.go) ; il prépare seulement les futures
// features "quitter un groupe" / "retirer un membre".
type GroupMembership struct {
	BaseModel
	GroupID   uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_group_membership_group_player" json:"group_id"`
	PlayerID  uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_group_membership_group_player" json:"player_id"`
	Role      string    `gorm:"type:string;not null;default:member" json:"role"`
	CreatedAt time.Time `json:"-"`
}

// RoleOwner et RoleMember sont les deux seules valeurs valides de
// GroupMembership.Role. Le créateur d'un groupe (POST /groups) devient
// RoleOwner ; tout autre joueur ajouté (POST /groups/join, POST
// /groups/:id/players) devient RoleMember.
const (
	RoleOwner  = "owner"
	RoleMember = "member"
)
