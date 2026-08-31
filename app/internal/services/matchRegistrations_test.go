package services

import (
	"errors"
	"testing"
	"time"

	"app/internal/models"

	"github.com/google/uuid"
)

// The tests in this file are pure: ComputeRegistrationPositions and
// RegistrationWindowError are functions of already-loaded data, so the whole
// waiting-list and window policy is covered without a database — the same
// split standings_test.go exercises on ComputePointsStandings.

func entriesFor(n int) []models.MatchRegistrationEntry {
	entries := make([]models.MatchRegistrationEntry, n)
	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	for i := range entries {
		entries[i] = models.MatchRegistrationEntry{
			PlayerID:     uuid.New(),
			Name:         "player",
			RegisteredAt: base.Add(time.Duration(i) * time.Minute),
		}
	}
	return entries
}

func intPtr(v int) *int { return &v }

func TestComputeRegistrationPositions_ConfirmedAndWaiting(t *testing.T) {
	got := ComputeRegistrationPositions(entriesFor(5), intPtr(3))

	for i, entry := range got {
		if entry.Position != i+1 {
			t.Errorf("entry %d position = %d, want %d (positions are 1-based and contiguous)", i, entry.Position, i+1)
		}
		wantWaiting := i >= 3
		if entry.IsWaiting != wantWaiting {
			t.Errorf("entry %d (position %d) IsWaiting = %v, want %v", i, entry.Position, entry.IsWaiting, wantWaiting)
		}
	}
}

func TestComputeRegistrationPositions_NoCapMeansNobodyWaiting(t *testing.T) {
	got := ComputeRegistrationPositions(entriesFor(4), nil)

	for i, entry := range got {
		if entry.IsWaiting {
			t.Errorf("entry %d IsWaiting = true, want false when no max is configured", i)
		}
		if entry.Position != i+1 {
			t.Errorf("entry %d position = %d, want %d", i, entry.Position, i+1)
		}
	}
}

func TestComputeRegistrationPositions_Empty(t *testing.T) {
	if got := ComputeRegistrationPositions([]models.MatchRegistrationEntry{}, intPtr(8)); len(got) != 0 {
		t.Errorf("got %d entries for an empty list, want 0", len(got))
	}
}

// scheduledMatch builds an in-memory scheduled match, with no DB involved.
func scheduledMatch(opensAt, kickOff time.Time, closedAt *time.Time) models.Match {
	max := 10
	return models.Match{
		ScheduledAt:           &kickOff,
		RegistrationOpensAt:   &opensAt,
		RegistrationsClosedAt: closedAt,
		MaxPlayers:            &max,
	}
}

