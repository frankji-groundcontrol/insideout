package main

// Product-facing CLI subcommands — the CLI projection of the /api/v1
// contract (docs/plans/2026-08-21-cli-mcp-parity.md). These run before
// server config and DB pools: a CLI user needs only INSIDEOUT_API
// (default: hosted API) and INSIDEOUT_TOKEN. Read output is the API's
// raw JSON, indented, so the CLI can never drift from the truth.

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
	"github.com/frankji-groundcontrol/insideout/server/internal/clientconfig"
)

// cliCtx is the context of the command now running. Ctrl-C or SIGTERM cancels
// it (runInterruptible) and every client a verb builds is bound to it, so the
// request in flight ends with the command. It is context.Background when no
// command is running through runInterruptible, as in a test that calls a verb.
var cliCtx = context.Background()

// interruptGrace is how long an interrupted command gets to wind down (its
// request aborts, its error prints) before the process ends anyway. A command
// that waits at a prompt has no request to abort.
var interruptGrace = 400 * time.Millisecond

// runInterruptible is runClientCommand under cliCtx. The first Ctrl-C (or
// SIGTERM) cancels the context and the command ends with status 130, the
// shell's code for an interrupted command; a command still running after
// interruptGrace, or a second signal, ends the process at once. The one thing
// an interrupt must not cut is the renewal of an expired session: the server
// rotates the refresh token as it reads the request, so a process that ends
// before the new pair is saved leaves the saved login holding a spent token.
// apiclient lets a renewal in flight finish (renewalDrain) and the forced exit
// waits for it (WaitForRenewal); only an answer slower than that, or a second
// signal, can still lose the pair (review-4 QUAL-04, rv4-FR-F66).
func runInterruptible(args []string) (handled bool, exitCode int) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	cliCtx = ctx
	finished := make(chan struct{})
	go func() {
		select {
		case <-sigs:
			cancel()
			signal.Stop(sigs) // the next signal takes the default action
			select {
			case <-finished:
			case <-time.After(interruptGrace):
				apiclient.WaitForRenewal()
				fmt.Fprintln(os.Stderr, "\ninsideout: interrupted")
				os.Exit(130)
			}
		case <-finished:
		}
	}()
	handled, exitCode = runClientCommand(args)
	interrupted := ctx.Err() != nil
	close(finished)
	signal.Stop(sigs)
	cancel()
	cliCtx = context.Background()
	if handled && interrupted && exitCode != 0 {
		fmt.Fprintln(os.Stderr, "insideout: interrupted")
		return true, 130
	}
	return handled, exitCode
}

// runClientCommand handles login/whoami/workspaces/projects/prd and
// reports whether the argument was a client command (already executed;
// the caller should exit with the returned code).
func runClientCommand(args []string) (handled bool, exitCode int) {
	if len(args) == 0 {
		return false, 0
	}
	cmd, rest := args[0], args[1:]

	// CRUD-completion sub-verbs piggyback on the existing read verbs
	// (idea/prd/workspaces/projects) and add ideas/evidence.
	if len(rest) > 0 {
		switch cmd + " " + rest[0] {
		case "idea update", "idea delete", "prd update", "workspaces create", "projects create":
			return runCrudCommand(cmd+" "+rest[0], rest[1:])
		}
	}
	switch cmd {
	case "setup":
		return runSetupCommand(rest)
	case "health":
		// Container probe: no API client, no flags — see cli_health.go.
		return true, runHealthCommand()
	case "ideas", "evidence", "join":
		return runCrudCommand(cmd, rest)
	case "roadmap":
		return runRoadmapCommand(rest)
	case "build", "expand", "build-confirm", "expand-confirm":
		return runAgentCommand(cmd, rest)
	case "repo", "sync":
		return runGithubCommand(cmd, rest)
	case "guide":
		return runGuideCommand(rest)
	case "commit", "versions", "readiness", "view", "agent-context", "checkpoint", "propose", "revisions", "snapshot":
		return runVersionCommands(cmd, rest)
	case "presence":
		return runRoadmapCommand(append([]string{"presence"}, rest...))
	case "members":
		return runMembersCommand(rest)
	case "idea":
		return runIdeaCommand(rest)
	case "login", "register", "forgot-password", "reset-password", "logout":
		return runAuthCommand(cmd, rest)
	case "invite", "invites", "signup-mode":
		return runAdminCommand(cmd, rest)
	case "whoami", "workspaces", "projects", "prd":
	default:
		return false, 0
	}

	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	api := apiFlag(fs)
	if err := fs.Parse(rest); err != nil {
		return true, 2
	}
	c := clientFromFlag(*api)

	switch cmd {
	case "whoami":
		return printJSON(c.Whoami())
	case "workspaces":
		return printJSON(c.Workspaces())
	case "projects":
		if len(fs.Args()) != 1 {
			fmt.Fprintln(os.Stderr, "usage: insideout projects <workspace-id>")
			return true, 2
		}
		return printJSON(c.Projects(fs.Args()[0]))
	case "prd":
		if len(fs.Args()) != 1 {
			fmt.Fprintln(os.Stderr, "usage: insideout prd <prd-id>")
			return true, 2
		}
		return printJSON(c.Prd(fs.Args()[0]))
	}
	return false, 0
}

