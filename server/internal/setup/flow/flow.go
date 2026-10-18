// Package flow is the pure setup state machine: given collected
// answers and injected capabilities it resolves the API base URL,
// signs in, and persists the result. It performs no I/O of its own —
// probes, HTTP and config writing arrive as interfaces, so the whole
// flow tests headless (docs/plans/2026-10-07-setup-tui.md, slice 1).
package flow

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/frankji-groundcontrol/insideout/server/internal/clientconfig"
)

const (
	ModeRemote = "remote"
	ModeLocal  = "local"

	AuthPassword = "password"
	AuthCode     = "code"
	AuthMagic    = "magic"
)

// Inputs are the answers the setup flow runs on (collected by the UI
// or supplied via flags).
type Inputs struct {
	Mode       string // ModeRemote or ModeLocal
	BaseURL    string // empty means the hosted default
	AuthMethod string // AuthPassword, AuthCode or AuthMagic
	Email      string
	Secret     string // password, emailed code, or pasted magic link
}

// Result reports what one Run achieved.
type Result struct {
	APIBase  string
	SignedIn bool
}

// Prober checks API reachability. Consumer-defined so callers can
// bring their own probe (slot E's probe package at the join).
type Prober interface {
	ProbeAPI(ctx context.Context, baseURL string) error
}

// LoginClient is the slice of apiclient the sign-in methods need.
// *apiclient.Client satisfies it (asserted in flow_test.go).
type LoginClient interface {
	Login(email, password string) error
	RequestLoginCode(email string) error
	VerifyLoginCode(email, code string) error
	RequestMagicLink(email string) error
	ConsumeMagicLink(token string) error
	AuthConfig() (string, error)
	Token() string
	RefreshToken() string
}

// Persister stores the finished config (internal/clientconfig's 0600
// file at the join).
type Persister interface {
	Save(clientconfig.Config) error
}

// SecretPrompt collects the method-specific secret after the API has been
// probed and, for email methods, after the request has been sent.
type SecretPrompt func(method string) (string, error)

// Deps carries the capabilities Run needs.
type Deps struct {
	Prober         Prober
	Client         LoginClient
	Persister      Persister
	PromptSecret   SecretPrompt
	DefaultBaseURL string // used when Inputs.BaseURL is empty
	// LocalSetup is the join hook for the interactive UI's local
	// branch: ui.Run calls it with "bundled"|"own" instead of Run
	// (Run stays pure and refuses local mode). Nil means the caller
	// has no local assembly wired.
	LocalSetup func(ctx context.Context, postgres string) error
}

// ErrLocalNotImplemented: local mode is slice 2's work; the remote
// branch returns it untouched so the caller can branch on it.
type ErrLocalNotImplemented struct{}

func (ErrLocalNotImplemented) Error() string {
	return "insideout setup: local mode is not implemented yet (slice 2); use remote mode for now"
}

// ErrBadURL rejects base URLs that are not http(s) or lack a host.
type ErrBadURL struct{ URL string }

func (e ErrBadURL) Error() string {
	return fmt.Sprintf("insideout setup: %q is not a valid API base URL (want http/https with a host)", e.URL)
}

// ErrEmailRequired and ErrSecretRequired cover the missing-credential
// cases for every sign-in method.
type ErrEmailRequired struct{}

func (ErrEmailRequired) Error() string {
	return "insideout setup: email is required to sign in"
}

type ErrSecretRequired struct{}

func (ErrSecretRequired) Error() string {
	return "insideout setup: the password, code or magic link is required to sign in"
}

// ProbeError wraps a failed reachability probe, carrying the URL that
// was probed so the UI can point at the right field.
type ProbeError struct {
	BaseURL string
	Err     error
}

func (e *ProbeError) Error() string {
	return fmt.Sprintf("insideout setup: %s is not reachable: %v", e.BaseURL, e.Err)
}

func (e *ProbeError) Unwrap() error { return e.Err }

// Run executes the remote branch (local mode is not implemented yet):
// resolve and validate the base URL, probe it, run the chosen sign-in
// method, and persist the resulting access and refresh tokens.
func Run(ctx context.Context, inputs Inputs, deps Deps) (Result, error) {
	if inputs.Mode == ModeLocal {
		return Result{}, ErrLocalNotImplemented{}
	}
	if deps.Prober == nil || deps.Client == nil || deps.Persister == nil {
		return Result{}, fmt.Errorf("insideout setup: flow needs a prober, login client and persister")
	}

	base := inputs.BaseURL
	if base == "" {
		base = deps.DefaultBaseURL
	}
	if err := validateBaseURL(base); err != nil {
		return Result{}, err
	}
	if inputs.Email == "" {
		return Result{}, ErrEmailRequired{}
	}
	if err := deps.Prober.ProbeAPI(ctx, base); err != nil {
		return Result{}, &ProbeError{BaseURL: base, Err: err}
	}
	if err := signIn(inputs, deps.Client, deps.PromptSecret); err != nil {
		return Result{}, err
	}
	if err := deps.Persister.Save(clientconfig.Config{APIBase: base, Token: deps.Client.Token(), RefreshToken: deps.Client.RefreshToken()}); err != nil {
		return Result{}, err
	}
	return Result{APIBase: base, SignedIn: true}, nil
}

func validateBaseURL(base string) error {
	u, err := url.Parse(base)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return ErrBadURL{URL: base}
	}
	return nil
}

func signIn(inputs Inputs, client LoginClient, prompt SecretPrompt) error {
	switch inputs.AuthMethod {
	case AuthPassword:
		secret, err := secretFor(inputs, prompt)
		if err != nil {
			return err
		}
		return client.Login(inputs.Email, secret)
	case AuthCode:
		if err := client.RequestLoginCode(inputs.Email); err != nil {
			return err
		}
		secret, err := secretFor(inputs, prompt)
		if err != nil {
			return err
		}
		return client.VerifyLoginCode(inputs.Email, secret)
	case AuthMagic:
		if err := client.RequestMagicLink(inputs.Email); err != nil {
			return err
		}
		secret, err := secretFor(inputs, prompt)
		if err != nil {
			return err
		}
		return client.ConsumeMagicLink(tokenFromLink(secret))
	default:
		return fmt.Errorf("insideout setup: unknown sign-in method %q", inputs.AuthMethod)
	}
}

func secretFor(inputs Inputs, prompt SecretPrompt) (string, error) {
	if inputs.Secret != "" {
		return inputs.Secret, nil
	}
	if prompt == nil {
		return "", ErrSecretRequired{}
	}
	secret, err := prompt(inputs.AuthMethod)
	if err != nil {
		return "", err
	}
	if secret == "" {
		return "", ErrSecretRequired{}
	}
	return secret, nil
}

// tokenFromLink accepts either a pasted URL (…?token=abc…) or a raw
// token; both emailed flows land in the user's clipboard as the URL.
// Mirrors cmd/insideout/cli_auth.go so setup and the plain CLI parse
// the same paste the same way.
func tokenFromLink(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.Index(s, "token="); idx >= 0 {
		v := s[idx+len("token="):]
		if amp := strings.IndexAny(v, "& \t"); amp >= 0 {
			v = v[:amp]
		}
		return v
	}
	return s
}
