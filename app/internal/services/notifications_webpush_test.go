package services

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"app/internal/models"

	webpush "github.com/SherClockHolmes/webpush-go"
)

// TestWebPushSender_AgainstAFakePushService runs the real library end to end
// against a local "push service": the request must be encrypted and
// VAPID-signed, and the service's answers must map to the right outcome.
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastRequest = r
		w.WriteHeader(status)
	}))
	defer server.Close()

	sender := &WebPushSender{PublicKey: publicKey, PrivateKey: privateKey, Subject: "mailto:test@example.com"}
	sub := models.PushSubscription{
		Endpoint: server.URL + "/push/abc",
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
}
