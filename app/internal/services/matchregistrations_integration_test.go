package services_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// registrationEnv is the fixture every test below starts from: a group, a
// match, and a handful of players to sign up. Players are not added to the
// group's membership on purpose — MatchRegistrationService knows nothing about
// membership, that check belongs to the route's middleware (see the service's
// own doc comment).
type registrationEnv struct {
	tx            *gorm.DB
	matches       *services.MatchService
	registrations *services.MatchRegistrationService
	groupID       uuid.UUID
	matchID       uuid.UUID
	players       []uuid.UUID
	playerNames   []string
}

// newRegistrationEnv creates a match from spec plus playerCount players.
func newRegistrationEnv(t *testing.T, label string, spec services.MatchSpec, playerCount int) *registrationEnv {
	t.Helper()

	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	playerService := services.NewPlayerService(tx)
	matchService := services.NewMatchService(tx)

	group, err := groupService.CreateGroup("Zzz Registrations "+label, services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	matchID, err := matchService.CreateMatch(spec, group.ID)
	if err != nil {
		t.Fatalf("failed to create match: %v", err)
	}

	env := &registrationEnv{
		tx:            tx,
		matches:       matchService,
		registrations: services.NewMatchRegistrationService(tx),
		groupID:       group.ID,
		matchID:       matchID,
	}
	for i := 0; i < playerCount; i++ {
		name := fmt.Sprintf("Zzz Reg %s %02d", label, i)
		id, err := playerService.CreatePlayer(name)
		if err != nil {
			t.Fatalf("failed to create player %q: %v", name, err)
		}
		env.players = append(env.players, id)
		env.playerNames = append(env.playerNames, name)
	}
	return env
}

// openSpec is a match whose registrations are already open and whose kick-off
// is still ahead — the normal state a player signs up in.
func openSpec(maxPlayers int) services.MatchSpec {
	now := time.Now()
	kickOff := now.Add(48 * time.Hour)
	opensAt := now.Add(-time.Hour)
	return services.MatchSpec{
		ScheduledAt:         &kickOff,
		RegistrationOpensAt: &opensAt,
		MaxPlayers:          &maxPlayers,
	}
}

func (env *registrationEnv) list(t *testing.T) []models.MatchRegistrationEntry {
	t.Helper()
	entries, err := env.registrations.ListRegistrations(env.matchID)
	if err != nil {
		t.Fatalf("ListRegistrations returned error: %v", err)
	}
	return entries
}

// registerAll signs up the first n players, in order, and fails the test on
// any refusal — the ordering it establishes is what every position assertion
// below depends on.
func (env *registrationEnv) registerAll(t *testing.T, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		if err := env.registrations.Register(env.matchID, env.players[i]); err != nil {
			t.Fatalf("Register(player %d) returned error: %v", i, err)
		}
	}
}

// assertList checks the full derived shape of the list in one go: who is in it,
// in which order, with contiguous 1-based positions, and where the
// confirmed/waiting boundary falls.
func (env *registrationEnv) assertList(t *testing.T, wantPlayers []uuid.UUID, wantConfirmed int) {
	t.Helper()
	entries := env.list(t)
	if len(entries) != len(wantPlayers) {
		t.Fatalf("got %d registrations, want %d", len(entries), len(wantPlayers))
	}
	for i, entry := range entries {
		if entry.PlayerID != wantPlayers[i] {
			t.Errorf("registration %d is player %s, want %s", i, entry.PlayerID, wantPlayers[i])
		}
		if entry.Position != i+1 {
			t.Errorf("registration %d position = %d, want %d", i, entry.Position, i+1)
		}
		wantWaiting := i >= wantConfirmed
		if entry.IsWaiting != wantWaiting {
			t.Errorf("registration %d (position %d) IsWaiting = %v, want %v", i, entry.Position, entry.IsWaiting, wantWaiting)
		}
		if entry.Name == "" {
			t.Errorf("registration %d has an empty name: the join on players did not resolve", i)
		}
	}
}

