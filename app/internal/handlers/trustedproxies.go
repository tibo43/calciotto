package handlers

import (
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// trustedProxiesEnv holds a comma-separated list of IPs/CIDRs whose
// X-Forwarded-For header this app believes, read the same way
// CORS_ALLOWED_ORIGINS is in main.go. The literal value "none" disables
// X-Forwarded-For entirely (see ConfigureTrustedProxies).
const trustedProxiesEnv = "TRUSTED_PROXIES"

// trustedProxiesNone is the explicit "believe nobody" value for
// TRUSTED_PROXIES — spelled out rather than expressed as an empty string,
// which is indistinguishable from "the variable isn't set" and would make an
// accidental blank env var silently change the policy.
const trustedProxiesNone = "none"

// defaultTrustedProxies is the fallback when TRUSTED_PROXIES is unset: the
// loopback and RFC1918/RFC4193 private ranges, and nothing else. Both
// environments this app runs in reach it over one of those — docker-compose
// puts the frontend and the backend on a bridge network (172.16.0.0/12), and a
// managed platform's load balancer reaches the container over its own internal
// network — while a request arriving straight off the public internet has a
// public peer address and is therefore never believed.
var defaultTrustedProxies = []string{
	"127.0.0.0/8",
	"::1/128",
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"fc00::/7",
}

// ConfigureTrustedProxies tells gin whose X-Forwarded-For to believe, and is
// the whole reason the per-IP rate limiter in ratelimit.go actually limits
// anything.
//
// gin's default is trustedProxies = ["0.0.0.0/0", "::/0"] — it trusts *every*
// caller, including the caller itself — so c.ClientIP() returned an IP taken
// straight from a client-supplied X-Forwarded-For. A script hitting
// /auth/login with a different X-Forwarded-For on every request therefore got
// a brand-new rate-limit bucket every time, which made all four /auth/*
// limiters decorative.
//
// The policy this settles on is "trust the immediate proxy, and only it":
//
//   - TRUSTED_PROXIES, when set, is the list of IPs/CIDRs to trust — set it to
//     the platform load balancer's own range once it's known.
//   - TRUSTED_PROXIES=none trusts nobody: c.ClientIP() then always reports the
//     real TCP peer (c.Request.RemoteAddr), spoof-proof but, behind a load
//     balancer, identical for every visitor — which would put the entire
//     internet in one shared rate-limit bucket. That is why it is not the
//     default despite being the strictest option: it trades a spoofable limit
//     for a self-inflicted outage.
//   - Unset falls back to defaultTrustedProxies (loopback + private ranges),
//     which is what both docker-compose and a platform LB reaching the
//     container over its internal network look like.
//
// Trusting the immediate proxy is not the same as trusting the header: gin
// walks X-Forwarded-For right-to-left and returns the right-most address that
// is *not* itself a trusted proxy (see Engine.validateHeader), so the entries
// a client prepends are skipped in favour of the one the proxy appended. If
// the peer turns out not to be in the trusted list at all — a platform whose
// LB has a public address, say — the effect is the conservative one: the
// header is ignored and every caller shares the LB's own bucket, which
// throttles too much rather than not at all. Set TRUSTED_PROXIES to fix that;
// it fails toward over-limiting, never toward spoofability.
//
// RemoteIPHeaders is narrowed to X-Forwarded-For alone, dropping gin's default
// X-Real-IP: X-Real-IP carries a single address with no chain, so there is
// nothing for the right-to-left walk to validate — from a trusted peer it is
// believed verbatim, which is exactly the property this function exists to
// remove.
func ConfigureTrustedProxies(engine *gin.Engine) error {
	engine.RemoteIPHeaders = []string{"X-Forwarded-For"}

	raw := strings.TrimSpace(os.Getenv(trustedProxiesEnv))
	if strings.EqualFold(raw, trustedProxiesNone) {
		log.Printf("%s=none: X-Forwarded-For is ignored, client IPs come from the TCP peer address", trustedProxiesEnv)
		return engine.SetTrustedProxies(nil)
	}

	proxies := parseTrustedProxies(raw)
	if len(proxies) == 0 {
		proxies = defaultTrustedProxies
		log.Printf("%s unset, trusting loopback and private ranges only: %s", trustedProxiesEnv, strings.Join(proxies, ", "))
	} else {
		log.Printf("%s=%s", trustedProxiesEnv, strings.Join(proxies, ", "))
	}
	return engine.SetTrustedProxies(proxies)
}

// parseTrustedProxies splits and trims the comma-separated env value, dropping
// empty entries — the same shape allowedOrigins() parses CORS_ALLOWED_ORIGINS
// with in main.go.
func parseTrustedProxies(raw string) []string {
	proxies := make([]string, 0, 4)
	for _, proxy := range strings.Split(raw, ",") {
		if proxy = strings.TrimSpace(proxy); proxy != "" {
			proxies = append(proxies, proxy)
		}
	}
	return proxies
}