func TestRegistrationWindowError(t *testing.T) {
	opensAt := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	kickOff := time.Date(2026, 9, 6, 18, 0, 0, 0, time.UTC)
	closedAt := time.Date(2026, 9, 5, 9, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		match models.Match
		now   time.Time
		want  error
	}{
		{
			name:  "unscheduled match",
			match: models.Match{},
			now:   opensAt,
			want:  ErrMatchNotScheduled,
		},
		{
			name:  "before opening",
			match: scheduledMatch(opensAt, kickOff, nil),
			now:   opensAt.Add(-time.Second),
			want:  ErrRegistrationsNotOpenYet,
		},
		{
			name:  "at opening instant",
			match: scheduledMatch(opensAt, kickOff, nil),
			now:   opensAt,
			want:  nil,
		},
		{
			name:  "open",
			match: scheduledMatch(opensAt, kickOff, nil),
			now:   opensAt.Add(time.Hour),
			want:  nil,
		},
		{
			name:  "closed by the admin",
			match: scheduledMatch(opensAt, kickOff, &closedAt),
			now:   closedAt.Add(time.Hour),
			want:  ErrRegistrationsClosed,
		},
		{
			// The backstop: nobody closed anything, but the match has started.
			name:  "kick-off passed, never closed",
			match: scheduledMatch(opensAt, kickOff, nil),
			now:   kickOff.Add(time.Minute),
			want:  ErrRegistrationsClosed,
		},
		{
			name:  "kick-off instant itself is already too late",
			match: scheduledMatch(opensAt, kickOff, nil),
			now:   kickOff,
			want:  ErrRegistrationsClosed,
		},
		{
			// Defensive: a row hand-crafted with a kick-off but no opening
			// time reads as "open since it was created".
			name:  "no opening time configured",
			match: models.Match{ScheduledAt: &kickOff},
			now:   opensAt,
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RegistrationWindowError(tt.match, tt.now); !errors.Is(got, tt.want) {
				t.Errorf("RegistrationWindowError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchSpecValidate(t *testing.T) {
	kickOff := time.Date(2026, 9, 6, 18, 0, 0, 0, time.UTC)
	opensAt := kickOff.Add(-48 * time.Hour)

	tests := []struct {
		name string
		spec MatchSpec
		want error
	}{
		{
			name: "no scheduling at all stays valid",
			spec: MatchSpec{Date: models.DateOf(kickOff)},
			want: nil,
		},
		{
			name: "fully scheduled",
			spec: MatchSpec{ScheduledAt: &kickOff, RegistrationOpensAt: &opensAt, MaxPlayers: intPtr(10)},
			want: nil,
		},
		{
			name: "kick-off without a max",
			spec: MatchSpec{ScheduledAt: &kickOff, RegistrationOpensAt: &opensAt},
			want: ErrIncompleteSchedule,
		},
		{
			name: "max without a kick-off",
			spec: MatchSpec{MaxPlayers: intPtr(10)},
			want: ErrIncompleteSchedule,
		},
		{
			name: "registrations opening at kick-off",
			spec: MatchSpec{ScheduledAt: &kickOff, RegistrationOpensAt: &kickOff, MaxPlayers: intPtr(10)},
			want: ErrRegistrationOpensAfterKickoff,
		},
		{
			name: "registrations opening after kick-off",
			spec: MatchSpec{ScheduledAt: &kickOff, RegistrationOpensAt: timePtr(kickOff.Add(time.Hour)), MaxPlayers: intPtr(10)},
			want: ErrRegistrationOpensAfterKickoff,
		},
		{
			name: "zero max players",
			spec: MatchSpec{ScheduledAt: &kickOff, RegistrationOpensAt: &opensAt, MaxPlayers: intPtr(0)},
			want: ErrInvalidMaxPlayers,
		},
		{
			name: "negative max players",
			spec: MatchSpec{ScheduledAt: &kickOff, RegistrationOpensAt: &opensAt, MaxPlayers: intPtr(-1)},
			want: ErrInvalidMaxPlayers,
		},
		{
			// A kick-off in the past is legitimate: backfilling a match that
			// already happened has always been allowed.
			name: "kick-off in the past is accepted",
			spec: MatchSpec{
				ScheduledAt:         timePtr(time.Now().Add(-72 * time.Hour)),
				RegistrationOpensAt: timePtr(time.Now().Add(-96 * time.Hour)),
				MaxPlayers:          intPtr(10),
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.spec.validate(); !errors.Is(got, tt.want) {
				t.Errorf("validate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func timePtr(t time.Time) *time.Time { return &t }

// TestDateOf_KeepsTheClientsCalendarDay is the reason DateOf does not
// normalize to UTC: a 21:00 kick-off two hours east of Greenwich is 19:00 UTC
// the same day, but a 00:30 one would roll back to the previous day, and the
// match belongs on the day the client scheduled it.
func TestDateOf_KeepsTheClientsCalendarDay(t *testing.T) {
	east := time.FixedZone("UTC+2", 2*60*60)

	tests := []struct {
		name string
		in   time.Time
		want string
	}{
		{"evening kick-off east of UTC", time.Date(2026, 9, 14, 21, 0, 0, 0, east), "2026-09-14"},
		{"just after midnight east of UTC", time.Date(2026, 9, 14, 0, 30, 0, 0, east), "2026-09-14"},
		{"plain UTC", time.Date(2026, 9, 14, 18, 0, 0, 0, time.UTC), "2026-09-14"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := models.DateOf(tt.in).String(); got != tt.want {
				t.Errorf("DateOf(%s) = %s, want %s", tt.in, got, tt.want)
			}
		})
	}
}
