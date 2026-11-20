package services_test

import (
	"encoding/json"
	"errors"
	"math"
	"sort"
	"strings"
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

func (f *fakePushSender) Send(sub models.PushSubscription, payload []byte) (int, error) {
	if f.gone[sub.Endpoint] {
		return 410, services.ErrPushSubscriptionGone
	}
	var message services.PushMessage
	if err := json.Unmarshal(payload, &message); err != nil {
		return 0, err
	}
	f.sent[sub.Endpoint] = append(f.sent[sub.Endpoint], message)
	return 201, nil
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
	groupName     string
	teamA, teamB  models.Team
}

func newNotificationEnv(t *testing.T) *notificationEnv {
	t.Helper()
	return newNotificationEnvOn(t, testutil.BeginTx(t, testutil.OpenDB(t)))
}

func newNotificationEnvOn(t *testing.T, tx *gorm.DB) *notificationEnv {
	t.Helper()

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

	return &notificationEnv{tx: tx, sender: sender, notifications: notifications, matches: matches, groupID: group.ID, groupName: group.Name, teamA: teams[0], teamB: teams[1]}
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
	endpoint := "https://fcm.googleapis.com/fixture/" + id.String()
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

func messagesOfType(sent []services.PushMessage, kind string) []services.PushMessage {
	var out []services.PushMessage
	for _, message := range sent {
		if message.Type == kind {
			out = append(out, message)
		}
	}
	return out
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
	aliceTeam := messagesOfType(env.sender.sent[aliceEndpoint], services.PushTypeTeamAssigned)
	if len(aliceTeam) != 1 || aliceTeam[0].TeamName != env.teamA.Name {
		t.Fatalf("Alice got %+v, want one team_assigned for %q", env.sender.sent[aliceEndpoint], env.teamA.Name)
	}
	// predictions_open is not part of the save any more: the tick sends it
	// PredictionsOpenDelay later (see the PredictionsOpen tests).
	if len(env.sender.sent[aliceEndpoint]) != 1 {
		t.Fatalf("Alice got %+v, want only team_assigned from the save", env.sender.sent[aliceEndpoint])
	}
	bobTeam := messagesOfType(env.sender.sent[bobEndpoint], services.PushTypeTeamAssigned)
	if len(bobTeam) != 1 || bobTeam[0].TeamName != env.teamB.Name || bobTeam[0].KickoffAt == nil || bobTeam[0].URL != services.MatchDeepLink(matchID) {
		t.Fatalf("Bob got %+v, want one team_assigned for %q with kick-off and link", env.sender.sent[bobEndpoint], env.teamB.Name)
	}
	if len(env.sender.sent[bobEndpoint]) != 1 {
		t.Fatalf("Bob got %+v, want only team_assigned from the save", env.sender.sent[bobEndpoint])
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

func TestNotifications_Integration_FormerMemberGetsNoMatchResult(t *testing.T) {
	env := newNotificationEnv(t)
	current, currentEndpoint := env.member(t, "Current player", true)
	former, formerEndpoint := env.member(t, "Former player", true)
	now := time.Now()
	matchID := env.scheduledMatch(t, now.Add(time.Hour), now.Add(-time.Hour), 10)
	roster := []models.TeamWithPlayers{
		{ID: env.teamA.ID, Players: []models.PlayerCustom{{ID: current}}},
		{ID: env.teamB.ID, Players: []models.PlayerCustom{{ID: former}}},
	}
	if err := env.matches.UpdateMatch(matchID, env.groupID, roster); err != nil {
		t.Fatal(err)
	}
	if err := services.NewGroupMembershipService(env.tx).LeaveGroup(env.groupID, former); err != nil {
		t.Fatal(err)
	}
	if err := env.tx.Model(&models.Match{}).Where("id = ?", matchID).Update("scheduled_at", now.Add(-time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	env.sender.reset()
	if err := env.matches.UpdateMatch(matchID, env.groupID, roster); err != nil {
		t.Fatal(err)
	}
	if len(messagesOfType(env.sender.sent[currentEndpoint], services.PushTypeResultRecorded)) != 1 ||
		len(env.sender.sent[formerEndpoint]) != 0 {
		t.Fatal("only current group members should receive the match result")
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

// TestNotifications_Integration_PredictionsOpenOnceThenSettledForPredictors
// covers the two prediction notifications: predictions_open comes from the
// tick once the roster is full and PredictionsOpenDelay has passed since the
// first team_assigned, predictions_settled from the result save; each goes
// out once, and the settled one is only for players who placed a prediction
// — with that prediction's own points.
func TestNotifications_Integration_PredictionsOpenOnceThenSettledForPredictors(t *testing.T) {
	env := newNotificationEnv(t)
	alice, aliceEndpoint := env.member(t, "Alice", true) // roster, predicts the winner
	bob, bobEndpoint := env.member(t, "Bob", true)       // roster, predicts the other team
	_, carolEndpoint := env.member(t, "Carol", true)     // member, never predicts
	eve, eveEndpoint := env.member(t, "Eve", true)       // not on the roster, predicts a draw
	dave, daveEndpoint := env.member(t, "Dave", false)   // predicts, feature off

	kickoff := time.Now().Add(48 * time.Hour)
	matchID := env.scheduledMatch(t, kickoff, kickoff.Add(-72*time.Hour), 10)

	roster := func(ids ...uuid.UUID) []models.PlayerCustom {
		players := make([]models.PlayerCustom, 0, len(ids))
		for _, id := range ids {
			players = append(players, models.PlayerCustom{ID: id})
		}
		return players
	}
	save := func(teams []models.TeamWithPlayers) {
		t.Helper()
		if err := env.matches.UpdateMatch(matchID, env.groupID, teams); err != nil {
			t.Fatalf("UpdateMatch: %v", err)
		}
	}
	notifiedAt := func() *time.Time {
		t.Helper()
		var match models.Match
		if err := env.tx.First(&match, "id = ?", matchID).Error; err != nil {
			t.Fatalf("reload match: %v", err)
		}
		return match.PredictionsOpenNotifiedAt
	}

	teamNotifiedAt := func() *time.Time {
		t.Helper()
		var match models.Match
		if err := env.tx.First(&match, "id = ?", matchID).Error; err != nil {
			t.Fatalf("reload match: %v", err)
		}
		return match.TeamAssignedNotifiedAt
	}
	tick := func(at time.Time) services.TickResult {
		t.Helper()
		result, err := env.notifications.Tick(at)
		if err != nil {
			t.Fatalf("Tick: %v", err)
		}
		return result
	}

	// One team still empty: the save tells Alice her team and starts the
	// clock, but predictions are not open, so even a tick well past the
	// delay sends nothing and claims nothing.
	save([]models.TeamWithPlayers{{ID: env.teamA.ID, Players: roster(alice)}})
	if got := env.types(aliceEndpoint); len(got) != 1 || got[0] != services.PushTypeTeamAssigned {
		t.Fatalf("one-sided roster: Alice got %v, want only team_assigned", got)
	}
	firstTeam := teamNotifiedAt()
	if firstTeam == nil {
		t.Fatal("the first team_assigned did not record team_assigned_notified_at")
	}
	env.sender.reset()
	if result := tick(firstTeam.Add(services.PredictionsOpenDelay + time.Minute)); result.PredictionsOpen != 0 || len(env.sender.sent) != 0 {
		t.Fatalf("one-sided roster announced predictions: %+v %+v", result, env.sender.sent)
	}
	if notifiedAt() != nil {
		t.Fatal("one-sided roster claimed predictions_open")
	}

	// Bob completes the roster. The save tells him his team and nothing
	// else: predictions_open is the tick's, never the save's.
	save([]models.TeamWithPlayers{
		{ID: env.teamA.ID, Players: roster(alice)},
		{ID: env.teamB.ID, Players: roster(bob)},
	})
	if got := env.types(bobEndpoint); len(got) != 1 || got[0] != services.PushTypeTeamAssigned {
		t.Fatalf("Bob got %v from the save, want only team_assigned", got)
	}
	if len(env.sender.sent[aliceEndpoint])+len(env.sender.sent[eveEndpoint])+len(env.sender.sent[carolEndpoint]) != 0 {
		t.Fatalf("the save announced predictions: %+v", env.sender.sent)
	}
	if got := teamNotifiedAt(); got == nil || !got.Equal(*firstTeam) {
		t.Errorf("team_assigned_notified_at moved from %v to %v", firstTeam, got)
	}

	// The delay has long passed and the roster is now full: the next tick
	// tells every member with the feature, whether or not they are playing.
	env.sender.reset()
	sentAt := firstTeam.Add(services.PredictionsOpenDelay + 2*time.Minute)
	if result := tick(sentAt); result.PredictionsOpen != 1 {
		t.Fatalf("tick result %+v, want one predictions_open", result)
	}
	for _, endpoint := range []string{aliceEndpoint, bobEndpoint, eveEndpoint, carolEndpoint} {
		got := env.sender.sent[endpoint]
		if len(got) != 1 || got[0].Type != services.PushTypePredictionsOpen || got[0].URL != services.MatchDeepLink(matchID) || got[0].GroupName != env.groupName || got[0].KickoffAt == nil {
			t.Fatalf("member %s got %+v, want one predictions_open", endpoint, got)
		}
		if got[0].Tag != "match-"+matchID.String()+"-predictions" {
			t.Errorf("predictions_open tag = %q", got[0].Tag)
		}
	}
	if len(env.sender.sent[daveEndpoint]) != 0 {
		t.Errorf("Dave, without the feature, got %+v", env.sender.sent[daveEndpoint])
	}
	firstClaim := notifiedAt()
	if firstClaim == nil || !firstClaim.Equal(sentAt) {
		t.Fatalf("predictions_open_notified_at = %v, want the tick's time %v", firstClaim, sentAt)
	}

	// Nothing sends it again: not a later tick, not the same roster saved
	// again, not a player moved.
	env.sender.reset()
	if result := tick(sentAt.Add(time.Minute)); result.PredictionsOpen != 0 || len(env.sender.sent) != 0 {
		t.Fatalf("a second tick sent %+v %+v", result, env.sender.sent)
	}
	save([]models.TeamWithPlayers{
		{ID: env.teamA.ID, Players: roster(alice)},
		{ID: env.teamB.ID, Players: roster(bob)},
	})
	if len(env.sender.sent) != 0 {
		t.Fatalf("saving the same roster again sent %+v", env.sender.sent)
	}
	tick(sentAt.Add(2 * time.Minute))
	if len(env.sender.sent) != 0 {
		t.Fatalf("a tick after an unchanged save sent %+v", env.sender.sent)
	}
	if second := notifiedAt(); second == nil || !second.Equal(*firstClaim) {
		t.Errorf("predictions_open claim moved from %v to %v", firstClaim, second)
	}

	predict := services.NewMatchPredictionService(env.tx, env.matches)
	if err := predict.Predict(matchID, alice, &env.teamA.ID); err != nil {
		t.Fatalf("Alice predict: %v", err)
	}
	if err := predict.Predict(matchID, bob, &env.teamB.ID); err != nil {
		t.Fatalf("Bob predict: %v", err)
	}
	if err := predict.Predict(matchID, eve, nil); err != nil {
		t.Fatalf("Eve predict draw: %v", err)
	}
	if err := predict.Predict(matchID, dave, &env.teamA.ID); err != nil {
		t.Fatalf("Dave predict: %v", err)
	}

	if err := env.tx.Model(&models.Match{}).Where("id = ?", matchID).Update("scheduled_at", time.Now().Add(-2*time.Hour)).Error; err != nil {
		t.Fatalf("move kick-off: %v", err)
	}
	env.sender.reset()
	save([]models.TeamWithPlayers{
		{ID: env.teamA.ID, Players: []models.PlayerCustom{{ID: alice, GoalsScored: 3}}},
		{ID: env.teamB.ID, Players: []models.PlayerCustom{{ID: bob, GoalsScored: 1}}},
	})

	details, err := env.matches.GetMatchDetailsByID(matchID, env.groupID)
	if err != nil {
		t.Fatalf("GetMatchDetailsByID: %v", err)
	}
	all, err := env.matches.GetMatchesDetails(env.groupID, "")
	if err != nil {
		t.Fatalf("GetMatchesDetails: %v", err)
	}
	byMatch, err := predict.TallyPredictionsForMatches([]uuid.UUID{matchID})
	if err != nil {
		t.Fatalf("TallyPredictionsForMatches: %v", err)
	}
	standings := services.ComputePredictionStandings(all, []models.MatchWithDetails{*details}, byMatch)
	wantPoints := map[uuid.UUID]float64{}
	for _, row := range standings {
		wantPoints[row.PlayerID] = row.Points
	}
	// A fresh group has no history, so both sides are even and a correct
	// team pick is the even-match odds minus one token. A miss, including a
	// draw that did not happen, is −1. Pinning both keeps this from passing
	// on a notification that merely echoes a broken settlement.
	even := services.ComputeMatchOdds(services.EloStartingRating, services.EloStartingRating)
	if math.Abs(wantPoints[alice]-(even.TeamAOdds-1)) > 1e-9 || wantPoints[bob] != -1 || wantPoints[eve] != -1 {
		t.Fatalf("standings points = %+v, want Alice ~%.3f and Bob and Eve −1", wantPoints, even.TeamAOdds-1)
	}

	checkSettled := func(playerID uuid.UUID, endpoint string, alsoResult bool) {
		t.Helper()
		settled := messagesOfType(env.sender.sent[endpoint], services.PushTypePredictionsSettled)
		if len(settled) != 1 || settled[0].Points == nil || math.Abs(*settled[0].Points-wantPoints[playerID]) > 1e-9 {
			t.Fatalf("%s got %+v, want one predictions_settled worth %v", endpoint, env.sender.sent[endpoint], wantPoints[playerID])
		}
		if settled[0].URL != services.MatchDeepLink(matchID) || settled[0].Tag != "match-"+matchID.String()+"-predictions-settled" {
			t.Errorf("%s settled link/tag = %q %q", endpoint, settled[0].URL, settled[0].Tag)
		}
		goals := map[string]int{}
		for _, line := range settled[0].Score {
			goals[line.Team] = line.Goals
		}
		if goals[env.teamA.Name] != 3 || goals[env.teamB.Name] != 1 {
			t.Errorf("%s score = %+v, want %s 3 – %s 1", endpoint, settled[0].Score, env.teamA.Name, env.teamB.Name)
		}
		results := messagesOfType(env.sender.sent[endpoint], services.PushTypeResultRecorded)
		if alsoResult {
			if len(results) != 1 {
				t.Errorf("%s got %d result_recorded, want 1 (they played)", endpoint, len(results))
			}
		} else if len(results) != 0 {
			t.Errorf("%s got a result_recorded without being on the roster: %+v", endpoint, results)
		}
	}
	checkSettled(alice, aliceEndpoint, true)
	checkSettled(bob, bobEndpoint, true)
	checkSettled(eve, eveEndpoint, false)
	if len(env.sender.sent[carolEndpoint]) != 0 {
		t.Errorf("Carol, who did not predict, got %+v", env.sender.sent[carolEndpoint])
	}
	if len(env.sender.sent[daveEndpoint]) != 0 {
		t.Errorf("Dave, who predicted without the feature, got %+v", env.sender.sent[daveEndpoint])
	}

	// Editing the score later changes what the leaderboard derives, and does
	// not send the result or the settlement again.
	env.sender.reset()
	save([]models.TeamWithPlayers{
		{ID: env.teamA.ID, Players: []models.PlayerCustom{{ID: alice, GoalsScored: 4}}},
		{ID: env.teamB.ID, Players: []models.PlayerCustom{{ID: bob}}},
	})
	if len(env.sender.sent) != 0 {
		t.Fatalf("a score correction sent %+v", env.sender.sent)
	}
}

// A correct draw (predicted team nil) earns the draw odds minus one token,
// the same number ComputePredictionStandings records for it.
func TestNotifications_Integration_SettledDrawPaysDrawOdds(t *testing.T) {
	env := newNotificationEnv(t)
	alice, aliceEndpoint := env.member(t, "Alice", true)
	bob, _ := env.member(t, "Bob", true)

	kickoff := time.Now().Add(36 * time.Hour)
	matchID := env.scheduledMatch(t, kickoff, kickoff.Add(-48*time.Hour), 10)
	if err := env.matches.UpdateMatch(matchID, env.groupID, []models.TeamWithPlayers{
		{ID: env.teamA.ID, Players: []models.PlayerCustom{{ID: alice}}},
		{ID: env.teamB.ID, Players: []models.PlayerCustom{{ID: bob}}},
	}); err != nil {
		t.Fatalf("UpdateMatch: %v", err)
	}
	if err := services.NewMatchPredictionService(env.tx, env.matches).Predict(matchID, alice, nil); err != nil {
		t.Fatalf("Predict draw: %v", err)
	}
	if err := env.tx.Model(&models.Match{}).Where("id = ?", matchID).Update("scheduled_at", time.Now().Add(-time.Hour)).Error; err != nil {
		t.Fatalf("move kick-off: %v", err)
	}
	env.sender.reset()
	if err := env.matches.UpdateMatch(matchID, env.groupID, []models.TeamWithPlayers{
		{ID: env.teamA.ID, Players: []models.PlayerCustom{{ID: alice, GoalsScored: 1}}},
		{ID: env.teamB.ID, Players: []models.PlayerCustom{{ID: bob, GoalsScored: 1}}},
	}); err != nil {
		t.Fatalf("UpdateMatch (draw): %v", err)
	}

	settled := messagesOfType(env.sender.sent[aliceEndpoint], services.PushTypePredictionsSettled)
	even := services.ComputeMatchOdds(services.EloStartingRating, services.EloStartingRating)
	if len(settled) != 1 || settled[0].Points == nil || math.Abs(*settled[0].Points-(even.DrawOdds-1)) > 1e-9 {
		t.Fatalf("Alice got %+v, want one predictions_settled worth draw odds − 1 (%.3f)", env.sender.sent[aliceEndpoint], even.DrawOdds-1)
	}
}

func TestNotifications_Integration_UnscheduledSaveSendsNothing(t *testing.T) {
	env := newNotificationEnv(t)
	alice, aliceEndpoint := env.member(t, "Alice", true)
	bob, bobEndpoint := env.member(t, "Bob", true)

	matchID, err := env.matches.CreateMatch(services.MatchSpec{Date: models.DateOf(time.Now())}, env.groupID)
	if err != nil {
		t.Fatalf("CreateMatch: %v", err)
	}
	if err := env.matches.UpdateMatch(matchID, env.groupID, []models.TeamWithPlayers{
		{ID: env.teamA.ID, Players: []models.PlayerCustom{{ID: alice, GoalsScored: 2}}},
		{ID: env.teamB.ID, Players: []models.PlayerCustom{{ID: bob, GoalsScored: 1}}},
	}); err != nil {
		t.Fatalf("UpdateMatch: %v", err)
	}
	if len(env.sender.sent[aliceEndpoint])+len(env.sender.sent[bobEndpoint]) != 0 {
		t.Fatalf("an unscheduled match sent %+v", env.sender.sent)
	}
}

func (e *notificationEnv) prefs(t *testing.T, playerID uuid.UUID, updates map[string]bool) {
	t.Helper()
	if _, err := e.notifications.UpdatePreferences(playerID, updates); err != nil {
		t.Fatalf("UpdatePreferences: %v", err)
	}
}

func (e *notificationEnv) wantTypes(t *testing.T, endpoint string, kinds ...string) {
	t.Helper()
	got := e.types(endpoint)
	sort.Strings(kinds)
	if len(got) != len(kinds) {
		t.Fatalf("%s got %v, want %v", endpoint, got, kinds)
	}
	for i := range kinds {
		if got[i] != kinds[i] {
			t.Fatalf("%s got %v, want %v", endpoint, got, kinds)
		}
	}
}

func wantAllOnExcept(t *testing.T, got map[string]bool, off ...string) {
	t.Helper()
	turnedOff := map[string]bool{}
	for _, kind := range off {
		turnedOff[kind] = true
	}
	if len(got) != len(services.AllPushTypes) {
		t.Fatalf("got %d preferences (%v), want the %d types", len(got), got, len(services.AllPushTypes))
	}
	for _, kind := range services.AllPushTypes {
		if got[kind] == turnedOff[kind] {
			t.Errorf("%s = %v, want %v", kind, got[kind], !turnedOff[kind])
		}
	}
}

func TestNotifications_Integration_PreferencesDefaultAllOn(t *testing.T) {
	env := newNotificationEnv(t)
	playerID, _ := env.member(t, "Chooser", true)
	noFeature, _ := env.member(t, "NoFeature", false)

	got, err := env.notifications.GetPreferences(playerID)
	if err != nil {
		t.Fatalf("GetPreferences: %v", err)
	}
	wantAllOnExcept(t, got)

	if _, err := env.notifications.GetPreferences(noFeature); !errors.Is(err, services.ErrPushFeatureNotEnabled) {
		t.Errorf("GetPreferences without the feature = %v, want ErrPushFeatureNotEnabled", err)
	}
	if _, err := env.notifications.UpdatePreferences(noFeature, map[string]bool{services.PushTypeKickoffReminder: false}); !errors.Is(err, services.ErrPushFeatureNotEnabled) {
		t.Errorf("UpdatePreferences without the feature = %v, want ErrPushFeatureNotEnabled", err)
	}
	if _, err := env.notifications.UpdatePreferences(playerID, map[string]bool{"not_a_type": false}); !errors.Is(err, services.ErrUnknownNotificationType) {
		t.Errorf("unknown type = %v, want ErrUnknownNotificationType", err)
	}
	var stored int64
	if err := env.tx.Model(&models.NotificationPreference{}).Where("player_id = ?", playerID).Count(&stored).Error; err != nil {
		t.Fatalf("count preferences: %v", err)
	}
	if stored != 0 {
		t.Errorf("an unknown type left %d rows", stored)
	}

	got, err = env.notifications.UpdatePreferences(playerID, map[string]bool{
		services.PushTypeKickoffReminder: false,
		services.PushTypeResultRecorded:  false,
	})
	if err != nil {
		t.Fatalf("UpdatePreferences: %v", err)
	}
	wantAllOnExcept(t, got, services.PushTypeKickoffReminder, services.PushTypeResultRecorded)

	// Turning one back on leaves the other off, and a second write of the
	// same value is a no-op success.
	got, err = env.notifications.UpdatePreferences(playerID, map[string]bool{services.PushTypeKickoffReminder: true})
	if err != nil {
		t.Fatalf("UpdatePreferences (back on): %v", err)
	}
	wantAllOnExcept(t, got, services.PushTypeResultRecorded)
	got, err = env.notifications.UpdatePreferences(playerID, map[string]bool{services.PushTypeResultRecorded: false})
	if err != nil {
		t.Fatalf("UpdatePreferences (same value): %v", err)
	}
	wantAllOnExcept(t, got, services.PushTypeResultRecorded)

	if got, err = env.notifications.UpdatePreferences(playerID, map[string]bool{}); err != nil {
		t.Fatalf("empty update: %v", err)
	} else {
		wantAllOnExcept(t, got, services.PushTypeResultRecorded)
	}
}

// TestNotifications_Integration_SendRespectsEachPreference checks the filter
// on every type: the player who turned that one off receives none of it, and
// a player who never set a preference still receives all six. Types that
// share a trigger stay independent — team_assigned off still allows
// predictions_open, and result_recorded off still allows predictions_settled.
func TestNotifications_Integration_SendRespectsEachPreference(t *testing.T) {
	env := newNotificationEnv(t)
	now := time.Now()
	alice, aliceEndpoint := env.member(t, "Alice", true)
	bob, bobEndpoint := env.member(t, "Bob", true) // never sets a preference

	roster := func(ids ...uuid.UUID) []models.PlayerCustom {
		players := make([]models.PlayerCustom, 0, len(ids))
		for _, id := range ids {
			players = append(players, models.PlayerCustom{ID: id})
		}
		return players
	}
	saveRoster := func(matchID uuid.UUID, sideA, sideB []uuid.UUID) {
		t.Helper()
		if err := env.matches.UpdateMatch(matchID, env.groupID, []models.TeamWithPlayers{
			{ID: env.teamA.ID, Players: roster(sideA...)},
			{ID: env.teamB.ID, Players: roster(sideB...)},
		}); err != nil {
			t.Fatalf("UpdateMatch: %v", err)
		}
	}

	// Sign-ups open: only Alice opted out.
	env.scheduledMatch(t, now.Add(72*time.Hour), now.Add(-10*time.Minute), 10)
	env.prefs(t, alice, map[string]bool{services.PushTypeRegistrationOpen: false})
	if _, err := env.notifications.Tick(now); err != nil {
		t.Fatalf("Tick: %v", err)
	}
	env.wantTypes(t, aliceEndpoint)
	env.wantTypes(t, bobEndpoint, services.PushTypeRegistrationOpen)

	// Her team is off, predictions open is on: she hears only the second.
	// Sign-ups opened long enough ago that a later tick does not announce
	// them — this test's second tick is the kick-off reminder.
	env.sender.reset()
	teams := env.scheduledMatch(t, now.Add(48*time.Hour), now.Add(-7*time.Hour), 10)
	env.prefs(t, alice, map[string]bool{
		services.PushTypeRegistrationOpen: true,
		services.PushTypeTeamAssigned:     false,
	})
	saveRoster(teams, []uuid.UUID{alice}, []uuid.UUID{bob})
	if _, err := env.notifications.Tick(time.Now().Add(services.PredictionsOpenDelay + time.Minute)); err != nil {
		t.Fatalf("Tick (predictions open): %v", err)
	}
	env.wantTypes(t, aliceEndpoint, services.PushTypePredictionsOpen)
	env.wantTypes(t, bobEndpoint, services.PushTypeTeamAssigned, services.PushTypePredictionsOpen)

	// The other way round, on a match that has not announced predictions yet.
	env.sender.reset()
	predictions := env.scheduledMatch(t, now.Add(36*time.Hour), now.Add(-7*time.Hour), 10)
	env.prefs(t, alice, map[string]bool{
		services.PushTypeTeamAssigned:    true,
		services.PushTypePredictionsOpen: false,
	})
	saveRoster(predictions, []uuid.UUID{alice}, []uuid.UUID{bob})
	if _, err := env.notifications.Tick(time.Now().Add(services.PredictionsOpenDelay + time.Minute)); err != nil {
		t.Fatalf("Tick (predictions open): %v", err)
	}
	env.wantTypes(t, aliceEndpoint, services.PushTypeTeamAssigned)
	env.wantTypes(t, bobEndpoint, services.PushTypeTeamAssigned, services.PushTypePredictionsOpen)

	// Kick-off reminder, after the roster save's own notifications are cleared.
	env.sender.reset()
	soon := env.scheduledMatch(t, now.Add(20*time.Minute), now.Add(-7*time.Hour), 10)
	saveRoster(soon, []uuid.UUID{alice}, []uuid.UUID{bob})
	env.sender.reset()
	env.prefs(t, alice, map[string]bool{services.PushTypeKickoffReminder: false})
	if _, err := env.notifications.Tick(now); err != nil {
		t.Fatalf("Tick (kick-off): %v", err)
	}
	env.wantTypes(t, aliceEndpoint)
	env.wantTypes(t, bobEndpoint, services.PushTypeKickoffReminder)

	// Final score off, predictions settled on.
	predict := services.NewMatchPredictionService(env.tx, env.matches)
	if err := predict.Predict(teams, alice, &env.teamA.ID); err != nil {
		t.Fatalf("Alice predict: %v", err)
	}
	if err := predict.Predict(teams, bob, &env.teamA.ID); err != nil {
		t.Fatalf("Bob predict: %v", err)
	}
	if err := env.tx.Model(&models.Match{}).Where("id = ?", teams).Update("scheduled_at", now.Add(-2*time.Hour)).Error; err != nil {
		t.Fatalf("move kick-off: %v", err)
	}
	env.sender.reset()
	env.prefs(t, alice, map[string]bool{
		services.PushTypeKickoffReminder:    false,
		services.PushTypeResultRecorded:     false,
		services.PushTypePredictionsSettled: true,
	})
	saveRoster(teams, []uuid.UUID{alice}, []uuid.UUID{bob})
	env.wantTypes(t, aliceEndpoint, services.PushTypePredictionsSettled)
	env.wantTypes(t, bobEndpoint, services.PushTypeResultRecorded, services.PushTypePredictionsSettled)

	// Predictions settled off, final score on. A different match, so the
	// first score save is still the one that sends.
	env.sender.reset()
	scored := env.scheduledMatch(t, now.Add(24*time.Hour), now.Add(-7*time.Hour), 10)
	saveRoster(scored, []uuid.UUID{alice}, []uuid.UUID{bob})
	if err := predict.Predict(scored, alice, &env.teamB.ID); err != nil {
		t.Fatalf("Alice predict (second): %v", err)
	}
	if err := predict.Predict(scored, bob, &env.teamB.ID); err != nil {
		t.Fatalf("Bob predict (second): %v", err)
	}
	if err := env.tx.Model(&models.Match{}).Where("id = ?", scored).Update("scheduled_at", now.Add(-time.Hour)).Error; err != nil {
		t.Fatalf("move second kick-off: %v", err)
	}
	env.sender.reset()
	env.prefs(t, alice, map[string]bool{
		services.PushTypeResultRecorded:     true,
		services.PushTypePredictionsSettled: false,
	})
	saveRoster(scored, []uuid.UUID{alice}, []uuid.UUID{bob})
	env.wantTypes(t, aliceEndpoint, services.PushTypeResultRecorded)
	env.wantTypes(t, bobEndpoint, services.PushTypeResultRecorded, services.PushTypePredictionsSettled)
}

// Signing up with a group's invite code tells that group's admins who
// joined, and nobody else, so a code that reached the wrong person shows up
// the moment it is used.
func TestNotifications_Integration_MemberJoinedGoesToAdminsOnly(t *testing.T) {
	env := newNotificationEnv(t)
	admin, adminEndpoint := env.member(t, "Admin", true)
	if err := env.tx.Model(&models.GroupMembership{}).
		Where("group_id = ? AND player_id = ?", env.groupID, admin).
		Update("role", models.RoleAdmin).Error; err != nil {
		t.Fatal(err)
	}
	_, memberEndpoint := env.member(t, "Member", true)

	var group models.Group
	if err := env.tx.First(&group, "id = ?", env.groupID).Error; err != nil {
		t.Fatal(err)
	}
	auth := services.NewAuthService(env.tx, "test-secret")
	auth.Notifier = env.notifications
	newcomer, err := auth.SignupNewPlayer("Zzz Push Newcomer", "push-newcomer@example.test", "fixture-password", group.InviteCode)
	if err != nil {
		t.Fatalf("SignupNewPlayer: %v", err)
	}

	joined := messagesOfType(env.sender.sent[adminEndpoint], services.PushTypeMemberJoined)
	if len(joined) != 1 {
		t.Fatalf("admin received %d member_joined notifications, want 1", len(joined))
	}
	if joined[0].PlayerName != "Zzz Push Newcomer" || joined[0].GroupName != env.groupName ||
		joined[0].URL != "/profile/groups/"+env.groupID.String()+"/settings" {
		t.Errorf("unexpected member_joined payload: %+v", joined[0])
	}
	if joined[0].Tag == "" || !strings.Contains(joined[0].Tag, newcomer.String()) {
		t.Errorf("member_joined tag %q should name the new member", joined[0].Tag)
	}
	if len(env.sender.sent[memberEndpoint]) != 0 {
		t.Error("a plain member was told about the new member")
	}
}

// An admin who turned member_joined off is not told.
func TestNotifications_Integration_MemberJoinedRespectsPreference(t *testing.T) {
	env := newNotificationEnv(t)
	admin, adminEndpoint := env.member(t, "Quiet admin", true)
	if err := env.tx.Model(&models.GroupMembership{}).
		Where("group_id = ? AND player_id = ?", env.groupID, admin).
		Update("role", models.RoleAdmin).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := env.notifications.UpdatePreferences(admin, map[string]bool{services.PushTypeMemberJoined: false}); err != nil {
		t.Fatal(err)
	}
	var group models.Group
	if err := env.tx.First(&group, "id = ?", env.groupID).Error; err != nil {
		t.Fatal(err)
	}
	auth := services.NewAuthService(env.tx, "test-secret")
	auth.Notifier = env.notifications
	if _, err := auth.SignupNewPlayer("Zzz Push Unnoticed", "push-unnoticed@example.test", "fixture-password", group.InviteCode); err != nil {
		t.Fatalf("SignupNewPlayer: %v", err)
	}
	if len(env.sender.sent[adminEndpoint]) != 0 {
		t.Error("an admin who turned member_joined off was notified")
	}
}

// predictionsEnv is a match with Alice and Bob subscribed, plus helpers to
// save a roster, read the match's two bookkeeping columns and tick.
type predictionsEnv struct {
	*notificationEnv
	matchID                    uuid.UUID
	alice, bob                 uuid.UUID
	aliceEndpoint, bobEndpoint string
}

func newPredictionsEnv(t *testing.T, kickoff time.Time) *predictionsEnv {
	t.Helper()
	env := newNotificationEnv(t)
	p := &predictionsEnv{notificationEnv: env}
	p.alice, p.aliceEndpoint = env.member(t, "Alice", true)
	p.bob, p.bobEndpoint = env.member(t, "Bob", true)
	p.matchID = env.scheduledMatch(t, kickoff, kickoff.Add(-72*time.Hour), 10)
	return p
}

func (p *predictionsEnv) save(t *testing.T, sideA, sideB []uuid.UUID) {
	t.Helper()
	roster := func(ids []uuid.UUID) []models.PlayerCustom {
		players := []models.PlayerCustom{}
		for _, id := range ids {
			players = append(players, models.PlayerCustom{ID: id})
		}
		return players
	}
	if err := p.matches.UpdateMatch(p.matchID, p.groupID, []models.TeamWithPlayers{
		{ID: p.teamA.ID, Players: roster(sideA)},
		{ID: p.teamB.ID, Players: roster(sideB)},
	}); err != nil {
		t.Fatalf("UpdateMatch: %v", err)
	}
}

func (p *predictionsEnv) reload(t *testing.T) models.Match {
	t.Helper()
	var match models.Match
	if err := p.tx.First(&match, "id = ?", p.matchID).Error; err != nil {
		t.Fatalf("reload match: %v", err)
	}
	return match
}

func (p *predictionsEnv) tick(t *testing.T, at time.Time) int {
	t.Helper()
	p.sender.reset()
	result, err := p.notifications.Tick(at)
	if err != nil {
		t.Fatalf("Tick: %v", err)
	}
	return result.PredictionsOpen
}

// The ten minutes are counted from the first team_assigned: a tick one
// minute short sends nothing, the first tick at or after it sends once, and
// predictions_open_notified_at is that tick's time.
func TestNotifications_Integration_PredictionsOpenWaitsForTheDelay(t *testing.T) {
	p := newPredictionsEnv(t, time.Now().Add(9*time.Hour))
	p.save(t, []uuid.UUID{p.alice}, []uuid.UUID{p.bob})
	p.wantTypes(t, p.aliceEndpoint, services.PushTypeTeamAssigned)
	p.wantTypes(t, p.bobEndpoint, services.PushTypeTeamAssigned)

	first := p.reload(t).TeamAssignedNotifiedAt
	if first == nil {
		t.Fatal("team_assigned_notified_at not set by the first team_assigned")
	}
	if p.reload(t).PredictionsOpenNotifiedAt != nil {
		t.Fatal("the save claimed predictions_open")
	}

	if sent := p.tick(t, first.Add(services.PredictionsOpenDelay-time.Minute)); sent != 0 || len(p.sender.sent) != 0 {
		t.Fatalf("a tick before the delay sent %d: %+v", sent, p.sender.sent)
	}
	due := first.Add(services.PredictionsOpenDelay)
	if sent := p.tick(t, due); sent != 1 {
		t.Fatalf("the tick at the delay sent %d, want 1", sent)
	}
	p.wantTypes(t, p.aliceEndpoint, services.PushTypePredictionsOpen)
	p.wantTypes(t, p.bobEndpoint, services.PushTypePredictionsOpen)
	if got := p.reload(t).PredictionsOpenNotifiedAt; got == nil || !got.Equal(due) {
		t.Fatalf("predictions_open_notified_at = %v, want %v", got, due)
	}

	if sent := p.tick(t, due.Add(time.Minute)); sent != 0 || len(p.sender.sent) != 0 {
		t.Fatalf("a later tick sent again: %d %+v", sent, p.sender.sent)
	}
}

// team_assigned_notified_at is the first save that told anyone their team:
// a save that places nobody does not set it, and a later move still notifies
// the moved player without changing it.
func TestNotifications_Integration_TeamAssignedNotifiedAtIsSetOnce(t *testing.T) {
	p := newPredictionsEnv(t, time.Now().Add(9*time.Hour))
	p.save(t, nil, nil)
	if got := p.reload(t).TeamAssignedNotifiedAt; got != nil {
		t.Fatalf("an empty roster set team_assigned_notified_at to %v", got)
	}

	p.save(t, []uuid.UUID{p.alice}, []uuid.UUID{p.bob})
	first := p.reload(t).TeamAssignedNotifiedAt
	if first == nil {
		t.Fatal("team_assigned_notified_at not set")
	}

	p.sender.reset()
	p.save(t, []uuid.UUID{p.alice, p.bob}, nil)
	p.wantTypes(t, p.aliceEndpoint)
	p.wantTypes(t, p.bobEndpoint, services.PushTypeTeamAssigned)
	if got := p.reload(t).TeamAssignedNotifiedAt; got == nil || !got.Equal(*first) {
		t.Fatalf("team_assigned_notified_at moved from %v to %v", first, got)
	}
}

// When kick-off is less than the delay after the first team_assigned,
// predictions_open does not wait for it: the first tick that sees a full
// roster sends it, and nothing is sent once kick-off has passed.
func TestNotifications_Integration_PredictionsOpenBeforeAnEarlyKickoff(t *testing.T) {
	kickoff := time.Now().Add(6 * time.Minute)
	p := newPredictionsEnv(t, kickoff)
	p.save(t, []uuid.UUID{p.alice}, nil)
	first := p.reload(t).TeamAssignedNotifiedAt
	if first == nil {
		t.Fatal("team_assigned_notified_at not set")
	}

	// One side empty: not open yet, however early the kick-off.
	if sent := p.tick(t, first.Add(time.Minute)); sent != 0 {
		t.Fatalf("a one-sided roster sent predictions_open")
	}

	p.save(t, []uuid.UUID{p.alice}, []uuid.UUID{p.bob})
	at := first.Add(2 * time.Minute)
	if sent := p.tick(t, at); sent != 1 {
		t.Fatalf("the first tick with a full roster before an early kick-off sent %d, want 1", sent)
	}
	p.wantTypes(t, p.aliceEndpoint, services.PushTypePredictionsOpen)
	if got := p.reload(t).PredictionsOpenNotifiedAt; got == nil || !got.Equal(at) {
		t.Fatalf("predictions_open_notified_at = %v, want %v", got, at)
	}
}

func TestNotifications_Integration_PredictionsOpenNeverAfterKickoff(t *testing.T) {
	kickoff := time.Now().Add(6 * time.Minute)
	p := newPredictionsEnv(t, kickoff)
	p.save(t, []uuid.UUID{p.alice}, []uuid.UUID{p.bob})
	if sent := p.tick(t, kickoff.Add(time.Minute)); sent != 0 || len(p.sender.sent) != 0 {
		t.Fatalf("a tick after kick-off sent predictions_open: %d %+v", sent, p.sender.sent)
	}
	if got := p.reload(t).PredictionsOpenNotifiedAt; got != nil {
		t.Fatalf("predictions_open claimed after kick-off at %v", got)
	}
}
