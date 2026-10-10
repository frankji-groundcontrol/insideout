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
	"sync"
	"time"
)

const resendEndpoint = "https://api.resend.com/emails"

type Sender struct {
	apiKey string
	from   string
	log    *slog.Logger
	// devBody opts in to logging the message body in dev mode; off by
	// default so magic-link tokens never land in logs.
	devBody bool
	client  *http.Client
	// endpoint defaults to the Resend API; tests point it at httptest.
	endpoint string
	// lastMu guards lastErr, the error of the most recent Send (nil after
	// a success). The admin mail probe surfaces it: the auth flows swallow
	// send failures into a uniform 200, so without it a dead mail funnel
	// leaves nothing but a server log line (review-3 ARCH-03).
	lastMu  sync.Mutex
	lastErr error
}

func NewSender(apiKey, from string, log *slog.Logger, devBody bool) *Sender {
	return &Sender{
		apiKey:   apiKey,
		from:     from,
		log:      log,
		devBody:  devBody,
		client:   &http.Client{Timeout: 10 * time.Second},
		endpoint: resendEndpoint,
	}
}

// Enabled reports whether real delivery is configured. False means dev
// mode: Send logs the message — recipient and subject always, the body
// only when NewSender got devBody — and returns nil.
func (s *Sender) Enabled() bool {
	return s.apiKey != ""
}

// Send delivers (or, in dev mode, logs) a message and records the outcome
// for LastSendError. The text body should carry the load-bearing content —
// links included; in dev mode it reaches the log line only when the sender
// was built with the devBody opt-in.
func (s *Sender) Send(ctx context.Context, to, subject, text, html string) error {
	err := s.send(ctx, to, subject, text, html)
	s.lastMu.Lock()
	s.lastErr = err
	s.lastMu.Unlock()
	return err
}

// LastSendError returns the error of the most recent Send, or "" when the
// last send succeeded (or nothing has been sent yet). The admin mail probe
// reports it so an operator can see the funnel's last real failure.
func (s *Sender) LastSendError() string {
	s.lastMu.Lock()
	defer s.lastMu.Unlock()
	if s.lastErr == nil {
		return ""
	}
	return s.lastErr.Error()
}

func (s *Sender) send(ctx context.Context, to, subject, text, html string) error {
	if !s.Enabled() {
		if s.devBody {
			s.log.Info("mail dev mode — not sent", "to", to, "subject", subject, "text", text)
		} else {
			s.log.Info("mail dev mode — not sent", "to", to, "subject", subject)
		}
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
