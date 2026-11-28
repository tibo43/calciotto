package handlers

import (
	"net"
	"net/http"
	"strings"
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
			c.AbortWithStatusJSON(http.StatusTooManyRequests, messageBody("too many requests, please try again later"))
			return
		}
		c.Next()
	}
}

// proxyPeerFactor is how much more a single TCP peer may send than a single
// client through ProxiedRateLimit. See there.
const proxyPeerFactor = 6

// ProxiedRateLimit is RateLimit for the /auth/* routes the frontend reaches
// through Vercel's /api/auth rewrite (frontend/vercel.json), which is what
// makes the session cookie first-party. Behind that rewrite c.ClientIP() is
// Vercel's outgoing address, not the player's, so a plain RateLimit would put
// every player in one bucket per Vercel IP: ten bad logins from anyone would
// lock everyone out.
//
// The client is instead keyed on the X-Forwarded-For hop just left of
// c.ClientIP(), which is the address Vercel saw the request come from. That
// hop is only as trustworthy as the peer: a caller reaching Koyeb directly can
// put anything there. So the peer itself (c.ClientIP()) also gets a bucket,
// proxyPeerFactor times larger: wide enough for every player behind one Vercel
// IP, narrow enough that forging a new hop per request buys a direct caller
// only a bounded multiple of the per-client budget. Without a forwarded hop,
// both keys are the peer and this behaves like RateLimit.
func ProxiedRateLimit(r rate.Limit, burst int) gin.HandlerFunc {
	perClient := newIPRateLimiter(r, burst)
	perPeer := newIPRateLimiter(r*proxyPeerFactor, burst*proxyPeerFactor)
	return func(c *gin.Context) {
		peer := c.ClientIP()
		if !perPeer.allow(peer) || !perClient.allow(ProxiedClientKey(c)) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, messageBody("too many requests, please try again later"))
			return
		}
		c.Next()
	}
}

// ProxiedClientKey is the address ProxiedRateLimit keys a client on: the
// X-Forwarded-For hop just left of c.ClientIP(), or c.ClientIP() itself when
// there is none.
func ProxiedClientKey(c *gin.Context) string {
	peer := c.ClientIP()
	var hops []string
	for _, header := range c.Request.Header.Values("X-Forwarded-For") {
		for _, hop := range strings.Split(header, ",") {
			hops = append(hops, strings.TrimSpace(hop))
		}
	}
	for i := len(hops) - 1; i > 0; i-- {
		if hops[i] != peer {
			continue
		}
		if ip := net.ParseIP(hops[i-1]); ip != nil {
			return ip.String()
		}
		break
	}
	return peer
}

// ipRateLimiterIdleTTL is how long an IP's bucket survives with no requests
// before ipRateLimiter.evictStale reclaims it. Sized well past every window
// this file configures (the widest is one hour) so a legitimate caller never
// gets a fresh, unpenalized bucket mid-window just because they paused.
const ipRateLimiterIdleTTL = 2 * time.Hour
const maxRateLimitBuckets = 10_000

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
		if len(rl.limiters) >= maxRateLimitBuckets {
			// Refusing here would let anyone spraying new keys lock every
			// new caller out until the next sweep. Dropping an arbitrary
			// bucket only ever makes one caller's limit more lenient.
			for key := range rl.limiters {
				delete(rl.limiters, key)
				break
			}
		}
		bucket = &ipBucket{limiter: rate.NewLimiter(rl.r, rl.burst)}
		rl.limiters[ip] = bucket
	}
	bucket.lastSeen = time.Now()
	return bucket.limiter.Allow()
}

// hasTokens reports whether key's bucket could take a request now, without
// taking one: for a limit that is only charged after the fact (a failed
// login), not on every request.
func (rl *ipRateLimiter) hasTokens(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	bucket, ok := rl.limiters[key]
	return !ok || bucket.limiter.Tokens() >= 1
}

func LimitRequestBody(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBytes {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, messageBody("request body is too large"))
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
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
