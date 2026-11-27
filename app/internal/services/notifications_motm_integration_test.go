package services_test

import (
	"testing"
	"time"

	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"
	"app/pkg/database"

	"github.com/google/uuid"
)

// The announcement is at 09:00 in Rome on D+2, whichever of winter (UTC+1)
// or summer (UTC+2) time applies that day, including the day the clocks go
// forward.
func TestMotmAnnounceAt(t *testing.T) {
	for _, tc := range []struct {
		played time.Time
		want   time.Time
	}{
		{time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC), time.Date(2026, time.January, 12, 8, 0, 0, 0, time.UTC)},
		{time.Date(2026, time.July, 10, 0, 0, 0, 0, time.UTC), time.Date(2026, time.July, 12, 7, 0, 0, 0, time.UTC)},
		// Clocks go forward on Sunday March 29th 2026 at 02:00.
		{time.Date(2026, time.March, 27, 0, 0, 0, 0, time.UTC), time.Date(2026, time.March, 29, 7, 0, 0, 0, time.UTC)},
		// And back on Sunday October 25th 2026 at 03:00.
		{time.Date(2026, time.October, 23, 0, 0, 0, 0, time.UTC), time.Date(2026, time.October, 25, 8, 0, 0, 0, time.UTC)},
		// Month and year boundaries.
		{time.Date(2026, time.December, 30, 0, 0, 0, 0, time.UTC), time.Date(2027, time.January, 1, 8, 0, 0, 0, time.UTC)},
	} {
		got := services.MotmAnnounceAt(models.Match{Date: models.Date(tc.played)})
		if !got.Equal(tc.want) {
			t.Errorf("played %s: MotmAnnounceAt = %s, want %s", tc.played.Format("2006-01-02"), got.UTC(), tc.want)
		}
		if services.VotingWindowError(models.Match{Date: models.Date(tc.played)}, got) == nil {
			t.Errorf("played %s: the vote is still open at the announcement", tc.played.Format("2006-01-02"))
		}
	}
}

// A scheduled match is announced the moment its vote closes, 24 hours after
// kick-off, whatever the time of day.
func TestMotmAnnounceAt_Scheduled(t *testing.T) {
	rome, err := time.LoadLocation("Europe/Rome")
	if err != nil {
		t.Fatal(err)
	}
	for _, kickoff := range []time.Time{
		time.Date(2026, time.November, 26, 19, 45, 0, 0, rome),
		time.Date(2026, time.November, 26, 8, 0, 0, 0, rome),
		time.Date(2026, time.March, 28, 21, 0, 0, 0, rome), // across the switch to summer time
	} {
		match := models.Match{Date: models.DateOf(kickoff), ScheduledAt: &kickoff}
		got := services.MotmAnnounceAt(match)
		if want := kickoff.Add(24 * time.Hour); !got.Equal(want) {
			t.Errorf("kick-off %s: MotmAnnounceAt = %s, want %s", kickoff, got, want)
		}
		if services.VotingWindowError(match, got) == nil {
			t.Errorf("kick-off %s: the vote is still open at the announcement", kickoff)
		}
	}
}

// motmEnv is a played match with five subscribed members on its roster,
// whose votes are written directly so the match can be dated in the past.
type motmEnv struct {
	*notificationEnv
	matchID                 uuid.UUID
	alice, bob, carol, dave uuid.UUID
	eve                     uuid.UUID
	aliceEP, bobEP, carolEP string
	daveEP, eveEP           string
	aliceName, bobName      string
}

func newMotmEnv(t *testing.T, played time.Time) *motmEnv {
	t.Helper()
	env := newNotificationEnv(t)
	m := &motmEnv{notificationEnv: env}
	m.alice, m.aliceEP = env.member(t, "Alice", true)
	m.bob, m.bobEP = env.member(t, "Bob", true)
	m.carol, m.carolEP = env.member(t, "Carol", true)
	m.dave, m.daveEP = env.member(t, "Dave", true)
	m.eve, m.eveEP = env.member(t, "Eve", true)
	m.matchID = m.playedMatch(t, played)
	m.aliceName = m.name(t, m.alice)
	m.bobName = m.name(t, m.bob)
	return m
}

