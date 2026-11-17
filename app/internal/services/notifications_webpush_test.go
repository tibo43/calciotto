package services

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"testing"

	"app/internal/models"

	webpush "github.com/SherClockHolmes/webpush-go"
)

type pushTestTransport func(*http.Request) (*http.Response, error)

func (f pushTestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestWebPushSender_AgainstAFakePushService(t *testing.T) {
	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		t.Fatalf("GenerateVAPIDKeys: %v", err)
	}
	// The browser side of the subscription: its own P-256 key and auth secret.
	browserKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("browser key: %v", err)
	}
	authSecret := make([]byte, 16)
	rand.Read(authSecret)

	status := http.StatusCreated
	var lastRequest *http.Request
	sender := &WebPushSender{
		PublicKey: publicKey, PrivateKey: privateKey, Subject: "mailto:test@example.com",
		client: &http.Client{Transport: pushTestTransport(func(req *http.Request) (*http.Response, error) {
			lastRequest = req
			return &http.Response{StatusCode: status}, nil
		})},
	}
	sub := models.PushSubscription{
		Endpoint: "https://fcm.googleapis.com/push/test",
		P256dh:   base64.RawURLEncoding.EncodeToString(browserKey.PublicKey().Bytes()),
		Auth:     base64.RawURLEncoding.EncodeToString(authSecret),
	}

	if err := sender.Send(sub, []byte(`{"type":"registration_open"}`)); err != nil {
		t.Fatalf("Send to a 201 = %v, want nil", err)
	}
	if got := lastRequest.Header.Get("Content-Encoding"); got != "aes128gcm" {
		t.Errorf("Content-Encoding = %q, want aes128gcm (an encrypted payload)", got)
	}
	if auth := lastRequest.Header.Get("Authorization"); !strings.HasPrefix(auth, "vapid t=") || !strings.Contains(auth, publicKey) {
		t.Errorf("Authorization = %q, want a VAPID signature carrying the public key", auth)
	}

	status = http.StatusGone
	if err := sender.Send(sub, []byte(`{}`)); !errors.Is(err, ErrPushSubscriptionGone) {
		t.Errorf("Send to a 410 = %v, want ErrPushSubscriptionGone", err)
	}
	status = http.StatusInternalServerError
	if err := sender.Send(sub, []byte(`{}`)); err == nil || errors.Is(err, ErrPushSubscriptionGone) {
		t.Errorf("Send to a 500 = %v, want a plain error", err)
	}

	lastRequest = nil
	sub.Endpoint = "https://example.invalid/push/test"
	if err := sender.Send(sub, []byte(`{}`)); !errors.Is(err, ErrInvalidPushSubscription) || lastRequest != nil {
		t.Fatal("a stored subscription outside the allowlist must be rejected before transport")
	}
}

func TestAllowedPushEndpoints(t *testing.T) {
	for _, endpoint := range []string{
		"https://fcm.googleapis.com/push/test",
		"https://updates.push.services.mozilla.com/push/test",
		"https://web.push.apple.com/test",
		"https://web.push.windows.com/test",
	} {
		if !isAllowedPushEndpoint(endpoint) {
			t.Errorf("supported push endpoint %q was rejected", endpoint)
		}
	}
	for _, endpoint := range []string{
		"https://example.invalid/test",
		"http://fcm.googleapis.com/push/test",
		"https://fcm.googleapis.com:444/push/test",
		"https://fcm.googleapis.com.example.invalid/test",
	} {
		if isAllowedPushEndpoint(endpoint) {
			t.Errorf("unsupported push endpoint %q was accepted", endpoint)
		}
	}
}

func TestPushTransportRequiresPublicHTTPS(t *testing.T) {
	for _, address := range []string{"127.0.0.1:443", "[::1]:443", "100.64.0.1:443", "192.0.2.1:443", "8.8.8.8:80"} {
		if err := checkPushDialAddress(context.Background(), "tcp", address, nil); err == nil {
			t.Errorf("push transport accepted %q", address)
		}
	}
	if err := checkPushDialAddress(context.Background(), "tcp", "8.8.8.8:443", nil); err != nil {
		t.Errorf("push transport rejected a public HTTPS address: %v", err)
	}
	if err := webPushHTTPClient.CheckRedirect(nil, nil); !errors.Is(err, http.ErrUseLastResponse) {
		t.Error("push transport must reject redirects")
	}
}
