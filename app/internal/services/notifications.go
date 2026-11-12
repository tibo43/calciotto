package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"app/internal/models"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Push notifications: who gets told what, and when.
//
// Four notifications, all for scheduled matches only:
//   - registration_open: the match's sign-up list has opened (every group
//     member not already signed up).
//   - team_assigned: an admin saved a roster that puts the player on a team,
//     or moves them, before kick-off (that player only).
//   - kickoff_reminder: kick-off is at most KickoffReminderLead away (the
//     roster once composed, otherwise the confirmed sign-ups).
//   - result_recorded: an admin saved the score for the first time, so the
//     result is in and Man of the Match voting is open (the roster, who are
//     the only ones allowed to vote).
//
// The two event-driven ones go out from MatchService.UpdateMatch through
// RosterSaved. The two time-driven ones need a clock, and the backend's free
// Koyeb instance sleeps when idle, so an in-process timer would not fire:
// instead an external scheduler (.github/workflows/notifications-tick.yml)
// calls POST /internal/notifications/tick, which runs Tick. Each match records
// when each of those two went out, so however many ticks see it, it is sent
// once, and a tick that runs late still sends what is due.
//
// Everything is gated by the push-notifications hidden feature
// (models.FeaturePushNotifications), checked when subscribing and again on
// every send.
//
// The payload is data, not text: the service worker (frontend/public/sw.js)
// writes the title and body, so that a kick-off time is shown in the device's
// own time zone, the way the rest of the app shows it.

const (
	PushTypeRegistrationOpen = "registration_open"
	PushTypeTeamAssigned     = "team_assigned"
	PushTypeKickoffReminder  = "kickoff_reminder"
	PushTypeResultRecorded   = "result_recorded"
)

// KickoffReminderLead is how long before kick-off the reminder becomes due.
// The tick runs every few minutes (and GitHub's scheduler can run late), so
// the reminder arrives at most this long before kick-off, usually a little
// less.
const KickoffReminderLead = 30 * time.Minute

// RegistrationOpenWindow bounds how late a "sign-ups are open" notification
// may still go out. A list that opened longer ago than this before any tick
// saw it — including every list that was already open when this feature
// shipped — is left alone instead of announced as news.
const RegistrationOpenWindow = 6 * time.Hour

var (
	ErrPushFeatureNotEnabled   = errors.New("push notifications are not enabled for this player")
	ErrInvalidPushSubscription = errors.New("a push subscription needs an endpoint and both keys")
	// ErrPushSubscriptionGone is what a PushSender returns when the push
	// service says the subscription no longer exists (the app was
	// uninstalled, permission was revoked): the row is then deleted.
	ErrPushSubscriptionGone = errors.New("push subscription is no longer valid")
)

// PushMessage is the JSON payload of one notification.
type PushMessage struct {
	Type      string      `json:"type"`
	URL       string      `json:"url"`
	Tag       string      `json:"tag"`
	GroupName string      `json:"groupName,omitempty"`
	KickoffAt *time.Time  `json:"kickoffAt,omitempty"`
	TeamName  string      `json:"teamName,omitempty"`
	Score     []PushScore `json:"score,omitempty"`
}

// PushScore is one team's line of a recorded result.
type PushScore struct {
	Team  string `json:"team"`
	Goals int    `json:"goals"`
}

// PushSubscriptionInput is the browser's PushSubscription.toJSON() shape.
type PushSubscriptionInput struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

// PushSender delivers one encrypted payload to one subscription.
type PushSender interface {
	Send(sub models.PushSubscription, payload []byte) error
}

// WebPushSender sends through the browsers' push services with VAPID.
type WebPushSender struct {
	PublicKey  string
	PrivateKey string
	Subject    string
}