func (m *motmEnv) playedMatch(t *testing.T, played time.Time) uuid.UUID {
	t.Helper()
	id, err := m.matches.CreateMatch(services.MatchSpec{Date: models.Date(played)}, m.groupID)
	if err != nil {
		t.Fatalf("CreateMatch: %v", err)
	}
	if err := m.matches.UpdateMatch(id, m.groupID, []models.TeamWithPlayers{
		{ID: m.teamA.ID, Players: []models.PlayerCustom{{ID: m.alice}, {ID: m.bob}, {ID: m.eve}}},
		{ID: m.teamB.ID, Players: []models.PlayerCustom{{ID: m.carol}, {ID: m.dave}}},
	}); err != nil {
		t.Fatalf("UpdateMatch: %v", err)
	}
	return id
}

func (m *motmEnv) name(t *testing.T, id uuid.UUID) string {
	t.Helper()
	var player models.Player
	if err := m.tx.First(&player, "id = ?", id).Error; err != nil {
		t.Fatalf("load player: %v", err)
	}
	return player.Name
}

func (m *motmEnv) vote(t *testing.T, voter, votedFor uuid.UUID) {
	t.Helper()
	if err := m.tx.Create(&models.MatchVote{MatchID: m.matchID, VoterID: voter, VotedForID: votedFor}).Error; err != nil {
		t.Fatalf("vote: %v", err)
	}
}

func (m *motmEnv) tick(t *testing.T, at time.Time) int {
	t.Helper()
	m.sender.reset()
	result, err := m.notifications.Tick(at)
	if err != nil {
		t.Fatalf("Tick: %v", err)
	}
	return result.MotmAnnounced
}

func (m *motmEnv) notifiedAt(t *testing.T) *time.Time {
	t.Helper()
	var match models.Match
	if err := m.tx.First(&match, "id = ?", m.matchID).Error; err != nil {
		t.Fatalf("reload match: %v", err)
	}
	return match.MotmNotifiedAt
}

func (m *motmEnv) announceAt(t *testing.T) time.Time {
	t.Helper()
	var match models.Match
	if err := m.tx.First(&match, "id = ?", m.matchID).Error; err != nil {
		t.Fatalf("reload match: %v", err)
	}
	return services.MotmAnnounceAt(match)
}

// wantMotm checks endpoint got exactly one motm_announced, as a winner or a
// voter, naming winners.
func (m *motmEnv) wantMotm(t *testing.T, endpoint string, winner bool, winners ...string) {
	t.Helper()
	got := m.sender.sent[endpoint]
	if len(got) != 1 || got[0].Type != services.PushTypeMotmAnnounced {
		t.Fatalf("%s got %+v, want one motm_announced", endpoint, got)
	}
	if got[0].Winner != winner {
		t.Errorf("%s winner = %v, want %v", endpoint, got[0].Winner, winner)
	}
	if len(got[0].Winners) != len(winners) {
		t.Fatalf("%s winners = %v, want %v", endpoint, got[0].Winners, winners)
	}
	for i := range winners {
		if got[0].Winners[i] != winners[i] {
			t.Errorf("%s winners = %v, want %v", endpoint, got[0].Winners, winners)
		}
	}
	if got[0].URL != services.MatchDeepLink(m.matchID) || got[0].Tag != "match-"+m.matchID.String()+"-motm" {
		t.Errorf("%s link/tag = %q %q", endpoint, got[0].URL, got[0].Tag)
	}
}

// Winter: 09:00 in Rome is 08:00 UTC. Nothing before it, once at it: the
// winner (who also voted) gets only the winner message, every other voter
// the voter message, and a player who was voted for or played without
// voting nothing. Never again.
func TestNotifications_Integration_MotmAnnouncedOnceAt9InRome(t *testing.T) {
	m := newMotmEnv(t, time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC))
	m.vote(t, m.bob, m.alice)
	m.vote(t, m.carol, m.alice)
	m.vote(t, m.alice, m.dave)

	at := m.announceAt(t)
	if want := time.Date(2026, time.January, 12, 8, 0, 0, 0, time.UTC); !at.Equal(want) {
		t.Fatalf("announce at %s, want %s", at.UTC(), want)
	}
	for _, early := range []time.Time{time.Date(2026, time.January, 12, 0, 0, 0, 0, time.UTC), at.Add(-time.Minute)} {
		if sent := m.tick(t, early); sent != 0 || len(m.sender.sent) != 0 {
			t.Fatalf("a tick at %s, before 09:00 in Rome, sent %d: %+v", early, sent, m.sender.sent)
		}
	}

	if sent := m.tick(t, at); sent != 1 {
		t.Fatalf("the tick at 09:00 sent %d, want 1", sent)
	}
	m.wantMotm(t, m.aliceEP, true, m.aliceName)
	m.wantMotm(t, m.bobEP, false, m.aliceName)
	m.wantMotm(t, m.carolEP, false, m.aliceName)
	if len(m.sender.sent[m.daveEP])+len(m.sender.sent[m.eveEP]) != 0 {
		t.Errorf("a player who did not vote and did not win was told: %+v %+v", m.sender.sent[m.daveEP], m.sender.sent[m.eveEP])
	}
	if got := m.notifiedAt(t); got == nil || !got.Equal(at) {
		t.Fatalf("motm_notified_at = %v, want %v", got, at)
	}

	if sent := m.tick(t, at.Add(time.Minute)); sent != 0 || len(m.sender.sent) != 0 {
		t.Fatalf("a later tick sent again: %d %+v", sent, m.sender.sent)
	}
}

