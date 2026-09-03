package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

// The scheduling fields on MatchWithDetails are a wire-format change, so these
// tests assert the JSON itself rather than the Go struct: what protects the
// existing frontend is not that the fields are pointers, it is that an
// unscheduled match still serializes to exactly the keys it always did.

func TestMatchWithDetails_JSON_UnscheduledMatchHasNoSchedulingKeys(t *testing.T) {
	match := MatchWithDetails{
		ID:      uuid.New(),
		GroupID: uuid.New(),
		Date:    Date(time.Date(2025, time.October, 5, 0, 0, 0, 0, time.UTC)),
		Teams:   []TeamWithPlayers{},
	}

	raw, err := json.Marshal(match)
	if err != nil {
		t.Fatalf("failed to marshal match: %v", err)
	}

	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("failed to unmarshal match: %v", err)
	}

	// Exactly the pre-feature key set: a client that knows nothing about
	// scheduling must not see a single new key on an ordinary match.
	wantKeys := map[string]bool{"ID": true, "GroupID": true, "Date": true, "Teams": true}
	for key := range decoded {
		if !wantKeys[key] {
			t.Errorf("unscheduled match JSON carries unexpected key %q: %s", key, raw)
		}
	}
	for key := range wantKeys {
		if _, ok := decoded[key]; !ok {
			t.Errorf("unscheduled match JSON is missing key %q: %s", key, raw)
		}
	}
}

func TestMatchWithDetails_JSON_ScheduledMatchCarriesEveryKey(t *testing.T) {
	kickoff := time.Date(2025, time.October, 5, 19, 30, 0, 0, time.UTC)
	opensAt := time.Date(2025, time.October, 1, 8, 0, 0, 0, time.UTC)
	closedAt := time.Date(2025, time.October, 4, 20, 0, 0, 0, time.UTC)
	maxPlayers := 16
	count := 12

	match := MatchWithDetails{
		ID:                    uuid.New(),
		GroupID:               uuid.New(),
		Date:                  DateOf(kickoff),
		ScheduledAt:           &kickoff,
		RegistrationOpensAt:   &opensAt,
		RegistrationsClosedAt: &closedAt,
		MaxPlayers:            &maxPlayers,
		RegistrationCount:     &count,
		Teams:                 []TeamWithPlayers{},
	}

	raw, err := json.Marshal(match)
	if err != nil {
		t.Fatalf("failed to marshal match: %v", err)
	}

	var decoded struct {
		Date                  string     `json:"Date"`
		ScheduledAt           *time.Time `json:"ScheduledAt"`
		RegistrationOpensAt   *time.Time `json:"RegistrationOpensAt"`
		RegistrationsClosedAt *time.Time `json:"RegistrationsClosedAt"`
		MaxPlayers            *int       `json:"MaxPlayers"`
		RegistrationCount     *int       `json:"RegistrationCount"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("failed to unmarshal match: %v", err)
	}

	// Date keeps its "YYYY-MM-DD" contract and stays consistent with the
	// kick-off day, which is the invariant CreateMatch maintains on write.
	if decoded.Date != "2025-10-05" {
		t.Errorf("Date = %q, want 2025-10-05", decoded.Date)
	}
	if decoded.ScheduledAt == nil || !decoded.ScheduledAt.Equal(kickoff) {
		t.Errorf("ScheduledAt = %v, want %v", decoded.ScheduledAt, kickoff)
	}
	if decoded.RegistrationOpensAt == nil || !decoded.RegistrationOpensAt.Equal(opensAt) {
		t.Errorf("RegistrationOpensAt = %v, want %v", decoded.RegistrationOpensAt, opensAt)
	}
	if decoded.RegistrationsClosedAt == nil || !decoded.RegistrationsClosedAt.Equal(closedAt) {
		t.Errorf("RegistrationsClosedAt = %v, want %v", decoded.RegistrationsClosedAt, closedAt)
	}
	if decoded.MaxPlayers == nil || *decoded.MaxPlayers != 16 {
		t.Errorf("MaxPlayers = %v, want 16", decoded.MaxPlayers)
	}
	if decoded.RegistrationCount == nil || *decoded.RegistrationCount != 12 {
		t.Errorf("RegistrationCount = %v, want 12", decoded.RegistrationCount)
	}
}

// TestMatchWithDetails_JSON_ZeroSignUpsIsStillPresent is the reason
// RegistrationCount is a *int rather than a plain int: a plain int would be
// dropped by omitempty at zero, making "nobody has signed up yet" and "this
// match has no sign-up list at all" indistinguishable on the wire — and
// "0 / 16 signed up" is a real thing for a match card to render.
func TestMatchWithDetails_JSON_ZeroSignUpsIsStillPresent(t *testing.T) {
	kickoff := time.Date(2025, time.October, 5, 19, 30, 0, 0, time.UTC)
	maxPlayers := 16
	count := 0

	raw, err := json.Marshal(MatchWithDetails{
		ID:                uuid.New(),
		GroupID:           uuid.New(),
		Date:              DateOf(kickoff),
		ScheduledAt:       &kickoff,
		MaxPlayers:        &maxPlayers,
		RegistrationCount: &count,
		Teams:             []TeamWithPlayers{},
	})
	if err != nil {
		t.Fatalf("failed to marshal match: %v", err)
	}

	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("failed to unmarshal match: %v", err)
	}
	if got, ok := decoded["RegistrationCount"]; !ok || string(got) != "0" {
		t.Errorf("RegistrationCount = %s (present=%v), want 0 and present: %s", got, ok, raw)
	}
	// An unset RegistrationsClosedAt still disappears, so "sign-ups are open"
	// stays an absent key rather than an explicit null.
	if _, ok := decoded["RegistrationsClosedAt"]; ok {
		t.Errorf("RegistrationsClosedAt should be omitted while sign-ups are open: %s", raw)
	}
}
