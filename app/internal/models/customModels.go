package models

import (
	"time"

	"github.com/google/uuid"
)

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

// MatchRegistrationEntry is one player's line in a scheduled match's sign-up
// list, as returned by MatchRegistrationService.ListRegistrations.
//
// Position and IsWaiting are *derived*, not stored: the list is ordered by
// RegisteredAt and the first Match.MaxPlayers entries are the confirmed
// roster, everything past it is the waiting list (see
// ComputeRegistrationPositions). Position is 1-based so it can be displayed
// as-is, and IsWaiting is exposed rather than left for the frontend to
// recompute from MaxPlayers — the rule of what counts as "waiting" belongs on
// this side, and the client would need MaxPlayers threaded into every view to
// derive it.
//
// RegisteredAt is the ordering key itself, kept in the payload so a client can
// show when someone signed up without a second query.
type MatchRegistrationEntry struct {
	PlayerID     uuid.UUID `json:"PlayerID"`
	Name         string    `json:"Name"`
	Position     int       `json:"Position"`
	IsWaiting    bool      `json:"IsWaiting"`
	RegisteredAt time.Time `json:"RegisteredAt"`
}

// GroupWithRole is a Group tagged with the role the *requesting* player holds
// in it — the shape GET /groups/me returns, so a client can tell which of its
// groups it may act as an admin in without a follow-up request per group. It
// embeds Group rather than restating its fields (same pattern as
// PlayerGroupStanding embedding PointsStandingRow), so the JSON stays flat and
// InviteCode keeps its json:"-": listing your groups still never leaks a code.
type GroupWithRole struct {
	Group
	Role       string `json:"role"`
	IsFavorite bool   `json:"is_favorite"`
}

// PlayerWithRole is a Player tagged with the role that player holds in the
// group being listed — the shape GET /groups/:id/players returns. It embeds
// Player rather than restating its fields (same "embed the base type, add a
// Role field" pattern as GroupWithRole), which is why its JSON keeps Player's
// own lowercase convention (id, name, email) with role riding alongside as a
// plain lowercase field — a different casing convention than GroupWithRole,
// which is fine since each embeds a different base type.
type PlayerWithRole struct {
	Player
	Role string `json:"role"`
}

// PointsStandingRow is one player's row in the win/draw/loss points standings.
// IsMember is a post-processing tag applied by StandingsService.GetPointsStandings
// (ComputePointsStandings itself knows nothing about current membership,
// staying a pure function of already-loaded match data): it's true when
// PlayerID still belongs to the group the standings were requested for, false
// when they've since been removed. A departed player's historical points/goals
// still appear here — standings are derived from match history, not current
// membership — but the frontend uses this flag to label the row as
// belonging to someone who's left the group.
type PointsStandingRow struct {
	PlayerID uuid.UUID `json:"PlayerID"`
	Name     string    `json:"Name"`
	Played   int       `json:"Played"`
	Won      int       `json:"Won"`
	Drawn    int       `json:"Drawn"`
	Lost     int       `json:"Lost"`
	GoalsFor int       `json:"GoalsFor"`
	Points   int       `json:"Points"`
	IsMember bool      `json:"IsMember"`
}

// ScorerRow is one player's row in the top-scorers ranking. IsMember carries
// the same meaning and is applied the same way as on PointsStandingRow — see
// its comment above.
type ScorerRow struct {
	PlayerID uuid.UUID `json:"PlayerID"`
	Name     string    `json:"Name"`
	Played   int       `json:"Played"`
	Goals    int       `json:"Goals"`
	IsMember bool      `json:"IsMember"`
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