// TestRegister_Integration_BeforeOpeningRefused: the "Participate" button is
// not usable before RegistrationOpensAt, and the service says so rather than
// trusting the frontend to hide it.
func TestRegister_Integration_BeforeOpeningRefused(t *testing.T) {
	now := time.Now()
	kickOff := now.Add(72 * time.Hour)
	opensAt := now.Add(24 * time.Hour)
	max := 10
	env := newRegistrationEnv(t, "NotOpenYet", services.MatchSpec{
		ScheduledAt: &kickOff, RegistrationOpensAt: &opensAt, MaxPlayers: &max,
	}, 1)

	if err := env.registrations.Register(env.matchID, env.players[0]); !errors.Is(err, services.ErrRegistrationsNotOpenYet) {
		t.Errorf("Register before opening = %v, want services.ErrRegistrationsNotOpenYet", err)
	}
	if entries := env.list(t); len(entries) != 0 {
		t.Errorf("got %d registrations after a refused sign-up, want 0", len(entries))
	}
}

// TestRegister_Integration_OnceOpenAndNoDoubleSignup covers the happy path and
// the double-click: the second sign-up is refused with a sentinel rather than
// relying on the unique index to surface a driver error.
func TestRegister_Integration_OnceOpenAndNoDoubleSignup(t *testing.T) {
	env := newRegistrationEnv(t, "Open", openSpec(10), 1)

	if err := env.registrations.Register(env.matchID, env.players[0]); err != nil {
		t.Fatalf("Register once open returned error: %v", err)
	}
	env.assertList(t, env.players[:1], 1)

	if err := env.registrations.Register(env.matchID, env.players[0]); !errors.Is(err, services.ErrAlreadyRegistered) {
		t.Errorf("Register twice = %v, want services.ErrAlreadyRegistered", err)
	}
	if entries := env.list(t); len(entries) != 1 {
		t.Errorf("got %d registrations after a duplicate sign-up, want 1", len(entries))
	}
}

// TestRegister_Integration_WaitingListBeyondMax: the (max+1)th sign-up is not
// refused, it lands on the waiting list — which is the whole design.
func TestRegister_Integration_WaitingListBeyondMax(t *testing.T) {
	const max = 3
	env := newRegistrationEnv(t, "Waiting", openSpec(max), max+2)

	env.registerAll(t, max+2)
	env.assertList(t, env.players, max)
}

// TestUnregister_Integration_PromotesFirstWaitingPlayer is the point of the
// derived waiting list: a withdrawal is a plain row delete, and the next
// player is confirmed simply because the list is re-derived on read. Asserted
// by re-reading, not by inspecting any stored status.
func TestUnregister_Integration_PromotesFirstWaitingPlayer(t *testing.T) {
	const max = 2
	env := newRegistrationEnv(t, "Promote", openSpec(max), 4)

	env.registerAll(t, 4)
	env.assertList(t, env.players, max)

	// The first confirmed player withdraws.
	if err := env.registrations.Unregister(env.matchID, env.players[0]); err != nil {
		t.Fatalf("Unregister returned error: %v", err)
	}

	// Everyone shifts up one place: the player who was first on the waiting
	// list (position max+1) is now confirmed.
	env.assertList(t, env.players[1:], max)

	entries := env.list(t)
	promoted := entries[max-1]
	if promoted.PlayerID != env.players[max] {
		t.Fatalf("player at the last confirmed slot is %s, want the promoted %s", promoted.PlayerID, env.players[max])
	}
	if promoted.IsWaiting {
		t.Errorf("promoted player is still flagged as waiting")
	}

	// Withdrawing again is refused: there is nothing left to delete.
	if err := env.registrations.Unregister(env.matchID, env.players[0]); !errors.Is(err, services.ErrNotRegistered) {
		t.Errorf("Unregister twice = %v, want services.ErrNotRegistered", err)
	}
}

// TestRegistrations_Integration_ClosedByAdminThenReopened covers the admin's
// manual close (which blocks both registering and withdrawing, since the
// roster is being turned into teams) and the re-open that recovers from a
// mis-click.
func TestRegistrations_Integration_ClosedByAdminThenReopened(t *testing.T) {
	env := newRegistrationEnv(t, "CloseReopen", openSpec(10), 2)

	env.registerAll(t, 1)

	if err := env.registrations.CloseRegistrations(env.matchID, env.groupID); err != nil {
		t.Fatalf("CloseRegistrations returned error: %v", err)
	}

	if err := env.registrations.Register(env.matchID, env.players[1]); !errors.Is(err, services.ErrRegistrationsClosed) {
		t.Errorf("Register after close = %v, want services.ErrRegistrationsClosed", err)
	}
	if err := env.registrations.Unregister(env.matchID, env.players[0]); !errors.Is(err, services.ErrRegistrationsClosed) {
		t.Errorf("Unregister after close = %v, want services.ErrRegistrationsClosed", err)
	}
	// Reading the list stays possible while closed — that is exactly when the
	// admin composes the teams from it.
	env.assertList(t, env.players[:1], 1)

	// Closing twice is a no-op, not an error (a retried request must not fail).
	if err := env.registrations.CloseRegistrations(env.matchID, env.groupID); err != nil {
		t.Errorf("CloseRegistrations twice returned error: %v", err)
	}

	if err := env.registrations.ReopenRegistrations(env.matchID, env.groupID); err != nil {
		t.Fatalf("ReopenRegistrations returned error: %v", err)
	}
	if err := env.registrations.Register(env.matchID, env.players[1]); err != nil {
		t.Errorf("Register after re-open returned error: %v", err)
	}
	if err := env.registrations.Unregister(env.matchID, env.players[0]); err != nil {
		t.Errorf("Unregister after re-open returned error: %v", err)
	}
	// Re-opening twice is a no-op too.
	if err := env.registrations.ReopenRegistrations(env.matchID, env.groupID); err != nil {
		t.Errorf("ReopenRegistrations twice returned error: %v", err)
	}
}

