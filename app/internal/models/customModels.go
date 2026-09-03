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
//
// This is the DTO the frontend consumes, which is why its JSON keys are
// PascalCase where Match's own are snake_case — the two are separate contracts
// and this one has been parsed by the client for months.
//
// The four scheduling fields mirror Match's (see its comment for why they are
// nullable and all-or-nothing) and exist here so a match card can tell a
// scheduled match from an ordinary one, and show its kick-off time, without a
// second request. Every one of them carries omitempty for a compatibility
// reason rather than a cosmetic one: an unscheduled match must serialize to
// byte-for-byte the same JSON it did before this feature existed, so a client
// that knows nothing about scheduling sees no new keys at all.
//
// RegistrationCount is how many players have signed up. It is a *int, not a
// plain int, precisely so that omitempty distinguishes "not a scheduled match,
// this question is meaningless" (nil, key absent) from "scheduled, nobody has
// signed up yet" (0, key present) — a plain int would collapse those two into
// the same omitted key, and "0 / 16 signed up" is a real thing to render. It is
// set iff the match is scheduled.
//
// There is deliberately no "am I registered" flag here: answering it would mean
// threading the requesting player's id through the whole read path, including
// the four StandingsService calls that have no caller identity to give, for
// something only the detail page needs — and GET /matches/:id/registrations
// already tells it.
type MatchWithDetails struct {
	ID                    uuid.UUID         `json:"ID"`
	GroupID               uuid.UUID         `json:"GroupID"`
	Date                  Date              `json:"Date"`
	ScheduledAt           *time.Time        `json:"ScheduledAt,omitempty"`
	RegistrationOpensAt   *time.Time        `json:"RegistrationOpensAt,omitempty"`
	RegistrationsClosedAt *time.Time        `json:"RegistrationsClosedAt,omitempty"`
	MaxPlayers            *int              `json:"MaxPlayers,omitempty"`
	RegistrationCount     *int              `json:"RegistrationCount,omitempty"`
	Teams                 []TeamWithPlayers `json:"Teams"`
}

// IsScheduled mirrors Match.IsScheduled on the read DTO, so the reconstruction
// loops in MatchService don't have to restate the "scheduled iff there is a
// kick-off time" rule inline.
func (m MatchWithDetails) IsScheduled() bool {
	return m.ScheduledAt != nil
}

// RowsMatchDetails is one flat row of the matches → match_players →
// teams/players join, before MatchService rebuilds the nested shape from it.
// The Match* fields repeat identically across every row of the same match.
//
// Note what is *not* here: the sign-up count. Adding match_registrations to
// that join would fan out a second one-to-many on top of the existing one and
// multiply the rows, silently corrupting every score and goal count derived
// from them — so it is fetched by a separate grouped count and merged in Go.
type RowsMatchDetails struct {
	MatchID                    uuid.UUID
	MatchGroupID               uuid.UUID
	MatchDate                  Date
	MatchScheduledAt           *time.Time
	MatchRegistrationOpensAt   *time.Time
	MatchRegistrationsClosedAt *time.Time
	MatchMaxPlayers            *int
	TeamID                     uuid.UUID
	TeamName                   string
	TeamColour                 string
	Score                      int
	PlayerID                   uuid.UUID
	PlayerName                 string
	GoalsScored                int
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
