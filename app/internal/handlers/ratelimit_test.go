package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func newRateLimitedRouter(r rate.Limit, burst int) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(RateLimit(r, burst))
	engine.POST("/thing", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return engine
}

func doPost(engine *gin.Engine, remoteAddr string) int {
	req := httptest.NewRequest(http.MethodPost, "/thing", nil)
	req.RemoteAddr = remoteAddr
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec.Code
}

func TestRateLimitAllowsUpToBurst(t *testing.T) {
	engine := newRateLimitedRouter(rate.Every(time.Hour), 3)

	for i := 0; i < 3; i++ {
		if code := doPost(engine, "1.2.3.4:5555"); code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i, code)
		}
	}
}

func TestRateLimitRejectsPastBurst(t *testing.T) {
	engine := newRateLimitedRouter(rate.Every(time.Hour), 3)

	for i := 0; i < 3; i++ {
		doPost(engine, "1.2.3.4:5555")
	}

	if code := doPost(engine, "1.2.3.4:5555"); code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 once the burst is exhausted, got %d", code)
	}
}

func TestRateLimitIsPerIP(t *testing.T) {
	engine := newRateLimitedRouter(rate.Every(time.Hour), 1)

	if code := doPost(engine, "1.2.3.4:5555"); code != http.StatusOK {
		t.Fatalf("first caller: expected 200, got %d", code)
	}
	if code := doPost(engine, "1.2.3.4:5555"); code != http.StatusTooManyRequests {
		t.Fatalf("first caller's second request: expected 429, got %d", code)
	}
	// A different IP has its own bucket and is unaffected by the first one's.
	if code := doPost(engine, "9.9.9.9:5555"); code != http.StatusOK {
		t.Fatalf("second caller: expected 200, got %d", code)
	}
}

// newTrustedProxyRouter is newRateLimitedRouter plus the production trusted-proxy
// policy (ConfigureTrustedProxies, exactly as main.go applies it). The three
// tests below are about that policy, so wiring it here rather than relying on
// gin's default is the whole point: gin's default trusts every caller.
func newTrustedProxyRouter(t *testing.T, r rate.Limit, burst int) *gin.Engine {
	t.Helper()
	engine := newRateLimitedRouter(r, burst)
	if err := ConfigureTrustedProxies(engine); err != nil {
		t.Fatalf("ConfigureTrustedProxies returned error: %v", err)
	}
	return engine
}

func doPostForwardedFor(engine *gin.Engine, remoteAddr, forwardedFor string) int {
	req := httptest.NewRequest(http.MethodPost, "/thing", nil)
	req.RemoteAddr = remoteAddr
	req.Header.Set("X-Forwarded-For", forwardedFor)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec.Code
}

// TestRateLimitIgnoresForwardedForFromUntrustedPeer is the regression test for
// the bypass: gin's default trustedProxies is 0.0.0.0/0 + ::/0, so
// c.ClientIP() used to return whatever the caller put in X-Forwarded-For and a
// script rotating that header got a fresh, unpenalized bucket on every single
// request to /auth/login, /auth/signup, /auth/forgot-password and
// /auth/reset-password.
//
// Same TCP peer, a different forged header each time: all of them must share
// one bucket.
func TestRateLimitIgnoresForwardedForFromUntrustedPeer(t *testing.T) {
	engine := newTrustedProxyRouter(t, rate.Every(time.Hour), 1)

	if code := doPostForwardedFor(engine, "203.0.113.7:5555", "1.1.1.1"); code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", code)
	}
	if code := doPostForwardedFor(engine, "203.0.113.7:5555", "2.2.2.2"); code != http.StatusTooManyRequests {
		t.Fatalf("a second request from the same peer with a different X-Forwarded-For: expected 429, got %d", code)
	}
	// X-Real-IP is dropped from RemoteIPHeaders for the same reason — it
	// carries a single address with no chain to validate, so believing it
	// would reopen the bypass under another header name.
	req := httptest.NewRequest(http.MethodPost, "/thing", nil)
	req.RemoteAddr = "203.0.113.7:5555"
	req.Header.Set("X-Real-IP", "3.3.3.3")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("a third request from the same peer with a forged X-Real-IP: expected 429, got %d", rec.Code)
	}
}

// TestRateLimitUsesRealClientBehindTrustedProxy is the other half of the
// policy: TRUSTED_PROXIES=none would be spoof-proof but would put every
// visitor behind a load balancer in one shared bucket, so the default trusts
// the immediate (private-range) peer. Trusting the proxy must not mean
// trusting the header, though: gin walks X-Forwarded-For right-to-left and
// takes the right-most non-proxy address — the one the load balancer appended
// — so entries the client prepended are ignored.
func TestRateLimitUsesRealClientBehindTrustedProxy(t *testing.T) {
	engine := newTrustedProxyRouter(t, rate.Every(time.Hour), 1)

	// Two requests from the same real client, each prepending a different
	// forged hop: one bucket, so the second is rejected.
	if code := doPostForwardedFor(engine, "10.1.2.3:5555", "1.1.1.1, 203.0.113.7"); code != http.StatusOK {
		t.Fatalf("first request through the proxy: expected 200, got %d", code)
	}
	if code := doPostForwardedFor(engine, "10.1.2.3:5555", "2.2.2.2, 203.0.113.7"); code != http.StatusTooManyRequests {
		t.Fatalf("same real client, different forged prefix: expected 429, got %d", code)
	}
	// A genuinely different client behind the same proxy is still its own
	// bucket — otherwise the fix would just be throttling everyone together.
	if code := doPostForwardedFor(engine, "10.1.2.3:5555", "198.51.100.4"); code != http.StatusOK {
		t.Fatalf("a different real client through the same proxy: expected 200, got %d", code)
	}
}

// TestRateLimitTrustedProxiesNone pins the explicit opt-out: with
// TRUSTED_PROXIES=none the header is ignored even when the peer is on a
// private range, and the bucket is keyed on the TCP peer alone.
func TestRateLimitTrustedProxiesNone(t *testing.T) {
	t.Setenv(trustedProxiesEnv, trustedProxiesNone)
	engine := newTrustedProxyRouter(t, rate.Every(time.Hour), 1)

	if code := doPostForwardedFor(engine, "10.1.2.3:5555", "203.0.113.7"); code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", code)
	}
	if code := doPostForwardedFor(engine, "10.1.2.3:5555", "198.51.100.4"); code != http.StatusTooManyRequests {
		t.Fatalf("same peer, different X-Forwarded-For: expected 429, got %d", code)
	}
}
