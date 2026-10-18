package main

// The setup front door (docs/plans/2026-10-07-setup-tui.md, join
// phase): wires the slot packages together — clientconfig (A),
// flow+ui (B), the hidden-prompt CLI (C) — behind one subcommand.
// `insideout setup` with no flags is the interactive form; the flags
// mirror scripts/setup.sh's agent contract so an agent can drive the
// same door without a terminal.

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
	"github.com/frankji-groundcontrol/insideout/server/internal/clientconfig"
	"github.com/frankji-groundcontrol/insideout/server/internal/setup/flow"
	"github.com/frankji-groundcontrol/insideout/server/internal/setup/probe"
	"github.com/frankji-groundcontrol/insideout/server/internal/setup/ui"
	"os"
	"sync"
)

// clientAPIBase and clientToken resolve client credentials through
// internal/clientconfig once for the whole process: per field, the
// environment (INSIDEOUT_API / INSIDEOUT_TOKEN) overrides the config
// file `insideout setup` writes, and the hosted default applies last.
// Every CLI verb resolves through these helpers instead of reading
// os.Getenv directly (the plan's single-precedence rule).
func clientAPIBase() string {
	cfg, err := clientconfig.Load(mustConfigPath())
	if err != nil || cfg.APIBase == "" {
		return defaultAPIBase
	}
	return cfg.APIBase
}

func clientToken() string {
	cfg, err := clientconfig.Load(mustConfigPath())
	if err != nil {
		return ""
	}
	return cfg.Token
}

func mustConfigPath() string {
	path, err := clientconfig.DefaultPath()
	if err != nil {
		return "" // Load("") fails safe: zero config, defaults apply
	}
	return path
}

// setupClient adapts *apiclient.Client to flow.LoginClient. The base
// URL is collected by the form after this struct exists, and
// apiclient has no SetBase — so the client is rebuilt on rebind. The
// rebind happens in the prober adapter, which flow.Run hands the
// resolved URL before it signs in (flow's own contract: probe
// precedes signIn; asserted by flow's call-order tests).
type setupClient struct {
	mu sync.Mutex
	c  *apiclient.Client
}

func (s *setupClient) rebind(base string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.c = apiclient.New(base)
}

func (s *setupClient) client() *apiclient.Client {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.c == nil {
		s.c = apiclient.New(defaultAPIBase)
	}
	return s.c
}

func (s *setupClient) Login(email, password string) error {
	return s.client().Login(email, password)
}

func (s *setupClient) RequestLoginCode(email string) error {
	return s.client().RequestLoginCode(email)
}

func (s *setupClient) VerifyLoginCode(email, code string) error {
	return s.client().VerifyLoginCode(email, code)
}

func (s *setupClient) RequestMagicLink(email string) error {
	return s.client().RequestMagicLink(email)
}

func (s *setupClient) ConsumeMagicLink(token string) error {
	return s.client().ConsumeMagicLink(token)
}

func (s *setupClient) AuthConfig() (string, error) { return s.client().AuthConfig() }

func (s *setupClient) Token() string { return s.client().Token() }

// setupProber adapts probe.APIProbe to flow.Prober and rebinds the
// shared login client to the URL flow resolved (see setupClient).
type setupProber struct{ sc *setupClient }

func (p setupProber) ProbeAPI(ctx context.Context, baseURL string) error {
	_, err := probe.APIProbe(ctx, nil, baseURL)
	if err == nil {
		p.sc.rebind(baseURL)
	}
	return err
}

// setupPersister adapts clientconfig.Save to flow.Persister.
type setupPersister struct{ path string }

func (p setupPersister) Save(cfg clientconfig.Config) error {
	return clientconfig.Save(p.path, cfg)
}

