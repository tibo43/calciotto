package services_test

import (
	"encoding/json"
	"errors"
	"sort"
	"testing"
	"time"

	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// fakePushSender records what would have been delivered, keyed by endpoint,
// and reports endpoints listed in gone as expired.
type fakePushSender struct {
	sent map[string][]services.PushMessage
	gone map[string]bool
}

func (f *fakePushSender) Send(sub models.PushSubscription, payload []byte) error {
	if f.gone[sub.Endpoint] {
		return services.ErrPushSubscriptionGone
	}
	var message services.PushMessage
	if err := json.Unmarshal(payload, &message); err != nil {
		return err
	}
	f.sent[sub.Endpoint] = append(f.sent[sub.Endpoint], message)
	return nil
}

func (f *fakePushSender) reset() { f.sent = map[string][]services.PushMessage{} }

// notificationEnv is a group with its two teams, a NotificationService wired
// into its MatchService the way main.go does it (but sending inline), and a
// helper to add members who have the feature and one subscribed device.
type notificationEnv struct {
	tx            *gorm.DB
	sender        *fakePushSender
	notifications *services.NotificationService
	matches       *services.MatchService
	groupID       uuid.UUID
	teamA, teamB  models.Team
}

func newNotificationEnv(t *testing.T) *notificationEnv {
	t.Helper()
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	group, err := services.NewGroupService(tx).CreateGroup("Zzz Notifications "+uuid.NewString(), services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	teams, err := services.NewTeamService(tx).GetTeamsByGroupID(group.ID)
	if err != nil || len(teams) != 2 {
		t.Fatalf("GetTeamsByGroupID: %v (%d teams)", err, len(teams))
	}

	sender := &fakePushSender{gone: map[string]bool{}}
	sender.reset()
	matches := services.NewMatchService(tx)
	notifications := services.NewNotificationService(tx, sender, matches)
	notifications.Dispatch = func(work func()) { work() }
	matches.Notifier = notifications

	return &notificationEnv{tx: tx, sender: sender, notifications: notifications, matches: matches, groupID: group.ID, teamA: teams[0], teamB: teams[1]}
}

// member creates a group member; withFeature also enables push for them and
// subscribes one device, whose endpoint is returned.
func (e *notificationEnv) member(t *testing.T, name string, withFeature bool) (uuid.UUID, string) {
	t.Helper()
	id, err := services.NewPlayerService(e.tx).CreatePlayer("Zzz Push " + name + " " + uuid.NewString())
	if err != nil {
		t.Fatalf("CreatePlayer: %v", err)
	}
	if err := services.NewGroupMembershipService(e.tx).AddPlayerToGroup(e.groupID, id); err != nil {
		t.Fatalf("AddPlayerToGroup: %v", err)
	}
	endpoint := "https://push.test/" + id.String()
	if withFeature {
		if err := e.tx.Create(&models.PlayerFeature{PlayerID: id, Feature: models.FeaturePushNotifications}).Error; err != nil {
			t.Fatalf("enable feature: %v", err)
		}
		if err := e.notifications.Subscribe(id, subscription(endpoint)); err != nil {
			t.Fatalf("Subscribe: %v", err)
		}
	}
	return id, endpoint
}

func subscription(endpoint string) services.PushSubscriptionInput {
	var input services.PushSubscriptionInput
	input.Endpoint = endpoint
	input.Keys.P256dh = "p256dh-key"
	input.Keys.Auth = "auth-key"
	return input
}

func (e *notificationEnv) scheduledMatch(t *testing.T, kickoff, opensAt time.Time, maxPlayers int) uuid.UUID {
	t.Helper()
	id, err := e.matches.CreateMatch(services.MatchSpec{
		Date:                models.DateOf(kickoff),
		ScheduledAt:         &kickoff,
		RegistrationOpensAt: &opensAt,
		MaxPlayers:          &maxPlayers,
	}, e.groupID)
	if err != nil {
		t.Fatalf("CreateMatch: %v", err)
	}
	return id
}

func (e *notificationEnv) types(endpoint string) []string {
	var types []string
	for _, message := range e.sender.sent[endpoint] {
		types = append(types, message.Type)
	}
	sort.Strings(types)
	return types
}

func TestNotifications_Integration_SubscribeRequiresFeature(t *testing.T) {
	env := newNotificationEnv(t)
	playerID, endpoint := env.member(t, "NoFeature", false)

	if err := env.notifications.Subscribe(playerID, subscription(endpoint)); !errors.Is(err, services.ErrPushFeatureNotEnabled) {
		t.Errorf("Subscribe without the feature = %v, want ErrPushFeatureNotEnabled", err)
	}
	if err := env.notifications.Subscribe(playerID, services.PushSubscriptionInput{Endpoint: endpoint}); !errors.Is(err, services.ErrInvalidPushSubscription) {
		t.Errorf("Subscribe without keys = %v, want ErrInvalidPushSubscription", err)
	}
}

func TestNotifications_Integration_SameDeviceMovesToTheLatestAccount(t *testing.T) {
	env := newNotificationEnv(t)
	first, endpoint := env.member(t, "First", true)
	second, _ := env.member(t, "Second", true)

	if err := env.notifications.Subscribe(second, subscription(endpoint)); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	var rows []models.PushSubscription
	env.tx.Where("endpoint = ?", endpoint).Find(&rows)
	if len(rows) != 1 || rows[0].PlayerID != second {
		t.Errorf("after a second account subscribed the same device: %d rows, owner %v; want 1 row owned by %v (not %v)", len(rows), rows, second, first)
	}

	if err := env.notifications.Unsubscribe(second, endpoint); err != nil {
		t.Fatalf("Unsubscribe: %v", err)
	}
	env.tx.Where("endpoint = ?", endpoint).Find(&rows)
	if len(rows) != 0 {
		t.Errorf("Unsubscribe left %d rows", len(rows))
	}
	if err := env.notifications.Unsubscribe(second, endpoint); err != nil {
		t.Errorf("Unsubscribe of a device not subscribed = %v, want a no-op success", err)
	}
}

func TestNotifications_Integration_TeamAssignedThenResult(t *testing.T) {
	env := newNotificationEnv(t)
	alice, aliceEndpoint := env.member(t, "Alice", true)
	bob, bobEndpoint := env.member(t, "Bob", true)
	carol, carolEndpoint := env.member(t, "Carol", false) // on the roster, no feature

	kickoff := time.Now().Add(48 * time.Hour)
	matchID := env.scheduledMatch(t, kickoff, kickoff.Add(-72*time.Hour), 10)

	save := func(teamA, teamB []uuid.UUID) {
		t.Helper()
		roster := func(ids []uuid.UUID) []models.PlayerCustom {
			players := make([]models.PlayerCustom, 0, len(ids))
			for _, id := range ids {
				players = append(players, models.PlayerCustom{ID: id})
			}
			return players
		}
		if err := env.matches.UpdateMatch(matchID, env.groupID, []models.TeamWithPlayers{
			{ID: env.teamA.ID, Players: roster(teamA)},
			{ID: env.teamB.ID, Players: roster(teamB)},
		}); err != nil {
			t.Fatalf("UpdateMatch: %v", err)
		}
	}

	save([]uuid.UUID{alice}, []uuid.UUID{bob, carol})
	if got := env.sender.sent[aliceEndpoint]; len(got) != 1 || got[0].Type != services.PushTypeTeamAssigned || got[0].TeamName != env.teamA.Name {
		t.Fatalf("Alice got %+v, want one team_assigned for %q", got, env.teamA.Name)
	}
	if got := env.sender.sent[bobEndpoint]; len(got) != 1 || got[0].TeamName != env.teamB.Name || got[0].KickoffAt == nil || got[0].URL != services.MatchDeepLink(matchID) {
		t.Fatalf("Bob got %+v, want one team_assigned for %q with kick-off and link", got, env.teamB.Name)
	}
	if len(env.sender.sent[carolEndpoint]) != 0 {
		t.Errorf("Carol, without the feature, got %+v", env.sender.sent[carolEndpoint])
	}

	// Saving again with Bob moved: only Bob hears about it.
	env.sender.reset()
	save([]uuid.UUID{alice, bob}, []uuid.UUID{carol})
	if len(env.sender.sent[aliceEndpoint]) != 0 {
		t.Errorf("Alice, unchanged, got %+v", env.sender.sent[aliceEndpoint])
	}
	if got := env.sender.sent[bobEndpoint]; len(got) != 1 || got[0].TeamName != env.teamA.Name {
		t.Errorf("Bob, moved, got %+v, want one team_assigned for %q", got, env.teamA.Name)
	}

	// Kick-off passes; the admin's first save after it is the result.
	if err := env.tx.Model(&models.Match{}).Where("id = ?", matchID).Update("scheduled_at", time.Now().Add(-2*time.Hour)).Error; err != nil {
		t.Fatalf("move kick-off: %v", err)
	}
	env.sender.reset()
	if err := env.matches.UpdateMatch(matchID, env.groupID, []models.TeamWithPlayers{
		{ID: env.teamA.ID, Players: []models.PlayerCustom{{ID: alice, GoalsScored: 2}, {ID: bob, GoalsScored: 1}}},
		{ID: env.teamB.ID, Players: []models.PlayerCustom{{ID: carol, GoalsScored: 1}}},
	}); err != nil {
		t.Fatalf("UpdateMatch (result): %v", err)
	}
	got := env.sender.sent[aliceEndpoint]
	if len(got) != 1 || got[0].Type != services.PushTypeResultRecorded || got[0].TeamName != env.teamA.Name || len(got[0].Score) != 2 {
		t.Fatalf("Alice got %+v, want one result_recorded with both teams' scores", got)
	}
	goals := map[string]int{}
	for _, line := range got[0].Score {
		goals[line.Team] = line.Goals
	}
	if goals[env.teamA.Name] != 3 || goals[env.teamB.Name] != 1 {
		t.Errorf("score = %+v, want %s 3 – 1 %s", got[0].Score, env.teamA.Name, env.teamB.Name)
	}

	// A later correction of the score is not a new result.
	env.sender.reset()
	save([]uuid.UUID{alice, bob}, []uuid.UUID{carol})
	if len(env.sender.sent) != 0 {
		t.Errorf("a correction after the result sent %+v", env.sender.sent)
	}
}

func TestNotifications_Integration_TickSignUpsOpen(t *testing.T) {
	env := newNotificationEnv(t)
	now := time.Now()
	reader, readerEndpoint := env.member(t, "Reader", true)
	creator, creatorEndpoint := env.member(t, "Creator", true)
	_ = reader

	justOpened := env.scheduledMatch(t, now.Add(72*time.Hour), now.Add(-10*time.Minute), 10)
	if err := env.tx.Create(&models.MatchRegistration{MatchID: justOpened, PlayerID: creator}).Error; err != nil {
		t.Fatalf("register creator: %v", err)
	}
	env.scheduledMatch(t, now.Add(72*time.Hour), now.Add(-7*time.Hour), 10) // opened too long ago
	env.scheduledMatch(t, now.Add(72*time.Hour), now.Add(time.Hour), 10)    // not open yet

	result, err := env.notifications.Tick(now)
	if err != nil {
		t.Fatalf("Tick: %v", err)
	}
	if result.RegistrationOpen != 1 {
		t.Errorf("RegistrationOpen = %d, want 1 (only the list that just opened)", result.RegistrationOpen)
	}
	if got := env.sender.sent[readerEndpoint]; len(got) != 1 || got[0].Type != services.PushTypeRegistrationOpen || got[0].URL != services.MatchDeepLink(justOpened) {
		t.Errorf("member got %+v, want one registration_open for the match that just opened", got)
	}
	if len(env.sender.sent[creatorEndpoint]) != 0 {
		t.Errorf("an already signed-up member got %+v", env.sender.sent[creatorEndpoint])
	}

	env.sender.reset()
	if result, _ := env.notifications.Tick(now.Add(5 * time.Minute)); result.RegistrationOpen != 0 || len(env.sender.sent) != 0 {
		t.Errorf("second tick sent %+v again", env.sender.sent)
	}
}

func TestNotifications_Integration_TickKickoffReminder(t *testing.T) {
	env := newNotificationEnv(t)
	now := time.Now()
	first, firstEndpoint := env.member(t, "First", true)
	second, secondEndpoint := env.member(t, "Second", true)
	registrations := services.NewMatchRegistrationService(env.tx)

	// Not composed, one confirmed place: only the first sign-up is reminded.
	uncomposed := env.scheduledMatch(t, now.Add(20*time.Minute), now.Add(-24*time.Hour), 1)
	for _, id := range []uuid.UUID{first, second} {
		if err := registrations.Register(uncomposed, id); err != nil {
			t.Fatalf("Register: %v", err)
		}
		time.Sleep(2 * time.Millisecond) // distinct sign-up order
	}
	env.scheduledMatch(t, now.Add(2*time.Hour), now.Add(-24*time.Hour), 10) // not due yet

	if _, err := env.notifications.Tick(now); err != nil {
		t.Fatalf("Tick: %v", err)
	}
	if got := env.types(firstEndpoint); len(got) != 1 || got[0] != services.PushTypeKickoffReminder {
		t.Errorf("confirmed player got %v, want one kickoff_reminder", got)
	}
	if got := env.types(secondEndpoint); len(got) != 0 {
		t.Errorf("waiting-list player got %v, want nothing", got)
	}

	// Composed: the roster is reminded, each with their own team.
	composed := env.scheduledMatch(t, now.Add(25*time.Minute), now.Add(-24*time.Hour), 10)
	if err := env.matches.UpdateMatch(composed, env.groupID, []models.TeamWithPlayers{
		{ID: env.teamA.ID, Players: []models.PlayerCustom{{ID: first}}},
		{ID: env.teamB.ID, Players: []models.PlayerCustom{{ID: second}}},
	}); err != nil {
		t.Fatalf("UpdateMatch: %v", err)
	}
	env.sender.reset()
	if result, err := env.notifications.Tick(now.Add(time.Minute)); err != nil || result.KickoffReminders != 1 {
		t.Fatalf("Tick = %+v, %v; want exactly the composed match's reminder", result, err)
	}
	if got := env.sender.sent[secondEndpoint]; len(got) != 1 || got[0].TeamName != env.teamB.Name {
		t.Errorf("roster player got %+v, want a reminder naming %q", got, env.teamB.Name)
	}
}

func TestNotifications_Integration_FeatureRemovedAndGoneSubscriptions(t *testing.T) {
	env := newNotificationEnv(t)
	now := time.Now()
	removed, removedEndpoint := env.member(t, "Removed", true)
	_, goneEndpoint := env.member(t, "Gone", true)
	env.sender.gone[goneEndpoint] = true

	if err := env.tx.Where("player_id = ?", removed).Delete(&models.PlayerFeature{}).Error; err != nil {
		t.Fatalf("remove feature: %v", err)
	}
	env.scheduledMatch(t, now.Add(72*time.Hour), now.Add(-time.Minute), 10)
	if _, err := env.notifications.Tick(now); err != nil {
		t.Fatalf("Tick: %v", err)
	}

	if len(env.sender.sent[removedEndpoint]) != 0 {
		t.Errorf("a player whose feature was removed still got %+v", env.sender.sent[removedEndpoint])
	}
	var count int64
	env.tx.Model(&models.PushSubscription{}).Where("endpoint = ?", goneEndpoint).Count(&count)
	if count != 0 {
		t.Error("a subscription the push service reported gone was kept")
	}
}