// apiFlag registers the shared --api flag. It stays empty unless the operator
// passes it, so clientconfig.Resolve alone decides the base (--api, then
// INSIDEOUT_API, then the saved login, then the hosted API).
func apiFlag(fs *flag.FlagSet) *string {
	return fs.String("api", "", "API base URL (default: INSIDEOUT_API, the saved login, then the hosted API)")
}

// clientFromFlag builds a verb's API client from its parsed --api value.
// Call it only after fs.Parse: a client built earlier cannot see the flag.
// When the credentials cannot be resolved (an unreadable config file) the
// client fails every request with that error, so the verb reports it
// instead of falling back to the hosted API.
func clientFromFlag(flagAPI string) *apiclient.Client {
	creds, err := clientconfig.Resolve(flagAPI)
	if err != nil {
		return apiclient.Failing(err)
	}
	c := creds.Client()
	c.SetContext(cliCtx)
	return c
}

func credentials(args []string) (string, string, error) {
	if len(args) != 1 {
		return "", "", fmt.Errorf("usage: insideout login <email>  (password is read from stdin)")
	}
	pw, err := readSecret("password: ")
	if err != nil {
		return "", "", err
	}
	if pw == "" {
		return "", "", fmt.Errorf("empty password")
	}
	return args[0], pw, nil
}

func printJSON(raw json.RawMessage, err error) (bool, int) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return true, 1
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err != nil {
		fmt.Println(string(raw))
		return true, 0
	}
	fmt.Println(buf.String())
	return true, 0
}

// savedLoginFor is the config file's login for base, or a zero config when
// nothing was saved for that server (missing file, or another server's login).
func savedLoginFor(base string) (clientconfig.Config, error) {
	path, err := clientconfig.DefaultPath()
	if err != nil {
		return clientconfig.Config{}, nil
	}
	cfg, err := clientconfig.Load(path)
	if err != nil || !basesMatch(base, savedBase(cfg.APIBase)) {
		return clientconfig.Config{}, err
	}
	return cfg, nil
}

// forgetSavedLogin removes the saved login for base. The file goes when it
// holds nothing else; any other field stays and only the token fields clear.
func forgetSavedLogin(base string) error {
	path, err := clientconfig.DefaultPath()
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	saved, _ := raw["api_base"].(string)
	if !basesMatch(base, savedBase(saved)) {
		return nil
	}
	extra := false
	for key := range raw {
		if key != "api_base" && key != "token" && key != "refresh_token" {
			extra = true
		}
	}
	if !extra {
		return os.Remove(path)
	}
	delete(raw, "token")
	delete(raw, "refresh_token")
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0o600)
}

func savedBase(apiBase string) string {
	if apiBase == "" {
		return apiclient.DefaultBase
	}
	return apiBase
}

func basesMatch(a, b string) bool {
	return strings.TrimRight(a, "/") == strings.TrimRight(b, "/")
}
