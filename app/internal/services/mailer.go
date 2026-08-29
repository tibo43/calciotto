package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// brevoSendEmailURL is Brevo's (formerly Sendinblue) transactional email
// endpoint — https://developers.brevo.com/reference/sendtransacemail.
const brevoSendEmailURL = "https://api.brevo.com/v3/smtp/email"

// brevoHTTPTimeout bounds how long a password-reset request waits on Brevo.
// sendPasswordResetLink's callers already treat the send as fire-and-forget
// (it runs after the DB write has committed), but an unbounded call could
// still hang a request goroutine indefinitely.
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

// brevoErrorResponse mirrors the {"code": "...", "message": "..."} shape
// Brevo's API returns on a non-2xx response, e.g. {"code":"not_enough_credits",
// "message":"..."} once the account's daily sending quota (300/day on Brevo's
// free plan) is exhausted.
type brevoErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// sendViaBrevo sends a single transactional email through Brevo's HTTP API
// using apiKey. It returns an error whenever Brevo did not accept the email —
// including a quota breach — so the caller can log it; it never sends a
// second time and never falls back to the dev log stub itself.
func sendViaBrevo(apiKey, toEmail, subject, htmlContent string) error {
	senderEmail := os.Getenv("BREVO_SENDER_EMAIL")
	if senderEmail == "" {
		return fmt.Errorf("BREVO_SENDER_EMAIL is not set (required alongside BREVO_API_KEY — must be a sender verified in the Brevo account)")
	}
	senderName := os.Getenv("BREVO_SENDER_NAME")
	if senderName == "" {
		senderName = "Calciotto"
	}

	payload := brevoSendEmailRequest{
		Sender:      brevoContact{Name: senderName, Email: senderEmail},
		To:          []brevoContact{{Email: toEmail}},
		Subject:     subject,
		HTMLContent: htmlContent,
	}
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

	client := &http.Client{Timeout: brevoHTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("brevo request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	respBody, _ := io.ReadAll(resp.Body)
	var brevoErr brevoErrorResponse
	if jsonErr := json.Unmarshal(respBody, &brevoErr); jsonErr == nil && brevoErr.Message != "" {
		return fmt.Errorf("brevo API error (status %d, code %q): %s", resp.StatusCode, brevoErr.Code, brevoErr.Message)
	}
	return fmt.Errorf("brevo API error (status %d): %s", resp.StatusCode, string(respBody))
}