// TestRegister_Integration_AfterKickoffRefused is the backstop: registrations
// were never closed by anyone, but the match has already started.
func TestRegister_Integration_AfterKickoffRefused(t *testing.T) {
	now := time.Now()
	kickOff := now.Add(-time.Hour)
	opensAt := now.Add(-72 * time.Hour)
	max := 10
	env := newRegistrationEnv(t, "KickedOff", services.MatchSpec{
		ScheduledAt: &kickOff, RegistrationOpensAt: &opensAt, MaxPlayers: &max,
	}, 1)

	// Nobody closed anything.
	var match models.Match
	if err := env.tx.Where("id = ?", env.matchID).First(&match).Error; err != nil {
		t.Fatalf("failed to reload match: %v", err)
	}
	if match.RegistrationsClosedAt != nil {
		t.Fatalf("precondition failed: registrations_closed_at is set, the backstop would not be what is under test")
	}

	if err := env.registrations.Register(env.matchID, env.players[0]); !errors.Is(err, services.ErrRegistrationsClosed) {
		t.Errorf("Register after kick-off = %v, want services.ErrRegistrationsClosed", err)
	}
}

// TestRegister_Integration_UnscheduledMatchRefused: a plain match recorded
// after the fact has nothing to sign up for.
func TestRegister_Integration_UnscheduledMatchRefused(t *testing.T) {
	env := newRegistrationEnv(t, "Unscheduled", services.MatchSpec{Date: models.Date(time.Now())}, 1)

	if err := env.registrations.Register(env.matchID, env.players[0]); !errors.Is(err, services.ErrMatchNotScheduled) {
		t.Errorf("Register on an unscheduled match = %v, want services.ErrMatchNotScheduled", err)
	}
	if err := env.registrations.Unregister(env.matchID, env.players[0]); !errors.Is(err, services.ErrMatchNotScheduled) {
		t.Errorf("Unregister on an unscheduled match = %v, want services.ErrMatchNotScheduled", err)
	}
	if err := env.registrations.CloseRegistrations(env.matchID, env.groupID); !errors.Is(err, services.ErrMatchNotScheduled) {
		t.Errorf("CloseRegistrations on an unscheduled match = %v, want services.ErrMatchNotScheduled", err)
	}
	// Listing, on the other hand, is not an error: an unscheduled match simply
	// has no sign-ups.
	env.assertList(t, nil, 0)
}