func (w *WebPushSender) Send(sub models.PushSubscription, payload []byte) error {
	resp, err := webpush.SendNotification(payload, &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys:     webpush.Keys{P256dh: sub.P256dh, Auth: sub.Auth},
	}, &webpush.Options{
		Subscriber:      w.Subject,
		VAPIDPublicKey:  w.PublicKey,
		VAPIDPrivateKey: w.PrivateKey,
		// A day: a phone that is off for the evening still gets the
		// result the next morning, but not a reminder from last week.
		TTL:     24 * 60 * 60,
		Urgency: webpush.UrgencyHigh,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		return ErrPushSubscriptionGone
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("push service answered %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

type NotificationService struct {
	DB *gorm.DB
	// Sender is nil when VAPID keys aren't configured: subscriptions are
	// still recorded and every trigger still runs (and marks matches as
	// notified), only nothing is delivered.
	Sender  PushSender
	Matches *MatchService
	// Dispatch runs the work behind RosterSaved. In production it is a
	// goroutine, so an admin saving a roster never waits on push services;
	// tests replace it to run inline.
	Dispatch func(func())
}

func NewNotificationService(db *gorm.DB, sender PushSender, matches *MatchService) *NotificationService {
	return &NotificationService{
		DB:       db,
		Sender:   sender,
		Matches:  matches,
		Dispatch: func(work func()) { go work() },
	}
}

func (s *NotificationService) hasFeature(playerID uuid.UUID) (bool, error) {
	var count int64
	err := s.DB.Model(&models.PlayerFeature{}).
		Where("player_id = ? AND feature = ?", playerID, models.FeaturePushNotifications).
		Count(&count).Error
	return count > 0, err
}

// Subscribe records a device for the player. The same endpoint subscribing
// again (a refreshed subscription, or another account on the same browser)
// replaces the existing row.
func (s *NotificationService) Subscribe(playerID uuid.UUID, input PushSubscriptionInput) error {
	if strings.TrimSpace(input.Endpoint) == "" || input.Keys.P256dh == "" || input.Keys.Auth == "" {
		return ErrInvalidPushSubscription
	}
	enabled, err := s.hasFeature(playerID)
	if err != nil {
		return err
	}
	if !enabled {
		return ErrPushFeatureNotEnabled
	}
	row := models.PushSubscription{PlayerID: playerID, Endpoint: input.Endpoint, P256dh: input.Keys.P256dh, Auth: input.Keys.Auth}
	return s.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "endpoint"}},
		DoUpdates: clause.AssignmentColumns([]string{"player_id", "p256dh", "auth"}),
	}).Create(&row).Error
}

// Unsubscribe forgets one of the player's devices. Not subscribed is a no-op
// success, like withdrawing a Man of the Match vote that was never cast.
func (s *NotificationService) Unsubscribe(playerID uuid.UUID, endpoint string) error {
	return s.DB.Where("player_id = ? AND endpoint = ?", playerID, endpoint).Delete(&models.PushSubscription{}).Error
}

// send delivers each player's message to every device they subscribed, if
// they still have the feature. Failures are logged, never returned: one dead
// phone must not stop the others, nor fail the admin action that triggered it.
func (s *NotificationService) send(messages map[uuid.UUID]PushMessage) {
	if s.Sender == nil || len(messages) == 0 {
		return
	}
	playerIDs := make([]uuid.UUID, 0, len(messages))
	for id := range messages {
		playerIDs = append(playerIDs, id)
	}
	var subs []models.PushSubscription
	if err := s.DB.
		Joins("JOIN player_features ON player_features.player_id = push_subscriptions.player_id AND player_features.feature = ?", models.FeaturePushNotifications).
		Where("push_subscriptions.player_id IN ?", playerIDs).
		Find(&subs).Error; err != nil {
		log.Printf("[push] could not load subscriptions: %v", err)
		return
	}
	for _, sub := range subs {
		payload, err := json.Marshal(messages[sub.PlayerID])
		if err != nil {
			log.Printf("[push] could not encode message: %v", err)
			continue
		}
		if err := s.Sender.Send(sub, payload); err != nil {
			if errors.Is(err, ErrPushSubscriptionGone) {
				s.DB.Delete(&models.PushSubscription{}, "id = ?", sub.ID)
				continue
			}
			log.Printf("[push] delivery to a subscription of player %s failed: %v", sub.PlayerID, err)
		}
	}
}

