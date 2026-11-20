package services

import (
	"log"
	"time"

	"app/internal/models"

	"github.com/google/uuid"

	// Embeds the zone database: the production image (alpine) ships none,
	// and MotmAnnounceAt needs Europe/Rome.
	_ "time/tzdata"
)

// MotmAnnounceLocation and MotmAnnounceHour place the Man of the Match
// announcement: 09:00 in Rome on the day the vote closes, so it does not
// arrive at midnight, when the vote actually closes (VotingWindowError).
const (
	MotmAnnounceLocation = "Europe/Rome"
	MotmAnnounceHour     = 9
)

// MotmAnnounceWindow bounds how late motm_announced may still go out: a
// match whose announcement time passed longer ago than this before any tick
// sent it is left alone instead of announced as news. With the startup
// backfill (database.AddMotmNotifiedAt), it is what keeps every vote closed
// before this notification existed from being announced.
const MotmAnnounceWindow = 24 * time.Hour

var motmLocation = mustLoadLocation(MotmAnnounceLocation)

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

// MotmAnnounceAt is when the tick announces match's Man of the Match: 09:00
// Europe/Rome (summer or winter time alike) on the calendar day the vote
// closes, which is D+2 for a match played on D (VotingWindowError closes it
// at 00:00 UTC that day, so 09:00 in Rome is always after the close).
func MotmAnnounceAt(match models.Match) time.Time {
	deadline := time.Time(match.Date).AddDate(0, 0, 2)
	return time.Date(deadline.Year(), deadline.Month(), deadline.Day(), MotmAnnounceHour, 0, 0, 0, motmLocation)
}

// motmAnnounceDue reports whether the tick at now should announce match's
// Man of the Match: from MotmAnnounceAt, and for MotmAnnounceWindow after.
func motmAnnounceDue(match models.Match, now time.Time) bool {
	at := MotmAnnounceAt(match)
	return VotingWindowError(match, now) != nil && !now.Before(at) && now.Before(at.Add(MotmAnnounceWindow))
}

// tickMotm announces the Man of the Match of every match due at now (see
// motmAnnounceDue). Candidates are narrowed in SQL by date, two to three
// days back in Rome, then checked exactly.
func (s *NotificationService) tickMotm(now time.Time) (int, error) {
	today := now.In(motmLocation)
	from := time.Date(today.Year(), today.Month(), today.Day()-4, 0, 0, 0, 0, time.UTC)
	to := time.Date(today.Year(), today.Month(), today.Day()-1, 0, 0, 0, 0, time.UTC)
	var closed []models.Match
	if err := s.DB.
		Where("date >= ? AND date <= ? AND motm_notified_at IS NULL", models.Date(from), models.Date(to)).
		Find(&closed).Error; err != nil {
		return 0, err
	}
	sent := 0
	for _, match := range closed {
		if !motmAnnounceDue(match, now) {
			continue
		}
		ok, err := s.notifyMotm(match, now)
		if err != nil {
			log.Printf("[push] man-of-the-match notification for match %s failed: %v", match.ID, err)
			continue
		}
		if ok {
			sent++
		}
	}
	return sent, nil
}

// notifyMotm tells the Man of the Match winners (ComputeMotmWinners, ties
// included) that they won, whether or not they voted, and every other player
// who voted who won. Nobody else is told, and only players with the
// design-v2 feature are (the award is shown as the "Star of the Match",
// which only the v2 screens show; send still applies the push feature and
// the player's preference on top). A match with no votes is claimed and
// sends nothing. The claim comes after the reads, so a failed read
// leaves the match for the next tick, and two racing ticks still send once.
// It reports whether this call sent it.
func (s *NotificationService) notifyMotm(match models.Match, now time.Time) (bool, error) {
	tally, err := NewMatchVoteService(s.DB).tallyForMatch(match.ID)
	if err != nil {
		return false, err
	}
	winnerIDs := ComputeMotmWinners(tally)
	if len(winnerIDs) == 0 {
		_, err := s.claim(match.ID, "motm_notified_at", now)
		return false, err
	}
	var voterIDs []uuid.UUID
	if err := s.DB.Model(&models.MatchVote{}).Where("match_id = ?", match.ID).Pluck("voter_id", &voterIDs).Error; err != nil {
		return false, err
	}
	v2 := map[uuid.UUID]bool{}
	for _, id := range append(append([]uuid.UUID{}, voterIDs...), winnerIDs...) {
		if _, seen := v2[id]; seen {
			continue
		}
		has, err := s.playerHasFeature(id, models.FeatureDesignV2)
		if err != nil {
			return false, err
		}
		v2[id] = has
	}
	claimed, err := s.claim(match.ID, "motm_notified_at", now)
	if err != nil || !claimed {
		return false, err
	}

	// The tally is ordered by votes then name, so tied winners are listed
	// alphabetically.
	won := make(map[uuid.UUID]bool, len(winnerIDs))
	for _, id := range winnerIDs {
		won[id] = true
	}
	var names []string
	for _, line := range tally {
		if won[line.PlayerID] {
			names = append(names, line.Name)
		}
	}
	message := PushMessage{
		Type:    PushTypeMotmAnnounced,
		URL:     MatchDeepLink(match.ID),
		Tag:     pushTag(match.ID, "motm"),
		Winners: names,
	}
	messages := make(map[uuid.UUID]PushMessage, len(voterIDs)+len(winnerIDs))
	for _, id := range voterIDs {
		messages[id] = message
	}
	for _, id := range winnerIDs {
		personal := message
		personal.Winner = true
		messages[id] = personal
	}
	for id := range messages {
		if !v2[id] {
			delete(messages, id)
		}
	}
	s.send(match.GroupID, messages)
	return true, nil
}
