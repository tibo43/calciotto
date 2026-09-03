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
