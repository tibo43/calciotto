package services

import (
	"errors"
	"time"

	"app/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	// ErrMatchNotScheduled is returned when a sign-up action targets a match
	// that was never scheduled (models.Match.IsScheduled is false). Such a
	// match is a plain record of a game, with no kick-off time and no
	// registration window — there is nothing to sign up for, so this is a
	// malformed request rather than a state conflict.
	ErrMatchNotScheduled = errors.New("this match is not open to registrations: it has no schedule")

	// ErrRegistrationsNotOpenYet is returned before Match.RegistrationOpensAt.
	// It is kept distinct from ErrRegistrationsClosed because the two are
	// genuinely different to the caller: "come back later" versus "too late".
	ErrRegistrationsNotOpenYet = errors.New("registrations for this match have not opened yet")

	// ErrRegistrationsClosed covers *both* ways sign-ups stop: an admin
	// closing them explicitly (Match.RegistrationsClosedAt), and kick-off
	// having passed (Match.ScheduledAt, the hard backstop that applies even
	// when the admin never closed anything). They deliberately share one
	// sentinel: from the caller's point of view it is the same refusal and the
	// same remedy (none — talk to an admin), so splitting them would only leak
	// scheduling detail without changing anything a client can do about it.
	ErrRegistrationsClosed = errors.New("registrations for this match are closed")

	// ErrAlreadyRegistered is returned when the player already holds a sign-up
	// for the match — the same "you are already in this state" shape as
	// ErrAlreadyMember on groups.
	ErrAlreadyRegistered = errors.New("player is already registered for this match")

	// ErrNotRegistered is returned when withdrawing a player who has no
	// sign-up for the match.
	ErrNotRegistered = errors.New("player is not registered for this match")
)

// MatchRegistrationService owns the sign-up list of a scheduled match: who
// registered, in which order, and therefore who is confirmed and who is on the
// waiting list. Nothing about that list is stored beyond one row per sign-up —
// see models.MatchRegistration for why the waiting list is derived.
//
// Note what this service deliberately does *not* check: that the player
// belongs to the match's group. That is authorization, enforced one layer up
// by RequireGroupMembership on the route, exactly as MatchService relies on
// requireGroupAdmin rather than re-checking roles itself.
type MatchRegistrationService struct {
	DB *gorm.DB
}

func NewMatchRegistrationService(db *gorm.DB) *MatchRegistrationService {
	return &MatchRegistrationService{DB: db}
}

// RegistrationWindowError reports why match is not accepting sign-ups at
// instant now, or nil when it is. It is a pure function of an already-loaded
// match — the same split as ComputePointsStandings versus its data loading —
// so the whole window policy is unit-testable without a database, and both
// Register and Unregister share exactly one definition of "open".
//
// Withdrawing is gated on the same window as registering: once the admin has
// closed sign-ups the roster is being turned into teams, and once the match
// has kicked off the list is history. A player who wants out after that talks
// to an admin.
//
// RegistrationOpensAt is nil-guarded rather than dereferenced: CreateMatch
// validates scheduling as all-or-nothing, so a scheduled match always has one,
// but a row hand-crafted in SQL might not — and a scheduled match with no
// stated opening time is best read as "open since it was created".
func RegistrationWindowError(match models.Match, now time.Time) error {
	if !match.IsScheduled() {
		return ErrMatchNotScheduled
	}
	if match.RegistrationOpensAt != nil && now.Before(*match.RegistrationOpensAt) {
		return ErrRegistrationsNotOpenYet
	}
	if match.RegistrationsClosedAt != nil {
		return ErrRegistrationsClosed
	}
	// Kick-off is the backstop: registering for a match that has already
	// started is never meaningful, whether or not an admin remembered to close
	// the list. !Before rather than After so that kick-off itself is closed.
	if !now.Before(*match.ScheduledAt) {
		return ErrRegistrationsClosed
	}
	return nil
}

// ComputeRegistrationPositions numbers an already-ordered sign-up list and
// flags the tail beyond maxPlayers as waiting. It is the entire waiting-list
// rule, kept as a pure function of already-loaded data (see
// ComputePointsStandings for the same split) — which is also what makes
// lowering maxPlayers a non-event: the tail simply reads as waiting on the
// next call, with no rows to rewrite.
//
// entries must already be in sign-up order; the ordering itself is
// ListRegistrations' job, since it is what the database can do deterministically.
// A nil maxPlayers means no cap is configured, so nobody is waiting.
func ComputeRegistrationPositions(entries []models.MatchRegistrationEntry, maxPlayers *int) []models.MatchRegistrationEntry {
	for i := range entries {
		entries[i].Position = i + 1
		entries[i].IsWaiting = maxPlayers != nil && entries[i].Position > *maxPlayers
	}
	return entries
}

// Register signs playerID up for matchID, if the registration window is open.
//
// Capacity is deliberately *not* a rejection: past Match.MaxPlayers the
// sign-up still happens and simply lands on the waiting list, which is the
// whole point of the derived design.
func (s *MatchRegistrationService) Register(matchID, playerID uuid.UUID) error {
	match, err := s.findMatch(matchID)
	if err != nil {
		return err
	}
	if err := RegistrationWindowError(*match, time.Now()); err != nil {
		return err
	}

	// Checked rather than left to the unique index so the caller gets a clean
	// sentinel instead of a driver-level constraint error. The index is still
	// the real guarantee under a concurrent double-click — this check only
	// makes the common case a well-typed refusal.
	var existing int64
	if err := s.DB.Model(&models.MatchRegistration{}).
		Where("match_id = ? AND player_id = ?", matchID, playerID).
		Count(&existing).Error; err != nil {
		return err
	}
	if existing > 0 {
		return ErrAlreadyRegistered
	}

	return s.DB.Create(&models.MatchRegistration{MatchID: matchID, PlayerID: playerID}).Error
}

