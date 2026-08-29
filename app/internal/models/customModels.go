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
	Name    string         `json:"Name"`
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
	TeamName     string
	TeamColour   string
	Score        int
	PlayerID     uuid.UUID
	PlayerName   string
	GoalsScored  int
}

// GroupWithRole is a Group tagged with the role the *requesting* player holds
// in it — the shape GET /groups/me returns, so a client can tell which of its
// groups it may act as an admin in without a follow-up request per group. It
// embeds Group rather than restating its fields (same pattern as
// PlayerGroupStanding embedding PointsStandingRow), so the JSON stays flat and
// InviteCode keeps its json:"-": listing your groups still never leaks a code.
type GroupWithRole struct {
	Group
	Role string `json:"role"`
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

// PlayerGroupStanding is one player's standings row inside a single group,
// tagged with the group it belongs to. The cross-group player profile needs
// the same shape as PointsStandingRow repeated once per group, so it embeds
// the row rather than restating its eight fields (the JSON stays flat).
type PlayerGroupStanding struct {
	PointsStandingRow
	GroupID   uuid.UUID `json:"GroupID"`
	GroupName string    `json:"GroupName"`
}

// PlayerProfileStats is one player's record across every group they belong
// to: Overall counts all of those groups' matches together, PerGroup breaks
// the same period down group by group.
type PlayerProfileStats struct {
	Overall  PointsStandingRow     `json:"Overall"`
	PerGroup []PlayerGroupStanding `json:"PerGroup"`
}