// sameMessageFor addresses one message to several players.
func sameMessageFor(playerIDs []uuid.UUID, message PushMessage) map[uuid.UUID]PushMessage {
	messages := make(map[uuid.UUID]PushMessage, len(playerIDs))
	for _, id := range playerIDs {
		messages[id] = message
	}
	return messages
}

func pushTag(matchID uuid.UUID, kind string) string {
	return "match-" + matchID.String() + "-" + kind
}

func (s *NotificationService) teamNames(groupID uuid.UUID) (map[uuid.UUID]string, error) {
	var teams []models.Team
	if err := s.DB.Where("group_id = ?", groupID).Find(&teams).Error; err != nil {
		return nil, err
	}
	names := make(map[uuid.UUID]string, len(teams))
	for _, team := range teams {
		names[team.ID] = team.Name
	}
	return names, nil
}

func (s *NotificationService) groupName(groupID uuid.UUID) string {
	var group models.Group
	if err := s.DB.Select("name").First(&group, "id = ?", groupID).Error; err != nil {
		return ""
	}
	return group.Name
}

// RosterSaved implements MatchEventNotifier.
func (s *NotificationService) RosterSaved(event RosterSavedEvent) {
	s.Dispatch(func() {
		if err := s.rosterSaved(event, time.Now()); err != nil {
			log.Printf("[push] roster notifications for match %s failed: %v", event.Match.ID, err)
		}
	})
}

func (s *NotificationService) rosterSaved(event RosterSavedEvent, now time.Time) error {
	match := event.Match
	if !match.IsScheduled() {
		return nil
	}
	if event.ResultRecorded {
		return s.notifyResult(match)
	}
	if !now.Before(*match.ScheduledAt) {
		return nil
	}
	names, err := s.teamNames(match.GroupID)
	if err != nil {
		return err
	}
	group := s.groupName(match.GroupID)
	messages := map[uuid.UUID]PushMessage{}
	for playerID, teamID := range event.After {
		if previous, ok := event.Before[playerID]; ok && previous == teamID {
			continue
		}
		messages[playerID] = PushMessage{
			Type:      PushTypeTeamAssigned,
			URL:       MatchDeepLink(match.ID),
			Tag:       pushTag(match.ID, "team"),
			GroupName: group,
			KickoffAt: match.ScheduledAt,
			TeamName:  names[teamID],
		}
	}
	s.send(messages)
	return nil
}

func (s *NotificationService) notifyResult(match models.Match) error {
	details, err := s.Matches.GetMatchDetailsByID(match.ID, match.GroupID)
	if err != nil {
		return err
	}
	score := make([]PushScore, 0, len(details.Teams))
	for _, team := range details.Teams {
		score = append(score, PushScore{Team: team.Name, Goals: team.Score})
	}
	group := s.groupName(match.GroupID)
	messages := map[uuid.UUID]PushMessage{}
	for _, team := range details.Teams {
		for _, player := range team.Players {
			messages[player.ID] = PushMessage{
				Type:      PushTypeResultRecorded,
				URL:       MatchDeepLink(match.ID),
				Tag:       pushTag(match.ID, "result"),
				GroupName: group,
				TeamName:  team.Name,
				Score:     score,
			}
		}
	}
	s.send(messages)
	return nil
}

// TickResult counts what one Tick sent, for the scheduler's log.
type TickResult struct {
	RegistrationOpen int `json:"registration_open"`
	KickoffReminders int `json:"kickoff_reminders"`
}