// TestCloseRegistrations_Integration_WrongGroupNotFound mirrors
// TestDeleteMatch_Integration_WrongGroupNotFound: an admin of another group
// must not be able to reach this match at all, so the group-scoped lookup
// reports it as missing.
func TestCloseRegistrations_Integration_WrongGroupNotFound(t *testing.T) {
	env := newRegistrationEnv(t, "WrongGroup", openSpec(10), 0)

	otherGroup, err := services.NewGroupService(env.tx).CreateGroup("Zzz Registrations WrongGroup Other", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create the other group: %v", err)
	}

	if err := env.registrations.CloseRegistrations(env.matchID, otherGroup.ID); !errors.Is(err, services.ErrMatchNotFound) {
		t.Errorf("CloseRegistrations with the wrong group = %v, want services.ErrMatchNotFound", err)
	}
	if err := env.registrations.ReopenRegistrations(env.matchID, otherGroup.ID); !errors.Is(err, services.ErrMatchNotFound) {
		t.Errorf("ReopenRegistrations with the wrong group = %v, want services.ErrMatchNotFound", err)
	}

	// And the match is genuinely untouched.
	var match models.Match
	if err := env.tx.Where("id = ?", env.matchID).First(&match).Error; err != nil {
		t.Fatalf("failed to reload match: %v", err)
	}
	if match.RegistrationsClosedAt != nil {
		t.Error("registrations were closed through another group's scope")
	}

	// An unknown id is the same refusal.
	if err := env.registrations.CloseRegistrations(uuid.New(), env.groupID); !errors.Is(err, services.ErrMatchNotFound) {
		t.Errorf("CloseRegistrations on an unknown match = %v, want services.ErrMatchNotFound", err)
	}
	if _, err := env.registrations.ListRegistrations(uuid.New()); !errors.Is(err, services.ErrMatchNotFound) {
		t.Errorf("ListRegistrations on an unknown match = %v, want services.ErrMatchNotFound", err)
	}
	if err := env.registrations.Register(uuid.New(), uuid.New()); !errors.Is(err, services.ErrMatchNotFound) {
		t.Errorf("Register on an unknown match = %v, want services.ErrMatchNotFound", err)
	}
}

// TestListRegistrations_Integration_LoweringMaxPlayers is the free property of
// deriving the waiting list: lowering the cap rolls the tail of an existing
// list into the waiting list with no rows rewritten.
func TestListRegistrations_Integration_LoweringMaxPlayers(t *testing.T) {
	env := newRegistrationEnv(t, "LowerMax", openSpec(4), 4)

	env.registerAll(t, 4)
	env.assertList(t, env.players, 4)

	// Written straight to the row: editing a schedule is a later slice, and
	// what is under test here is only that the list re-derives from whatever
	// MaxPlayers currently says.
	if err := env.tx.Model(&models.Match{}).Where("id = ?", env.matchID).
		Update("max_players", 2).Error; err != nil {
		t.Fatalf("failed to lower max_players: %v", err)
	}

	env.assertList(t, env.players, 2)
}

// TestCreateMatch_Integration_Scheduled asserts the one thing the pure
// validation tests cannot: that a scheduled match's stored Date is
// ScheduledAt's calendar day, so Date (which seasons, ordering and the
// existing JSON contract are built on) can never drift from the kick-off.
func TestCreateMatch_Integration_Scheduled(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	matchService := services.NewMatchService(tx)

	group, err := groupService.CreateGroup("Zzz Scheduled Match Group", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	// An evening kick-off east of UTC: normalizing to UTC would be harmless
	// here, but 00:30 would roll back a day — see TestDateOf_KeepsTheClientsCalendarDay.
	kickOff := time.Date(2026, 9, 13, 21, 0, 0, 0, time.FixedZone("UTC+2", 2*60*60))
	opensAt := kickOff.Add(-72 * time.Hour)
	max := 10

	// The Date supplied here is deliberately wrong: a scheduled match derives
	// it from ScheduledAt instead, so there is a single write path.
	matchID, err := matchService.CreateMatch(services.MatchSpec{
		Date:                models.Date(time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC)),
		ScheduledAt:         &kickOff,
		RegistrationOpensAt: &opensAt,
		MaxPlayers:          &max,
	}, group.ID)
	if err != nil {
		t.Fatalf("CreateMatch returned error: %v", err)
	}

	var match models.Match
	if err := tx.Where("id = ?", matchID).First(&match).Error; err != nil {
		t.Fatalf("failed to reload match: %v", err)
	}

	if got := match.Date.String(); got != "2026-09-13" {
		t.Errorf("stored Date = %s, want 2026-09-13 (ScheduledAt's calendar day)", got)
	}
	if !match.IsScheduled() {
		t.Error("IsScheduled() = false for a match created with a kick-off time")
	}
	if match.ScheduledAt == nil || !match.ScheduledAt.Equal(kickOff) {
		t.Errorf("stored ScheduledAt = %v, want %v", match.ScheduledAt, kickOff)
	}
	if match.RegistrationOpensAt == nil || !match.RegistrationOpensAt.Equal(opensAt) {
		t.Errorf("stored RegistrationOpensAt = %v, want %v", match.RegistrationOpensAt, opensAt)
	}
	if match.MaxPlayers == nil || *match.MaxPlayers != max {
		t.Errorf("stored MaxPlayers = %v, want %d", match.MaxPlayers, max)
	}
	if match.RegistrationsClosedAt != nil {
		t.Errorf("stored RegistrationsClosedAt = %v, want nil (registrations open)", match.RegistrationsClosedAt)
	}
}

