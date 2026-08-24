// Package mail sends transactional email through the Resend HTTP API
// (https://resend.com/docs/api-reference). With no API key configured the
// sender runs in dev mode: it logs the message instead of delivering it,
// the same graceful-degradation posture as the offline PRD Coach.
package mail

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

const resendEndpoint = "https://api.resend.com/emails"

type Sender struct {
	apiKey string
	from   string
	log    *slog.Logger
	client *http.Client
	// endpoint defaults to the Resend API; tests point it at httptest.
	endpoint string
}

func NewSender(apiKey, from string, log *slog.Logger) *Sender {
	return &Sender{
		apiKey:    apiKey,
		from:      from,
		log:       log,
		client:    &http.Client{Timeout: 10 * time.Second},
		endpoint:  resendEndpoint,
	}
}

// Enabled reports whether real delivery is configured. False means dev
// mode: Send logs the message (including any links) and returns nil.
func (s *Sender) Enabled() bool {
	return s.apiKey != ""
}

// Send delivers (or, in dev mode, logs) a message. The text body should
// carry the load-bearing content — links included — so it doubles as the
// readable dev-mode log line.
func (s *Sender) Send(ctx context.Context, to, subject, text, html string) error {
	if !s.Enabled() {
		s.log.Info("mail dev mode — not sent", "to", to, "subject", subject, "text", text)
		return nil
	}

	payload, err := json.Marshal(map[string]string{
		"from":    s.from,
		"to":      to,
		"subject": subject,
		"text":    text,
		"html":    html,
	})
	if err != nil {
		return fmt.Errorf("mail: encode payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("mail: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	res, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("mail: send: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("mail: resend returned %s", res.Status)
	}
	return nil
}