// Tick sends every time-driven notification that is due at now.
func (s *NotificationService) Tick(now time.Time) (TickResult, error) {
	var result TickResult

	var opening []models.Match
	if err := s.DB.
		Where("scheduled_at IS NOT NULL AND scheduled_at > ?", now).
		Where("registration_opens_at <= ? AND registration_opens_at > ?", now, now.Add(-RegistrationOpenWindow)).
		Where("registrations_closed_at IS NULL AND registration_open_notified_at IS NULL").
		Find(&opening).Error; err != nil {
		return result, err
	}
	for _, match := range opening {
		claimed, err := s.claim(match.ID, "registration_open_notified_at", now)
		if err != nil {
			return result, err
		}
		if !claimed {
			continue
		}
		if err := s.notifyRegistrationOpen(match); err != nil {
			log.Printf("[push] sign-ups-open notification for match %s failed: %v", match.ID, err)
		}
		result.RegistrationOpen++
	}

	var starting []models.Match
	if err := s.DB.
		Where("scheduled_at > ? AND scheduled_at <= ?", now, now.Add(KickoffReminderLead)).
		Where("kickoff_reminder_notified_at IS NULL").
		Find(&starting).Error; err != nil {
		return result, err
	}
	for _, match := range starting {
		claimed, err := s.claim(match.ID, "kickoff_reminder_notified_at", now)
		if err != nil {
			return result, err
		}
		if !claimed {
			continue
		}
		if err := s.notifyKickoffSoon(match); err != nil {
			log.Printf("[push] kick-off reminder for match %s failed: %v", match.ID, err)
		}
		result.KickoffReminders++
	}
	return result, nil
}

// claim marks a match's notification as sent, and reports whether this call
// is the one that did: two overlapping ticks can both find the match, but
// only one update matches the IS NULL guard, so only one of them sends.
func (s *NotificationService) claim(matchID uuid.UUID, column string, now time.Time) (bool, error) {
	update := s.DB.Model(&models.Match{}).
		Where("id = ? AND "+column+" IS NULL", matchID).
		UpdateColumn(column, now)
	return update.RowsAffected == 1, update.Error
}

func (s *NotificationService) notifyRegistrationOpen(match models.Match) error {
	var memberIDs []uuid.UUID
	if err := s.DB.Model(&models.GroupMembership{}).Where("group_id = ?", match.GroupID).Pluck("player_id", &memberIDs).Error; err != nil {
		return err
	}
	// The match creator may already be on the list ("Sign me up for this
	// match"); there is nothing to announce to them.
	var registeredIDs []uuid.UUID
	if err := s.DB.Model(&models.MatchRegistration{}).Where("match_id = ?", match.ID).Pluck("player_id", &registeredIDs).Error; err != nil {
		return err
	}
	registered := make(map[uuid.UUID]bool, len(registeredIDs))
	for _, id := range registeredIDs {
		registered[id] = true
	}
	recipients := make([]uuid.UUID, 0, len(memberIDs))
	for _, id := range memberIDs {
		if !registered[id] {
			recipients = append(recipients, id)
		}
	}
	s.send(sameMessageFor(recipients, PushMessage{
		Type:      PushTypeRegistrationOpen,
		URL:       MatchDeepLink(match.ID),
		Tag:       pushTag(match.ID, "registration"),
		GroupName: s.groupName(match.GroupID),
		KickoffAt: match.ScheduledAt,
	}))
	return nil
}

func (s *NotificationService) notifyKickoffSoon(match models.Match) error {
	message := PushMessage{
		Type:      PushTypeKickoffReminder,
		URL:       MatchDeepLink(match.ID),
		Tag:       pushTag(match.ID, "kickoff"),
		GroupName: s.groupName(match.GroupID),
		KickoffAt: match.ScheduledAt,
	}

	var roster []models.MatchPlayer
	if err := s.DB.Where("match_id = ?", match.ID).Find(&roster).Error; err != nil {
		return err
	}
	if len(roster) > 0 {
		names, err := s.teamNames(match.GroupID)
		if err != nil {
			return err
		}
		messages := make(map[uuid.UUID]PushMessage, len(roster))
		for _, row := range roster {
			personal := message
			personal.TeamName = names[row.TeamID]
			messages[row.PlayerID] = personal
		}
		s.send(messages)
		return nil
	}

	// Teams not composed yet: remind whoever has a confirmed place.
	entries, err := NewMatchRegistrationService(s.DB).ListRegistrations(match.ID)
	if err != nil {
		return err
	}
	var confirmed []uuid.UUID
	for _, entry := range entries {
		if !entry.IsWaiting {
			confirmed = append(confirmed, entry.PlayerID)
		}
	}
	s.send(sameMessageFor(confirmed, message))
	return nil
}
