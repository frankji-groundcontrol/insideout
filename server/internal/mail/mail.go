// Package mail sends transactional email through the Resend HTTP API
// (https://resend.com/docs/api-reference). With no API key configured the
// sender runs in dev mode: it logs the message instead of delivering it,
// the same graceful-degradation posture as the offline PRD Coach. Auth mail
// goes through Enqueue, a small bounded background queue, so a request never
// waits on the provider.
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
	// qmu guards the Enqueue state: jobs waiting for a worker, how many workers
	// run, and a WaitGroup over everything accepted (see Enqueue, Wait).
	qmu     sync.Mutex
	waiting []queuedJob
	workers int
	pending sync.WaitGroup
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

// The queue bounds (rv4-REL-07). Starting values, not measured: 4 concurrent
// jobs keep a slow provider from holding more than 4 connections, 64 waiting
// jobs outlast the auth throttle's ceiling for several minutes, and a job's 30 s
// covers a token write plus the client's own 10 s provider timeout.
const (
	queueWorkers = 4
	queueDepth   = 64
	jobTimeout   = 30 * time.Second
)

type queuedJob struct {
	name string
	run  func(ctx context.Context) error
}

// Enqueue runs job on a background worker and returns at once, so the caller's
// response time cannot depend on what the job does (the provider's latency, a
// failure, whether there was anything to send): the emailed auth requests use
// it to answer a known and an unknown address in the same time (rv4-SEC-03).
// The job gets a fresh context, not the request's, which ends when the
// response does. A failed or panicking job is logged as "<name>: send", never
// surfaced. With queueDepth jobs already waiting the new one is dropped and
// logged, and Enqueue reports false: a flooded or stuck provider costs bounded
// memory, not an unbounded pile of goroutines.
//
// ponytail: in-memory only. Jobs still waiting when the process exits are lost
// (a redeploy can eat a just-requested link; the user asks again), and Wait is
// not wired into shutdown. Persist to a table or drain on SIGTERM if that bites.
func (s *Sender) Enqueue(name string, job func(ctx context.Context) error) bool {
	s.qmu.Lock()
	defer s.qmu.Unlock()
	if len(s.waiting) >= queueDepth {
		s.log.Warn(name+": mail queue full, send dropped", "depth", queueDepth)
		return false
	}
	s.waiting = append(s.waiting, queuedJob{name: name, run: job})
	s.pending.Add(1)
	if s.workers < queueWorkers {
		s.workers++
		go s.work()
	}
	return true
}

// Wait blocks until every job accepted so far has finished. Tests use it to
// read a request's outcome deterministically; the server never calls it.
func (s *Sender) Wait() { s.pending.Wait() }

// work drains the queue and exits when it is empty, so an idle sender holds no
// goroutines. Taking a job and deciding to exit share one lock with Enqueue's
// decision to start a worker, so a job is never left with nobody to run it.
func (s *Sender) work() {
	for {
		s.qmu.Lock()
		if len(s.waiting) == 0 {
			s.workers--
			s.qmu.Unlock()
			return
		}
		job := s.waiting[0]
		s.waiting = s.waiting[1:]
		s.qmu.Unlock()
		s.runJob(job)
		s.pending.Done()
	}
}

func (s *Sender) runJob(job queuedJob) {
	ctx, cancel := context.WithTimeout(context.Background(), jobTimeout)
	defer cancel()
	// A panic here is outside every request's recover middleware and would end
	// the process; a mail job must not be able to do that.
	defer func() {
		if p := recover(); p != nil {
			s.log.Error(job.name+": send panicked", "panic", p)
		}
	}()
	if err := job.run(ctx); err != nil {
		s.log.Error(job.name+": send", "error", err)
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
