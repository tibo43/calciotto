package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ModelWithUUID est une interface que tous les modèles avec UUID doivent implémenter
type ModelWithUUID interface {
	BeforeCreate(tx *gorm.DB) error
}

// Modèle de base pour les champs communs
type BaseModel struct {
	ID string `gorm:"type:uuid;primaryKey" json:"id"`
}

// BeforeCreate est un hook GORM qui s'exécute avant la création d'un enregistrement
func (bm *BaseModel) BeforeCreate(tx *gorm.DB) error {
	bm.ID = uuid.New().String() // Génération d'un nouvel UUID
	return nil
}

// Player représente un joueur.
type Player struct {
	BaseModel
	Name             string            `gorm:"type:string" json:"name"`
	TeamCompositions []TeamComposition `gorm:"foreignKey:PlayerID"`
	Goals            []Goal            `gorm:"foreignKey:PlayerID"`
}

// Team représente une équipe.
type Team struct {
	BaseModel
	Colour           string            `gorm:"type:string" json:"colour"`
	TeamCompositions []TeamComposition `gorm:"foreignKey:TeamID"`
	Goals            []Goal            `gorm:"foreignKey:TeamID"`
}

// Match représente un match.
type Match struct {
	BaseModel
	Date             string            `gorm:"type:string" json:"date"`
	TeamCompositions []TeamComposition `gorm:"foreignKey:MatchID"`
	Goals            []Goal            `gorm:"foreignKey:MatchID"`
}

// TeamComposition représente la composition d'une équipe pour un match.
type TeamComposition struct {
	BaseModel
	MatchID  string `gorm:"type:uuid" json:"match_id"`
	TeamID   string `gorm:"type:uuid" json:"team_id"`
	PlayerID string `gorm:"type:uuid" json:"player_id"`
}

// Goal représente un but marqué pendant un match.
type Goal struct {
	BaseModel
	MatchID  string `gorm:"type:uuid" json:"match_id"`
	TeamID   string `gorm:"type:uuid" json:"team_id"`
	PlayerID string `gorm:"type:uuid" json:"player_id"`
	Number   int    `gorm:"type:int" json:"number"`
}
