package services

import (
	"sync"
	"time"

	"app/internal/models"

	"github.com/google/uuid"
)

// LastSeenThrottle is how stale Player.LastSeenAt may get while the player
// keeps using the app: it is rewritten at most once per interval, so a page
// firing a dozen requests costs one write, not twelve.
const LastSeenThrottle = 5 * time.Minute

// lastSeenCache is the per-process half of the throttle. Its zero value is
// ready to use. It only ever grows by one entry per player who has used the
// app since the process started, which stays small for this app.
type lastSeenCache struct {
	mu sync.Mutex
	at map[uuid.UUID]time.Time
}

func (c *lastSeenCache) fresh(playerID uuid.UUID, now time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	at, ok := c.at[playerID]
	return ok && now.Sub(at) < LastSeenThrottle
}

func (c *lastSeenCache) remember(playerID uuid.UUID, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.at == nil {
		c.at = make(map[uuid.UUID]time.Time)
	}
	c.at[playerID] = now
}

// TouchLastSeen records that playerID used the app at now. Called by
// AuthMiddleware on every authenticated request, it does nothing when this
// process already wrote the value less than LastSeenThrottle ago, and the
// UPDATE itself only matches a value older than that, so several replicas
// don't each rewrite it either. UpdateColumn leaves updated_at alone: being
// seen is not an edit of the player.
func (s *AuthService) TouchLastSeen(playerID uuid.UUID, now time.Time) error {
	if s.lastSeen.fresh(playerID, now) {
		return nil
	}
	err := s.DB.Model(&models.Player{}).
		Where("id = ? AND (last_seen_at IS NULL OR last_seen_at < ?)", playerID, now.Add(-LastSeenThrottle)).
		UpdateColumn("last_seen_at", now).Error
	if err != nil {
		return err
	}
	s.lastSeen.remember(playerID, now)
	return nil
}