// Summer: 09:00 in Rome is 07:00 UTC; 08:59 in Rome is still too early.
func TestNotifications_Integration_MotmAnnouncedSummerTime(t *testing.T) {
	m := newMotmEnv(t, time.Date(2026, time.July, 10, 0, 0, 0, 0, time.UTC))
	m.vote(t, m.bob, m.alice)

	if sent := m.tick(t, time.Date(2026, time.July, 12, 6, 59, 0, 0, time.UTC)); sent != 0 {
		t.Fatalf("08:59 in Rome sent %d", sent)
	}
	if sent := m.tick(t, time.Date(2026, time.July, 12, 7, 0, 0, 0, time.UTC)); sent != 1 {
		t.Fatalf("09:00 in Rome sent %d, want 1", sent)
	}
	m.wantMotm(t, m.aliceEP, true, m.aliceName)
	m.wantMotm(t, m.bobEP, false, m.aliceName)
}

// A tie makes every tied player a winner, and the voters are told every
// name, in alphabetical order.
func TestNotifications_Integration_MotmAnnouncedTie(t *testing.T) {
	m := newMotmEnv(t, time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC))
	m.vote(t, m.bob, m.alice)
	m.vote(t, m.alice, m.bob)
	m.vote(t, m.carol, m.alice)
	m.vote(t, m.dave, m.bob)

	if sent := m.tick(t, m.announceAt(t)); sent != 1 {
		t.Fatalf("the tick sent %d, want 1", sent)
	}
	names := []string{m.aliceName, m.bobName} // "Zzz Push Alice …" < "Zzz Push Bob …"
	m.wantMotm(t, m.aliceEP, true, names...)
	m.wantMotm(t, m.bobEP, true, names...)
	m.wantMotm(t, m.carolEP, false, names...)
	m.wantMotm(t, m.daveEP, false, names...)
	if len(m.sender.sent[m.eveEP]) != 0 {
		t.Errorf("Eve, who did not vote, got %+v", m.sender.sent[m.eveEP])
	}
}

// No votes: nothing to announce, and the match is marked done.
func TestNotifications_Integration_MotmNoVotes(t *testing.T) {
	m := newMotmEnv(t, time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC))
	at := m.announceAt(t)
	if sent := m.tick(t, at); sent != 0 || len(m.sender.sent) != 0 {
		t.Fatalf("a match nobody voted on sent %d: %+v", sent, m.sender.sent)
	}
	if got := m.notifiedAt(t); got == nil || !got.Equal(at) {
		t.Fatalf("motm_notified_at = %v, want %v (done, nothing sent)", got, at)
	}
}

// A winner who turned motm_announced off is not told; the voters still are,
// and the match is claimed all the same.
func TestNotifications_Integration_MotmRespectsPreference(t *testing.T) {
	m := newMotmEnv(t, time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC))
	m.vote(t, m.bob, m.alice)
	m.prefs(t, m.alice, map[string]bool{services.PushTypeMotmAnnounced: false})

	at := m.announceAt(t)
	if sent := m.tick(t, at); sent != 1 {
		t.Fatalf("the tick sent %d, want 1", sent)
	}
	m.wantTypes(t, m.aliceEP)
	m.wantMotm(t, m.bobEP, false, m.aliceName)
	if m.notifiedAt(t) == nil {
		t.Fatal("the match was not claimed")
	}
}

