package services

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"syscall"
	"time"

	"app/internal/models"

	webpush "github.com/SherClockHolmes/webpush-go"
)

// WebPushSender sends through the browsers' push services with VAPID.
type WebPushSender struct {
	PublicKey  string
	PrivateKey string
	Subject    string
	client     *http.Client
}

func (w *WebPushSender) Send(sub models.PushSubscription, payload []byte) (int, error) {
	// Subscriptions saved before endpoint validation are checked here too.
	if !isAllowedPushEndpoint(sub.Endpoint) {
		return 0, ErrInvalidPushSubscription
	}
	client := w.client
	if client == nil {
		client = webPushHTTPClient
	}
	resp, err := webpush.SendNotification(payload, &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys:     webpush.Keys{P256dh: sub.P256dh, Auth: sub.Auth},
	}, &webpush.Options{
		HTTPClient:      client,
		Subscriber:      w.Subject,
		VAPIDPublicKey:  w.PublicKey,
		VAPIDPrivateKey: w.PrivateKey,
		TTL:             24 * 60 * 60,
		Urgency:         webpush.UrgencyHigh,
	})
	if err != nil {
		// URL errors from net/http can contain the subscription's credentials.
		return 0, errors.New("web push delivery failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		return resp.StatusCode, ErrPushSubscriptionGone
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, fmt.Errorf("push service returned HTTP %d", resp.StatusCode)
	}
	return resp.StatusCode, nil
}

func isAllowedPushEndpoint(raw string) bool {
	endpoint, err := url.Parse(raw)
	if err != nil || endpoint.Scheme != "https" || endpoint.User != nil || endpoint.Fragment != "" {
		return false
	}
	if port := endpoint.Port(); port != "" && port != "443" {
		return false
	}
	host := strings.ToLower(endpoint.Hostname())
	switch host {
	case "fcm.googleapis.com", "updates.push.services.mozilla.com", "web.push.apple.com", "web.push.windows.com":
		return true
	}
	// Edge hands out Windows Notification Service endpoints on per-region
	// hosts such as wns2-par02p.notify.windows.com.
	return strings.HasSuffix(host, ".notify.windows.com")
}

var webPushHTTPClient = &http.Client{
	Timeout: 10 * time.Second,
	CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	},
	Transport: &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
			// Runs with the numeric destination after DNS resolution and
			// before connecting. The transport connects directly.
			ControlContext: checkPushDialAddress,
		}).DialContext,
		ForceAttemptHTTP2:   true,
		TLSHandshakeTimeout: 5 * time.Second,
		IdleConnTimeout:     90 * time.Second,
	},
}

// IsGlobalUnicast includes several special-purpose ranges. Reject those
// alongside the private ranges, and exclude IPv6 translation and tunnel ranges.
var nonPublicPushNetworks = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("2001::/23"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("2002::/16"),
	netip.MustParsePrefix("3fff::/20"),
}

var globalPushIPv6 = netip.MustParsePrefix("2000::/3")

func checkPushDialAddress(_ context.Context, _, address string, _ syscall.RawConn) error {
	destination, err := netip.ParseAddrPort(address)
	if err != nil || destination.Port() != 443 || !isPublicPushIP(destination.Addr().Unmap()) {
		return errors.New("push service must use a public HTTPS address")
	}
	return nil
}

func isPublicPushIP(ip netip.Addr) bool {
	if !ip.IsGlobalUnicast() || ip.IsPrivate() || (ip.Is6() && !globalPushIPv6.Contains(ip)) {
		return false
	}
	for _, network := range nonPublicPushNetworks {
		if network.Contains(ip) {
			return false
		}
	}
	return true
}
