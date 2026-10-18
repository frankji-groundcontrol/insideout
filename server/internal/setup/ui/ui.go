// Package ui renders the setup flow's remote branch with huh: mode
// select, API base with the hosted default prefilled, sign-in method,
// email, a staged masked secret prompt, and the MCP-registration confirm. All
// behavior (probe, sign-in, persistence) stays in the pure flow
// package; rendering is thin by design so flow logic tests headless.
package ui

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/huh"

	"github.com/frankji-groundcontrol/insideout/server/internal/setup/flow"
)

// Result carries the flow outcome plus the answers the join phase
// consumes beyond flow.Run's contract.
type Result struct {
	flow.Result
	// RegisterMCP is the user's answer to the MCP-registration confirm.
	RegisterMCP bool
}

// ErrNoTTY is returned when setup cannot render an interactive form.
type ErrNoTTY struct{}

func (ErrNoTTY) Error() string {
	return "insideout setup: interactive setup needs a terminal; " +
		"run instead: insideout setup --hosted | --local --postgres bundled|own"
}

// Run shows the remote-branch forms and drives flow.Run with the
// collected answers. It refuses to start without a terminal.
func Run(ctx context.Context, deps flow.Deps) (Result, error) {
	// Checked on the real os.Stdin / os.Stdout: under `go test` stdout
	// is a pipe, so the refusal branch stays reachable in tests.
	if !isCharDevice(os.Stdin) || !isCharDevice(os.Stdout) {
		return Result{}, ErrNoTTY{}
	}

	mode, err := askMode(ctx)
	if err != nil {
		return Result{}, err
	}
	if mode == flow.ModeLocal {
		// The local branch is the join hook: ask bundled/own here,
		// then hand off to the assembly (env write, Postgres, migrate,
		// boot offer). The DSN paste for "own" happens inside the
		// assembly's hidden read — one hidden-input path, not two.
		if deps.LocalSetup == nil {
			res, err := flow.Run(ctx, flow.Inputs{Mode: mode}, deps)
			return Result{Result: res}, err
		}
		postgres, err := askPostgres(ctx)
		if err != nil {
			return Result{}, err
		}
		if err := deps.LocalSetup(ctx, postgres); err != nil {
			return Result{}, fmt.Errorf("insideout setup: %w", err)
		}
		return Result{}, nil
	}

	inputs, registerMCP, err := askRemote(ctx, deps.DefaultBaseURL)
	if err != nil {
		return Result{}, err
	}
	deps.PromptSecret = func(method string) (string, error) {
		return askSecret(ctx, method)
	}
	res, err := flow.Run(ctx, inputs, deps)
	return Result{Result: res, RegisterMCP: registerMCP}, err
}

func askMode(ctx context.Context) (string, error) {
	var mode string
	if err := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title("Where should this machine connect?").
			Options(
				huh.NewOption("Remote service (hosted, or self-hosted)", flow.ModeRemote),
				huh.NewOption("Server on this machine (PostgreSQL)", flow.ModeLocal),
			).
			Value(&mode),
	)).RunWithContext(ctx); err != nil {
		return "", fmt.Errorf("insideout setup: %w", err)
	}
	return mode, nil
}

// askPostgres is the local branch's one question; the own-URL DSN is
// collected inside the local assembly (hidden read), not here.
func askPostgres(ctx context.Context) (string, error) {
	var postgres string
	if err := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title("PostgreSQL for this server").
			Options(
				huh.NewOption("Bundled Postgres in Docker (localhost port 5442)", "bundled"),
				huh.NewOption("A Postgres URL I already have", "own"),
			).
			Value(&postgres),
	)).RunWithContext(ctx); err != nil {
		return "", fmt.Errorf("insideout setup: %w", err)
	}
	return postgres, nil
}

func askRemote(ctx context.Context, defaultBase string) (flow.Inputs, bool, error) {
	var (
		baseURL = defaultBase // hosted default prefilled; enter accepts it
		method  = flow.AuthPassword
	)
	if err := huh.NewForm(huh.NewGroup(
		huh.NewInput().
			Title("API base URL").
			Description("Enter accepts the default. Self-hosted servers use their own /api/v1 URL.").
			Value(&baseURL),
		huh.NewSelect[string]().
			Title("Sign in with").
			Options(
				huh.NewOption("Password", flow.AuthPassword),
				huh.NewOption("Emailed login code", flow.AuthCode),
				huh.NewOption("Magic link", flow.AuthMagic),
			).
			Value(&method),
	)).RunWithContext(ctx); err != nil {
		return flow.Inputs{}, false, fmt.Errorf("insideout setup: %w", err)
	}

	var (
		email       string
		registerMCP bool
	)
	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Email").Value(&email),
		),
		huh.NewGroup(
			huh.NewConfirm().
				Title("Merge the MCP registration into your MCP client config?").
				Description("Adds the InsideOut MCP server entry the CLI can print later.").
				Value(&registerMCP),
		),
	).RunWithContext(ctx); err != nil {
		return flow.Inputs{}, false, fmt.Errorf("insideout setup: %w", err)
	}

	return flow.Inputs{
		Mode:       flow.ModeRemote,
		BaseURL:    baseURL,
		AuthMethod: method,
		Email:      email,
	}, registerMCP, nil
}

func askSecret(ctx context.Context, method string) (string, error) {
	var secret string
	if err := huh.NewForm(huh.NewGroup(
		huh.NewInput().
			Title(secretTitle(method)).
			Description("Check your email if requested; hidden as you type.").
			EchoMode(huh.EchoModePassword).
			Value(&secret),
	)).RunWithContext(ctx); err != nil {
		return "", fmt.Errorf("insideout setup: %w", err)
	}
	return secret, nil
}

func secretTitle(method string) string {
	switch method {
	case flow.AuthCode:
		return "Login code (check your email)"
	case flow.AuthMagic:
		return "Magic link (paste the link from your email)"
	default:
		return "Password"
	}
}

// isCharDevice reports whether f is attached to a terminal-like
// character device. /dev/null also counts, which is why both ends are
// checked and stdout is the decisive leg under `go test`.
func isCharDevice(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
