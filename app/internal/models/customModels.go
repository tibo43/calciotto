package models

import (
	"time"

	"github.com/google/uuid"
)

// Player représente un joueur.
//
// OwnGoals is credited to this player but counts toward the OTHER team's
// Score, kept separate from GoalsScored so it never inflates this player's
// personal scorer/points tallies (see ComputeScorers/ComputePointsStandings,
// neither of which reads OwnGoals).
type PlayerCustom struct {
	ID          uuid.UUID `json:"ID"`
	Name        string    `json:"Name"`
	GoalsScored int       `json:"GoalNumber"`
	OwnGoals    int       `json:"OwnGoals"`
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
//
// CreatedAt is not omitempty, unlike the scheduling fields: every match, past
// or future, has one, so there is no "unscheduled match" case to keep byte-
// identical here the way there is for ScheduledAt et al. It exists on this DTO
// so the frontend can compute the Man of the Match voting window's own anchor
// for a match with no kick-off at all (see matchRegistration.js's
// votingWindowError/MatchVoteService.VotingWindowError on the Go side) without
// a second request.
type MatchWithDetails struct {
	ID                    uuid.UUID         `json:"ID"`
	GroupID               uuid.UUID         `json:"GroupID"`
	Date                  Date              `json:"Date"`
	CreatedAt             time.Time         `json:"CreatedAt"`
	ScheduledAt           *time.Time        `json:"ScheduledAt,omitempty"`
	RegistrationOpensAt   *time.Time        `json:"RegistrationOpensAt,omitempty"`
	RegistrationsClosedAt *time.Time        `json:"RegistrationsClosedAt,omitempty"`
	MaxPlayers            *int              `json:"MaxPlayers,omitempty"`
	ResultRecordedAt      *time.Time        `json:"ResultRecordedAt,omitempty"`
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
	MatchCreatedAt             time.Time
	MatchScheduledAt           *time.Time
	MatchRegistrationOpensAt   *time.Time
	MatchRegistrationsClosedAt *time.Time
	MatchMaxPlayers            *int
	MatchResultRecordedAt      *time.Time
	TeamID                     uuid.UUID
	TeamName                   string
	TeamColour                 string
	Score                      int
	PlayerID                   uuid.UUID
	PlayerName                 string
	GoalsScored                int
	OwnGoals                   int
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

// MatchVoteTally is one candidate's line in a match's Man of the Match tally,
// part of MatchVoteSummary below. PascalCase JSON, matching the other
// match-scoped DTOs (MatchRegistrationEntry) rather than the group/player
// DTOs' lowercase convention — see CLAUDE.md's "API payload field casing"
// note.
type MatchVoteTally struct {
	PlayerID uuid.UUID `json:"PlayerID"`
	Name     string    `json:"Name"`
	Votes    int       `json:"Votes"`
}

// MatchVoteSummary is GET /matches/:id/votes' whole response: the tally
// (every candidate with at least one vote, ordered by vote count desc then
// name asc) plus which player, if any, the caller has voted for.
//
// MyVoteFor is a *uuid.UUID with omitempty rather than a plain uuid.UUID, the
// same reasoning as MatchWithDetails.RegistrationCount: "the caller has not
// voted" (nil, key absent) has to stay distinguishable from a real player id,
// and uuid.Nil is a value a genuine vote could never hold anyway but a zero
// value would be an ambiguous way to say so.
type MatchVoteSummary struct {
	Tally     []MatchVoteTally `json:"Tally"`
	MyVoteFor *uuid.UUID       `json:"MyVoteFor,omitempty"`
}

// MatchOdds is the three-way betting odds for one match, derived fresh from
// both teams' pre-match Elo ratings (see ComputeMatchOdds) — never stored,
// the same "derive it, don't store it" choice this whole feature makes
// throughout. A multiplier of 2.0 means a correct 1-token prediction nets
// +1 (pays out 2, stake included); see MatchPrediction's own comment for why
// there is no stake field to multiply this against.
type MatchOdds struct {
	TeamAOdds float64 `json:"TeamAOdds"`
	DrawOdds  float64 `json:"DrawOdds"`
	TeamBOdds float64 `json:"TeamBOdds"`
}

// MatchPredictionEntry is one player's prediction line for a match, as
// returned by MatchPredictionService.ListPredictions — mirrors
// MatchRegistrationEntry's shape. PredictedTeamID carries no omitempty: nil
// here means "predicted a draw", which must always be visible on an entry
// that exists, never hidden the way an *absent* prediction would be (see
// MatchPredictionSummary.MyPrediction for that distinction).
type MatchPredictionEntry struct {
	PlayerID        uuid.UUID  `json:"PlayerID"`
	Name            string     `json:"Name"`
	PredictedTeamID *uuid.UUID `json:"PredictedTeamID"`
}

// MatchPredictionSummary is GET /matches/:id/predictions' whole response:
// every member's prediction, the odds they were/are made against, and the
// caller's own prediction.
//
// MyPrediction is a *MatchPredictionEntry, not a bare *uuid.UUID: a
// predicted draw is itself represented as PredictedTeamID: nil (see
// MatchPrediction's comment), so a bare pointer could never distinguish
// "hasn't predicted at all" from "predicted a draw" — both would be nil.
// Wrapping the whole entry resolves that the same way MatchWithDetails.
// RegistrationCount resolves "N/A" vs "zero": the outer pointer is what
// carries "absent", not the inner one.
//
// Odds is nil when the match's roster isn't fully composed on both sides yet
// — there is nothing to average a rating over, so nothing to show.
//
// Other members' picks stay private until kick-off: before it, Entries holds
// only the caller's own prediction (if any) and Revealed is false, so nobody
// can follow the crowd while the choice is still open. Count is always the
// total number of predictions made, which reveals no one's pick.
type MatchPredictionSummary struct {
	Entries      []MatchPredictionEntry `json:"Entries"`
	Count        int                    `json:"Count"`
	Revealed     bool                   `json:"Revealed"`
	Odds         *MatchOdds             `json:"Odds,omitempty"`
	MyPrediction *MatchPredictionEntry  `json:"MyPrediction,omitempty"`
}

// PredictionStandingRow is one player's row in the season-long prediction
// leaderboard. Points is a float, not an int, because a correct prediction
// nets its odds' value minus the one implicit token staked (an evens
// favourite nets barely above 0, a long-shot draw nets several) rather than
// a flat +1 — see MatchPrediction's own comment for why there is no actual
// balance anywhere, just this derived running total. IsMember is tagged the
// same post-processing way as PointsStandingRow/ScorerRow/MotmStandingRow.
//
// History is every settled prediction behind the row, oldest first — what
// the leaderboard's last-five form and per-player detail are drawn from.
type PredictionStandingRow struct {
	PlayerID    uuid.UUID                `json:"PlayerID"`
	Name        string                   `json:"Name"`
	Predictions int                      `json:"Predictions"`
	Correct     int                      `json:"Correct"`
	Points      float64                  `json:"Points"`
	IsMember    bool                     `json:"IsMember"`
	History     []PredictionHistoryEntry `json:"History"`
}

// PredictionHistoryEntry is one settled prediction: the match, its final
// score, what was picked (PredictedTeamID nil = draw) at which odds, and
// what it earned — odds - 1 on a hit, -1 on a miss.
type PredictionHistoryEntry struct {
	MatchID         uuid.UUID  `json:"MatchID"`
	Date            Date       `json:"Date"`
	TeamAID         uuid.UUID  `json:"TeamAID"`
	TeamAName       string     `json:"TeamAName"`
	TeamBID         uuid.UUID  `json:"TeamBID"`
	TeamBName       string     `json:"TeamBName"`
	ScoreA          int        `json:"ScoreA"`
	ScoreB          int        `json:"ScoreB"`
	PredictedTeamID *uuid.UUID `json:"PredictedTeamID"`
	Odds            float64    `json:"Odds"`
	Correct         bool       `json:"Correct"`
	Points          float64    `json:"Points"`
}

// MotmStandingRow is one player's row in the season-long Man of the Match
// leaderboard: how many matches' MOTM award they have won. A match tied
// between several players counts as an award for each of them (see
// ComputeMotmWinners), so Awards summed across a row is not the same as the
// number of matches played. IsMember is tagged the same post-processing way
// as PointsStandingRow/ScorerRow — see their comment.
type MotmStandingRow struct {
	PlayerID uuid.UUID `json:"PlayerID"`
	Name     string    `json:"Name"`
	Awards   int       `json:"Awards"`
	IsMember bool      `json:"IsMember"`
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
	// OwnGoals is this player's own-goal total across the matches counted
	// above — reported alongside GoalsFor rather than netted into it, since
	// an own goal already counts against the player's own team's result
	// (see MatchPlayer.OwnGoals) without ever inflating GoalsFor itself.
	OwnGoals int  `json:"OwnGoals"`
	Points   int  `json:"Points"`
	IsMember bool `json:"IsMember"`
}

// ScorerRow is one player's row in the top-scorers ranking. IsMember carries
// the same meaning and is applied the same way as on PointsStandingRow — see
// its comment above.
type ScorerRow struct {
	PlayerID uuid.UUID `json:"PlayerID"`
	Name     string    `json:"Name"`
	Played   int       `json:"Played"`
	Goals    int       `json:"Goals"`
	// OwnGoals mirrors PointsStandingRow.OwnGoals — see its comment. Reported
	// alongside Goals, never folded into it.
	OwnGoals int  `json:"OwnGoals"`
	IsMember bool `json:"IsMember"`
}

// PlayerGroupStanding is one player's standings row inside a single group,
// tagged with the group it belongs to. The cross-group player profile needs
// the same shape as PointsStandingRow repeated once per group, so it embeds
// the row rather than restating its eight fields (the JSON stays flat).
// MotmAwards rides alongside rather than living on PointsStandingRow itself,
// since that type is also GET /standings/points' own row shape and has no
// business carrying a Man of the Match count.
type PlayerGroupStanding struct {
	PointsStandingRow
	GroupID    uuid.UUID `json:"GroupID"`
	GroupName  string    `json:"GroupName"`
	MotmAwards int       `json:"MotmAwards"`
	// PredictionPoints is this player's settled prediction return inside
	// this one group (odds − 1 on a hit, −1 on a miss). It is computed per
	// group because the odds come from that group's own Elo, and it stays
	// 0 when the player has no settled prediction there.
	PredictionPoints float64 `json:"PredictionPoints"`
}

// PlayerOverallStanding is the profile's Overall row: the same shape as
// PointsStandingRow across every group combined, plus MotmAwards for the
// same reason PlayerGroupStanding carries its own copy rather than
// PointsStandingRow growing the field.
type PlayerOverallStanding struct {
	PointsStandingRow
	MotmAwards int `json:"MotmAwards"`
	// PredictionPoints is the sum of the per-group totals, not a second
	// settlement over every group mixed together: odds are a group's own
	// Elo, so concatenating histories would price the bets wrong.
	PredictionPoints float64 `json:"PredictionPoints"`
}

// PlayerProfileStats is one player's record across every group they belong
// to: Overall counts all of those groups' matches together, PerGroup breaks
// the same period down group by group.
type PlayerProfileStats struct {
	Overall  PlayerOverallStanding `json:"Overall"`
	PerGroup []PlayerGroupStanding `json:"PerGroup"`
	// Seasons is every season label across all of the player's groups,
	// newest-or-oldest sorted ascending, independent of the ?season= filter
	// so the profile's season select stays complete while a filter is on.
	// An empty slice (never null) means the player has no dated match yet.
	Seasons []string `json:"Seasons"`
}

// EloStandingRow is one player's row in a group's continuous, derive-fresh
// Elo rating table — see internal/services/elo.go's ComputeEloStandings and
// CLAUDE.md's "Elo rating" section for the full rules. Rating is a float64
// because the update itself is fractional (K * (actual - expected)); it's
// only rounded for display, by cmd/elo, never in computation. IsMember is
// tagged the same post-processing way as PointsStandingRow/ScorerRow/
// MotmStandingRow — see their comments. This DTO is not returned by any HTTP
// handler today: it exists purely for StandingsService.GetEloStandings,
// consumed only by cmd/elo — a deliberate, CLI-only scope, not an oversight.
type EloStandingRow struct {
	PlayerID    uuid.UUID `json:"PlayerID"`
	Name        string    `json:"Name"`
	Rating      float64   `json:"Rating"`
	GamesPlayed int       `json:"GamesPlayed"`
	IsMember    bool      `json:"IsMember"`
}
