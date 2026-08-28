package models

import (
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

// Player représente un joueur.
type Player struct {
	BaseModel
	Name             string        `gorm:"type:string;uniqueIndex" json:"name"`
	TeamCompositions []MatchPlayer `gorm:"foreignKey:PlayerID"`
}

// Group représente un groupe de joueurs (ex: un club, une salle). Team et
// Match sont scopés par groupe ; un Player peut appartenir à plusieurs
// groupes via GroupMembership.
type Group struct {
	BaseModel
	Name string `gorm:"type:string" json:"name"`
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
// Group : un joueur peut appartenir à plusieurs groupes.
type GroupMembership struct {
	BaseModel
	GroupID  uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_group_membership_group_player" json:"group_id"`
	PlayerID uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_group_membership_group_player" json:"player_id"`
}