// runSetupCommand is the setup subcommand. Flags mirror
// scripts/setup.sh's agent contract: --hosted confirms the hosted
// path; --local --postgres bundled|own is the local-server path
// (assembled in slice 2 — until then it points at the script); no
// flags runs the interactive form, which refuses without a terminal.
func runSetupCommand(rest []string) (bool, int) {
	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	hosted := fs.Bool("hosted", false, "confirm the hosted path (no environment variables)")
	local := fs.Bool("local", false, "run a server on this machine (PostgreSQL)")
	postgres := fs.String("postgres", "", "bundled|own — required with --local")
	noStart := fs.Bool("no-start", false, "with --local --postgres bundled: prepare only, do not start")
	createRoles := fs.Bool("create-roles", false, "with --local --postgres own: create the insideout_owner/insideout_app roles if missing, without asking")
	check := fs.Bool("check", false, "re-run the health probes against the current config")
	llm := fs.Bool("llm", false, "with --check: probe the legacy openai schema; current schemas need a coach turn")
	if err := fs.Parse(rest); err != nil {
		return true, 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "usage: insideout setup [--hosted | --local --postgres bundled|own [--no-start] [--create-roles] | --check [--llm]]\n")
		return true, 2
	}
	if *check {
		if *hosted || *local || *postgres != "" || *noStart || *createRoles {
			fmt.Fprintln(os.Stderr, "setup: --check cannot be combined with --hosted/--local/--postgres/--no-start/--create-roles")
			return true, 2
		}
		return runHealthCheck(*llm)
	}
	if *llm {
		fmt.Fprintln(os.Stderr, "setup: --llm belongs to --check")
		return true, 2
	}
	if *hosted && *local {
		fmt.Fprintln(os.Stderr, "setup: pick one of --hosted or --local")
		return true, 2
	}
	switch {
	case *hosted:
		if *postgres != "" || *noStart || *createRoles {
			fmt.Fprintln(os.Stderr, "setup: --postgres/--no-start/--create-roles belong to --local")
			return true, 2
		}
		fmt.Printf("Hosted path. No environment variables are required.\n\n  %s\n\n", clientAPIBase())
		fmt.Println("Sign in interactively:  insideout setup")
		fmt.Println("Or with the plain CLI:  insideout login you@example.com")
		fmt.Println("A server on this machine: insideout setup --local --postgres bundled|own")
		fmt.Println("Guide: docs/usage/installation.md")
		return true, 0
	case *local:
		if *postgres != "bundled" && *postgres != "own" {
			fmt.Fprintln(os.Stderr, "setup: --local needs --postgres bundled|own")
			return true, 2
		}
		if *createRoles && (*postgres != "own" || *noStart) {
			fmt.Fprintln(os.Stderr, "setup: --create-roles belongs to --local --postgres own (without --no-start, which never touches the database)")
			return true, 2
		}
		// Slice 2: the full local assembly (env write, Postgres,
		// pgx probe, in-process migrate, boot offer, summary gate).
		return runLocalSetup(*postgres, *noStart, *createRoles)
	default:
		if *postgres != "" || *noStart || *createRoles {
			fmt.Fprintln(os.Stderr, "setup: --postgres/--no-start/--create-roles belong to --local")
			return true, 2
		}
		return runInteractiveSetup()
	}
}

func runInteractiveSetup() (bool, int) {
	sc := &setupClient{}
	deps := flow.Deps{
		Prober:         setupProber{sc: sc},
		Client:         sc,
		Persister:      setupPersister{path: mustConfigPath()},
		DefaultBaseURL: defaultAPIBase,
		LocalSetup: func(ctx context.Context, postgres string) error {
			_, code := runLocalSetup(postgres, false, false)
			if code == 0 {
				return nil
			}
			return fmt.Errorf("local setup exited with code %d", code)
		},
	}
	res, err := ui.Run(context.Background(), deps)
	var noTTY ui.ErrNoTTY
	if errors.As(err, &noTTY) {
		fmt.Fprintln(os.Stderr, err)
		return true, 2
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return true, 1
	}
	// The local branch's own assembly already printed its summary and
	// boot offer; only the remote branch signs in and saves the token.
	if res.SignedIn {
		fmt.Fprintf(os.Stderr, "setup: signed in and saved the config (token not printed) — API base %s\n", res.APIBase)
		if res.RegisterMCP {
			fmt.Println("Merge the MCP registration: ./scripts/mcp-config.sh -q")
		}
		fmt.Println("Verify: insideout whoami")
	}
	return true, 0
}
