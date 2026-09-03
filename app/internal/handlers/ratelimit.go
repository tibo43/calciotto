package handlers

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimit answers 429 once a caller (identified by c.ClientIP()) exceeds r
// events per second with the given burst. It exists for the four unauthenticated
// /auth/* routes: nothing else in this codebase lets an unauthenticated caller
// trigger repeated server-side work, so nowhere else needs it. Login is the
// main target (bcrypt slows one guess, but not a script making thousands), and
// signup/forgot-password/reset-password get the same treatment since all three
// are otherwise free to hammer — an unthrottled forgot-password is a mail-quota
// exhaustion vector (see the Brevo free-tier cap in CLAUDE.md) as much as a
// login one is a credential-stuffing vector.
//
// This deliberately does not distinguish "wrong password" from "right
// password" — the limiter runs before the handler even sees the request, so a
// legitimate user mistyping their password a few times in a row shares the
// same budget as an attacker. That is the standard trade-off for an IP-level
// limiter and is why the burst is generous rather than tight.
func RateLimit(r rate.Limit, burst int) gin.HandlerFunc {
	limiter := newIPRateLimiter(r, burst)
	return func(c *gin.Context) {
		if !limiter.allow(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, please try again later"})
			return
		}
		c.Next()
	}
}

// ipRateLimiterIdleTTL is how long an IP's bucket survives with no requests
// before ipRateLimiter.evictStale reclaims it. Sized well past every window
// this file configures (the widest is one hour) so a legitimate caller never
// gets a fresh, unpenalized bucket mid-window just because they paused.
const ipRateLimiterIdleTTL = 2 * time.Hour

// ipRateLimiter hands out one token-bucket limiter per client IP. A plain
// map[string]*rate.Limiter would grow forever on a public endpoint, so each
// entry also tracks when it was last touched and evictStale sweeps the ones
// that have gone idle.
type ipRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*ipBucket
	r        rate.Limit
	burst    int
}

type ipBucket struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func newIPRateLimiter(r rate.Limit, burst int) *ipRateLimiter {
	rl := &ipRateLimiter{
		limiters: make(map[string]*ipBucket),
		r:        r,
		burst:    burst,
	}
	go rl.evictStaleLoop()
	return rl
}

func (rl *ipRateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, ok := rl.limiters[ip]
	if !ok {
		bucket = &ipBucket{limiter: rate.NewLimiter(rl.r, rl.burst)}
		rl.limiters[ip] = bucket
	}
	bucket.lastSeen = time.Now()
	return bucket.limiter.Allow()
}

// evictStaleLoop runs for the lifetime of the process — there is exactly one
// ipRateLimiter per route, created once in main.go, so this is not a leak.
func (rl *ipRateLimiter) evictStaleLoop() {
	ticker := time.NewTicker(ipRateLimiterIdleTTL)
	defer ticker.Stop()
	for range ticker.C {
		rl.evictStale()
	}
}

func (rl *ipRateLimiter) evictStale() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	for ip, bucket := range rl.limiters {
		if time.Since(bucket.lastSeen) > ipRateLimiterIdleTTL {
			delete(rl.limiters, ip)
		}
	}
}
