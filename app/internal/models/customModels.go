package models

// Player représente un joueur.
type PlayerCustom struct {
	ID          string `json:"ID"`
	Name        string `json:"Name"`
	GoalsScored int    `json:"GoalNumber"`
}

// TeamWithPlayers représente une équipe avec ses joueurs.
type TeamWithPlayers struct {
	ID      string         `json:"ID"`
	Colour  string         `json:"Colour"`
	Score   int            `json:"Score"`
	Players []PlayerCustom `json:"Players"`
}

// MatchWithDetails représente un match avec ses équipes et joueurs.
type MatchWithDetails struct {
	ID    string            `json:"ID"`
	Date  string            `json:"Date"`
	Teams []TeamWithPlayers `json:"Teams"`
}

type RowsMatchDetails struct {
	MatchID     string
	MatchDate   string
	TeamID      string
	TeamColour  string
	Score       int
	PlayerID    string
	PlayerName  string
	GoalsScored int
}

type RowsTeamDetails struct {
	TeamID     string
	TeamColour string
}

type UserLogin struct {
	UserName string
	Password string
	IsAdmin  bool
}