// Unregister withdraws playerID from matchID. Deleting the row is all it takes
// to promote the first waiting player: positions are recomputed from the
// remaining rows' order on the next read, so there is no promotion step to run
// (and none to get wrong).
func (s *MatchRegistrationService) Unregister(matchID, playerID uuid.UUID) error {
	match, err := s.findMatch(matchID)
	if err != nil {
		return err
	}
	if err := RegistrationWindowError(*match, time.Now()); err != nil {
		return err
	}

	result := s.DB.Where("match_id = ? AND player_id = ?", matchID, playerID).
		Delete(&models.MatchRegistration{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotRegistered
	}
	return nil
}

// ListRegistrations returns matchID's sign-ups in order, each tagged with its
// 1-based position and whether it is on the waiting list.
//
// The ordering is total and deterministic: created_at first, then id as a
// tie-break. Two sign-ups landing in the same microsecond would otherwise be
// ordered arbitrarily by the database, which — right at the confirmed/waiting
// boundary — could swap the last confirmed player and the first waiting one
// between two reads of the same unchanged data.
//
// An unscheduled match is not an error here (unlike Register): it simply has
// no sign-ups, and returning an empty list keeps a caller from having to know
// about scheduling just to render a match.
func (s *MatchRegistrationService) ListRegistrations(matchID uuid.UUID) ([]models.MatchRegistrationEntry, error) {
	match, err := s.findMatch(matchID)
	if err != nil {
		return nil, err
	}

	// A genuine two-table join (the player's name lives on players), which is
	// the case Joins/Scan is for — the rest of this service stays on the query
	// builder.
	var entries []models.MatchRegistrationEntry
	if err := s.DB.Model(&models.MatchRegistration{}).
		Select("match_registrations.player_id AS player_id, players.name AS name, match_registrations.created_at AS registered_at").
		Joins("JOIN players ON players.id = match_registrations.player_id").
		Where("match_registrations.match_id = ?", matchID).
		Order("match_registrations.created_at ASC").
		Order("match_registrations.id ASC").
		Scan(&entries).Error; err != nil {
		return nil, err
	}

	// GORM leaves the slice nil when nothing matched; an empty list is a normal
	// state (nobody has signed up yet), so it is normalized the same way
	// MatchHandler.GetMatchesDetails normalizes its own.
	if entries == nil {
		entries = []models.MatchRegistrationEntry{}
	}
	return ComputeRegistrationPositions(entries, match.MaxPlayers), nil
}

// CloseRegistrations stops sign-ups for a scheduled match, recording when it
// happened. This is the admin's "the roster is final, let me compose the
// teams" action.
//
// Closing an already-closed match is a successful no-op rather than an error —
// the same reasoning as UpdateMemberRole setting the role a member already
// has: a retried request must not fail for nothing, and the timestamp of the
// *first* close is the one worth keeping.
func (s *MatchRegistrationService) CloseRegistrations(matchID, groupID uuid.UUID) error {
	match, err := s.findMatchInGroup(matchID, groupID)
	if err != nil {
		return err
	}
	if !match.IsScheduled() {
		return ErrMatchNotScheduled
	}
	if match.RegistrationsClosedAt != nil {
		return nil
	}

	now := time.Now()
	return s.DB.Model(&models.Match{}).Where("id = ?", match.ID).
		Updates(map[string]any{"registrations_closed_at": now}).Error
}

// ReopenRegistrations clears the close timestamp, so a mis-clicked
// CloseRegistrations can be undone. Re-opening an already-open match is a
// no-op, symmetrically with CloseRegistrations.
//
// It is allowed even after kick-off, where it has no practical effect:
// RegistrationWindowError still refuses every sign-up on the ScheduledAt
// backstop. Refusing it here would only add a rule without protecting
// anything.
func (s *MatchRegistrationService) ReopenRegistrations(matchID, groupID uuid.UUID) error {
	match, err := s.findMatchInGroup(matchID, groupID)
	if err != nil {
		return err
	}
	if !match.IsScheduled() {
		return ErrMatchNotScheduled
	}
	if match.RegistrationsClosedAt == nil {
		return nil
	}

	// A map is required to write a NULL: GORM skips zero values passed to
	// Updates as a struct.
	return s.DB.Model(&models.Match{}).Where("id = ?", match.ID).
		Updates(map[string]any{"registrations_closed_at": nil}).Error
}

// findMatch loads a match by id alone. The player-facing actions (Register,
// Unregister, ListRegistrations) reach a match through a route that already
// authorized the caller against the group carrying it, so there is no group to
// scope against here.
func (s *MatchRegistrationService) findMatch(matchID uuid.UUID) (*models.Match, error) {
	var match models.Match
	if err := s.DB.Where("id = ?", matchID).First(&match).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMatchNotFound
		}
		return nil, err
	}
	return &match, nil
}

// findMatchInGroup loads a match scoped to a group, exactly as
// MatchService.DeleteMatch and GetMatchDetailsByID do: a match id belonging to
// another group reads as ErrMatchNotFound rather than being reachable, since
// the admin middleware only proves the caller administers groupID — it says
// nothing about which group the match id in the path belongs to.
func (s *MatchRegistrationService) findMatchInGroup(matchID, groupID uuid.UUID) (*models.Match, error) {
	var match models.Match
	if err := s.DB.Where("id = ? AND group_id = ?", matchID, groupID).First(&match).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMatchNotFound
		}
		return nil, err
	}
	return &match, nil
}
