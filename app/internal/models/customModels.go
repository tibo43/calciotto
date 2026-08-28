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
	ID      uuid.UUID         `json:"ID"`
	GroupID uuid.UUID         `json:"GroupID"`
	Date    Date              `json:"Date"`
	Teams   []TeamWithPlayers `json:"Teams"`
}

type RowsMatchDetails struct {
	MatchID      uuid.UUID
	MatchGroupID uuid.UUID
	MatchDate    Date
	TeamID       uuid.UUID
	TeamColour   string
	Score        int
	PlayerID     uuid.UUID
	PlayerName   string
	GoalsScored  int
}

// PointsStandingRow is one player's row in the win/draw/loss points standings.
type PointsStandingRow struct {
	PlayerID uuid.UUID `json:"PlayerID"`
	Name     string    `json:"Name"`
	Played   int       `json:"Played"`
	Won      int       `json:"Won"`
	Drawn    int       `json:"Drawn"`
	Lost     int       `json:"Lost"`
	GoalsFor int       `json:"GoalsFor"`
	Points   int       `json:"Points"`
}

// ScorerRow is one player's row in the top-scorers ranking.
type ScorerRow struct {
	PlayerID uuid.UUID `json:"PlayerID"`
	Name     string    `json:"Name"`
	Played   int       `json:"Played"`
	Goals    int       `json:"Goals"`
}
