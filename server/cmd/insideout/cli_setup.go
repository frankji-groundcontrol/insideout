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
	s.c = newSetupAPIClient(base)
}

func (s *setupClient) client() *apiclient.Client {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.c == nil {
		s.c = newSetupAPIClient(apiclient.DefaultBase)
	}
	return s.c
}

// newSetupAPIClient is apiclient.New bound to the running command's context, so
// an interrupted setup ends the sign-in request it is waiting on.
func newSetupAPIClient(base string) *apiclient.Client {
	c := apiclient.New(base)
	c.SetContext(cliCtx)
	return c
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

func (s *setupClient) RefreshToken() string { return s.client().RefreshToken() }

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
		fmt.Printf("Hosted path. No environment variables are required.\n\n  %s\n\n", apiclient.DefaultBase)
		fmt.Print(`Sign in (the password is read from the terminal; the login is saved for the
CLI and the MCP server):

  insideout login you@example.com
  insideout whoami

No account yet? Signing up on the hosted service takes an invitation code:
ask the server's administrator for one, then

  insideout register --invite <invitation-code> you@example.com

Headless or CI: set INSIDEOUT_TOKEN to an access token instead of signing in.
It wins over the saved login and is never renewed.

A local server is a different API. Set INSIDEOUT_API=http://127.0.0.1:8080/api/v1
only after ./scripts/setup.sh --local has finished.
Guide: docs/usage/installation.md
`)
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
		DefaultBaseURL: apiclient.DefaultBase,
		LocalSetup: func(ctx context.Context, postgres string) error {
			_, code := runLocalSetup(postgres, false, false)
			if code == 0 {
				return nil
			}
			return fmt.Errorf("local setup exited with code %d", code)
		},
	}
	res, err := ui.Run(cliCtx, deps)
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
			fmt.Println(`Merge the MCP registration: {"mcpServers":{"insideout":{"command":"insideout-mcp"}}} (docs/usage/cli-mcp-and-guidance.md)`)
		}
		fmt.Println("Verify: insideout whoami")
	}
	return true, 0
}
