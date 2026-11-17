package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"os"
	"strings"
	"time"
)

// brevoSendEmailURL is Brevo's (formerly Sendinblue) transactional email
// endpoint — https://developers.brevo.com/reference/sendtransacemail.
const brevoSendEmailURL = "https://api.brevo.com/v3/smtp/email"

// brevoHTTPTimeout bounds how long a password-reset request waits on Brevo.
const brevoHTTPTimeout = 10 * time.Second

type brevoContact struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email"`
}

type brevoSendEmailRequest struct {
	Sender      brevoContact   `json:"sender"`
	To          []brevoContact `json:"to"`
	Subject     string         `json:"subject"`
	HTMLContent string         `json:"htmlContent"`
}

func newPasswordResetMailer() func(email, link string) error {
	apiKey := strings.TrimSpace(os.Getenv("BREVO_API_KEY"))
	senderEmail := strings.TrimSpace(os.Getenv("BREVO_SENDER_EMAIL"))
	if apiKey == "" || senderEmail == "" {
		return nil
	}
	senderName := os.Getenv("BREVO_SENDER_NAME")
	if senderName == "" {
		senderName = "Calciotto"
	}

	client := &http.Client{
		Timeout: brevoHTTPTimeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return func(email, link string) error {
		escapedLink := html.EscapeString(link)
		return sendViaBrevo(client, apiKey, brevoSendEmailRequest{
			Sender:  brevoContact{Name: senderName, Email: senderEmail},
			To:      []brevoContact{{Email: email}},
			Subject: "Reset your Calciotto password",
			HTMLContent: fmt.Sprintf(
				`<p>Click the link below to reset your Calciotto password:</p><p><a href="%s">%s</a></p><p>This link expires in one hour. If you didn't request this, you can ignore this email.</p>`,
				escapedLink, escapedLink,
			),
		})
	}
}

func sendViaBrevo(client *http.Client, apiKey string, payload brevoSendEmailRequest) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode brevo request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, brevoSendEmailURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build brevo request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("api-key", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("brevo request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return fmt.Errorf("Brevo returned HTTP %d", resp.StatusCode)
}