// Past MotmAnnounceWindow after its time, a vote is never announced, and a
// match whose vote closed days ago is not either.
func TestNotifications_Integration_MotmNeverAfterTheWindow(t *testing.T) {
	m := newMotmEnv(t, time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC))
	m.vote(t, m.bob, m.alice)
	at := m.announceAt(t)
	if sent := m.tick(t, at.Add(services.MotmAnnounceWindow)); sent != 0 || len(m.sender.sent) != 0 {
		t.Fatalf("a tick past the window sent %d: %+v", sent, m.sender.sent)
	}
	if sent := m.tick(t, at.Add(10*24*time.Hour)); sent != 0 || len(m.sender.sent) != 0 {
		t.Fatalf("a tick days later sent %d: %+v", sent, m.sender.sent)
	}
	if got := m.notifiedAt(t); got != nil {
		t.Fatalf("a tick past the window claimed the match at %v", got)
	}
	if sent := m.tick(t, at.Add(services.MotmAnnounceWindow-time.Minute)); sent != 1 {
		t.Fatalf("the last tick inside the window sent %d, want 1", sent)
	}
}

// TestAddMotmNotifiedAt_Integration replays the deploy that adds the column:
// a match whose announcement time has passed is marked announced at that
// time, so the first tick sends nothing for it, even inside the window; a
// match whose vote is still open keeps NULL and is announced when due. A
// later start, the column present, backfills nothing.
//
// After the column is dropped and added back it is the table's last one, so
// a statement prepared on matches earlier in this run would fail with
// "cached plan must not change result type": the reads below are raw
// statements of their own, and the rollback restores the original table.
func TestAddMotmNotifiedAt_Integration(t *testing.T) {
	now := time.Now()
	tx := testutil.BeginTx(t, testutil.OpenDB(t))
	env := newNotificationEnvOn(t, tx)
	m := &motmEnv{notificationEnv: env}
	m.alice, m.aliceEP = env.member(t, "Alice", true)
	m.bob, m.bobEP = env.member(t, "Bob", true)
	m.carol, _ = env.member(t, "Carol", true)
	m.dave, _ = env.member(t, "Dave", true)
	m.eve, _ = env.member(t, "Eve", true)
	// Played three days ago: announced yesterday at 09:00 in Rome, inside
	// the window, so only the backfill keeps it quiet.
	closed := m.playedMatch(t, now.AddDate(0, 0, -3))
	m.matchID = closed
	m.vote(t, m.bob, m.alice)
	open := m.playedMatch(t, now)

	notifiedAt := func(matchID uuid.UUID) *time.Time {
		t.Helper()
		var at []time.Time
		if err := tx.Raw(`SELECT motm_notified_at AS at FROM matches WHERE id = ? AND motm_notified_at IS NOT NULL /* motm backfill test */`, matchID).Pluck("at", &at).Error; err != nil {
			t.Fatalf("read motm_notified_at: %v", err)
		}
		if len(at) == 0 {
			return nil
		}
		return &at[0]
	}
	var played models.Match
	if err := tx.First(&played, "id = ?", closed).Error; err != nil {
		t.Fatalf("reload match: %v", err)
	}
	wantAt := services.MotmAnnounceAt(played)

	if err := tx.Exec(`ALTER TABLE matches DROP COLUMN motm_notified_at`).Error; err != nil {
		t.Fatalf("drop column: %v", err)
	}
	if err := database.AddMotmNotifiedAt(tx); err != nil {
		t.Fatalf("AddMotmNotifiedAt: %v", err)
	}
	if got := notifiedAt(closed); got == nil || !got.Equal(wantAt) {
		t.Fatalf("closed vote: motm_notified_at = %v, want its announcement time %v", got, wantAt)
	}
	if got := notifiedAt(open); got != nil {
		t.Fatalf("open vote backfilled to %v", got)
	}

	if err := tx.Exec(`UPDATE matches SET motm_notified_at = NULL WHERE id = ?`, closed).Error; err != nil {
		t.Fatalf("clear column: %v", err)
	}
	if err := database.AddMotmNotifiedAt(tx); err != nil {
		t.Fatalf("AddMotmNotifiedAt (second start): %v", err)
	}
	if got := notifiedAt(closed); got != nil {
		t.Fatalf("a later start backfilled motm_notified_at to %v", got)
	}
}
