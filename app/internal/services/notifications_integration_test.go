package services_test

import (
	"encoding/json"
	"errors"
	"math"
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
	groupName     string
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
	// Both teams have a player, so predictions open in the same save, for
	// every member who can receive a push — alongside the team notification,
	// not instead of it.
	aliceOpen := messagesOfType(env.sender.sent[aliceEndpoint], services.PushTypePredictionsOpen)
	if len(aliceOpen) != 1 || aliceOpen[0].URL != services.MatchDeepLink(matchID) || aliceOpen[0].GroupName != env.groupName || aliceOpen[0].KickoffAt == nil {
		t.Fatalf("Alice got %+v, want one predictions_open for the match", env.sender.sent[aliceEndpoint])
	}
	if len(env.sender.sent[aliceEndpoint]) != 2 {
		t.Fatalf("Alice got %+v, want exactly team_assigned and predictions_open", env.sender.sent[aliceEndpoint])
	}
	bobTeam := messagesOfType(env.sender.sent[bobEndpoint], services.PushTypeTeamAssigned)
	if len(bobTeam) != 1 || bobTeam[0].TeamName != env.teamB.Name || bobTeam[0].KickoffAt == nil || bobTeam[0].URL != services.MatchDeepLink(matchID) {
		t.Fatalf("Bob got %+v, want one team_assigned for %q with kick-off and link", env.sender.sent[bobEndpoint], env.teamB.Name)
	}
	if len(messagesOfType(env.sender.sent[bobEndpoint], services.PushTypePredictionsOpen)) != 1 {
		t.Fatalf("Bob got %+v, want predictions_open alongside the team", env.sender.sent[bobEndpoint])
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

// TestNotifications_Integration_PredictionsOpenOnceThenSettledForPredictors
// covers the two prediction notifications: they share the roster-save
// trigger with team_assigned and result_recorded, they go out once, and the
// settled one is only for players who placed a prediction — with that
// prediction's own points.
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

	// One team still empty: predictions are not open, and nothing is claimed,
	// so the save that completes the roster can still announce them.
	save([]models.TeamWithPlayers{{ID: env.teamA.ID, Players: roster(alice)}})
	if got := env.types(aliceEndpoint); len(got) != 1 || got[0] != services.PushTypeTeamAssigned {
		t.Fatalf("one-sided roster: Alice got %v, want only team_assigned", got)
	}
	if len(env.sender.sent[eveEndpoint])+len(env.sender.sent[carolEndpoint]) != 0 {
		t.Fatalf("one-sided roster announced predictions: eve %+v carol %+v", env.sender.sent[eveEndpoint], env.sender.sent[carolEndpoint])
	}
	if notifiedAt() != nil {
		t.Fatal("one-sided roster claimed predictions_open")
	}

	env.sender.reset()
	save([]models.TeamWithPlayers{
		{ID: env.teamA.ID, Players: roster(alice)},
		{ID: env.teamB.ID, Players: roster(bob)},
	})
	// Alice was already on her team, so she hears only that predictions
	// opened. Bob was placed, so he hears both. Eve is not on the roster and
	// Carol is not in this save either: every member with the feature gets
	// predictions_open, whether or not they are playing.
	if got := env.types(aliceEndpoint); len(got) != 1 || got[0] != services.PushTypePredictionsOpen {
		t.Fatalf("Alice, already placed, got %v, want only predictions_open", got)
	}
	if got := env.types(bobEndpoint); len(got) != 2 || got[0] != services.PushTypePredictionsOpen || got[1] != services.PushTypeTeamAssigned {
		t.Fatalf("Bob got %v, want predictions_open and team_assigned", got)
	}
	for _, endpoint := range []string{eveEndpoint, carolEndpoint} {
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
	if firstClaim == nil {
		t.Fatal("composed roster did not claim predictions_open")
	}

	// Same roster again: nobody moved, and predictions were already announced.
	env.sender.reset()
	save([]models.TeamWithPlayers{
		{ID: env.teamA.ID, Players: roster(alice)},
		{ID: env.teamB.ID, Players: roster(bob)},
	})
	if len(env.sender.sent) != 0 {
		t.Fatalf("saving the same roster again sent %+v", env.sender.sent)
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