// TestCreateMatch_Integration_UnscheduledUnchanged pins the "purely additive"
// promise: a match created with no scheduling stores the caller's date
// verbatim and stays unscheduled.
func TestCreateMatch_Integration_UnscheduledUnchanged(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	group, err := services.NewGroupService(tx).CreateGroup("Zzz Unscheduled Match Group", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	date := models.Date(time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC))
	matchID, err := services.NewMatchService(tx).CreateMatch(services.MatchSpec{Date: date}, group.ID)
	if err != nil {
		t.Fatalf("CreateMatch returned error: %v", err)
	}

	var match models.Match
	if err := tx.Where("id = ?", matchID).First(&match).Error; err != nil {
		t.Fatalf("failed to reload match: %v", err)
	}
	if got := match.Date.String(); got != "2026-03-08" {
		t.Errorf("stored Date = %s, want 2026-03-08", got)
	}
	if match.IsScheduled() {
		t.Error("IsScheduled() = true for a match created without a kick-off time")
	}
	if match.RegistrationOpensAt != nil || match.MaxPlayers != nil || match.RegistrationsClosedAt != nil {
		t.Errorf("scheduling fields = %v/%v/%v, want all nil", match.RegistrationOpensAt, match.MaxPlayers, match.RegistrationsClosedAt)
	}
}

// TestCreateMatch_Integration_ScheduleValidationRejected re-checks the
// validation sentinels through the service (rather than the pure validate()
// unit test) and asserts nothing is written on refusal.
func TestCreateMatch_Integration_ScheduleValidationRejected(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	group, err := services.NewGroupService(tx).CreateGroup("Zzz Invalid Schedule Group", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	matchService := services.NewMatchService(tx)

	kickOff := time.Now().Add(48 * time.Hour)
	opensAt := kickOff.Add(-24 * time.Hour)
	zero, ten := 0, 10

	tests := []struct {
		name string
		spec services.MatchSpec
		want error
	}{
		{"kick-off only", services.MatchSpec{ScheduledAt: &kickOff}, services.ErrIncompleteSchedule},
		{"no max players", services.MatchSpec{ScheduledAt: &kickOff, RegistrationOpensAt: &opensAt}, services.ErrIncompleteSchedule},
		{"no opening time", services.MatchSpec{ScheduledAt: &kickOff, MaxPlayers: &ten}, services.ErrIncompleteSchedule},
		{"opening at kick-off", services.MatchSpec{ScheduledAt: &kickOff, RegistrationOpensAt: &kickOff, MaxPlayers: &ten}, services.ErrRegistrationOpensAfterKickoff},
		{"zero max players", services.MatchSpec{ScheduledAt: &kickOff, RegistrationOpensAt: &opensAt, MaxPlayers: &zero}, services.ErrInvalidMaxPlayers},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := matchService.CreateMatch(tt.spec, group.ID)
			if !errors.Is(err, tt.want) {
				t.Errorf("CreateMatch error = %v, want %v", err, tt.want)
			}
			if id != uuid.Nil {
				t.Errorf("CreateMatch returned id %s on refusal, want uuid.Nil", id)
			}
		})
	}

	var count int64
	if err := tx.Model(&models.Match{}).Where("group_id = ?", group.ID).Count(&count).Error; err != nil {
		t.Fatalf("failed to count matches: %v", err)
	}
	if count != 0 {
		t.Errorf("%d matches were created despite every schedule being invalid", count)
	}
}

// TestDeleteMatch_Integration_RemovesRegistrations: MatchRegistration rows are
// not cascaded by the database, so deleting a scheduled match has to take its
// sign-up list with it — otherwise a re-used match id (or a stats query) would
// see sign-ups for a match that no longer exists.
func TestDeleteMatch_Integration_RemovesRegistrations(t *testing.T) {
	env := newRegistrationEnv(t, "DeleteMatch", openSpec(10), 2)

	env.registerAll(t, 2)

	if err := env.matches.DeleteMatch(env.matchID, env.groupID); err != nil {
		t.Fatalf("DeleteMatch returned error: %v", err)
	}

	var remaining int64
	if err := env.tx.Model(&models.MatchRegistration{}).Where("match_id = ?", env.matchID).
		Count(&remaining).Error; err != nil {
		t.Fatalf("failed to count registrations: %v", err)
	}
	if remaining != 0 {
		t.Errorf("%d registrations survived the match deletion, want 0", remaining)
	}
}
