package models

import "github.com/google/uuid"

// Player représente un joueur.
type PlayerCustom struct {
	ID          uuid.UUID `json:"ID"`
	Name        string    `json:"Name"`
	GoalsScored int       `json:"GoalNumber"`
}

// TeamWithPlayers représente une équipe avec ses joueurs.
type TeamWithPlayers struct {
	ID      uuid.UUID      `json:"ID"`
	Colour  string         `json:"Colour"`
	Score   int            `json:"Score"`
	Players []PlayerCustom `json:"Players"`
}

// MatchWithDetails représente un match avec ses équipes et joueurs.
type MatchWithDetails struct {
	ID    uuid.UUID         `json:"ID"`
	Date  Date              `json:"Date"`
	Teams []TeamWithPlayers `json:"Teams"`
}

type RowsMatchDetails struct {
	MatchID     uuid.UUID
	MatchDate   Date
	TeamID      uuid.UUID
	TeamColour  string
	Score       int
	PlayerID    uuid.UUID
	PlayerName  string
	GoalsScored int
}
