package main

// Product-facing CLI subcommands — the CLI projection of the /api/v1
// contract (docs/plans/2026-08-21-cli-mcp-parity.md). These run before
// server config and DB pools: a CLI user needs only INSIDEOUT_API
// (default: hosted API) and INSIDEOUT_TOKEN. Read output is the API's
// raw JSON, indented, so the CLI can never drift from the truth.

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

const defaultAPIBase = "https://server-production-9c338.up.railway.app/api/v1"

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
	apiBase := envOr("INSIDEOUT_API", defaultAPIBase)
	if len(rest) > 0 {
		switch cmd + " " + rest[0] {
		case "idea update", "idea delete", "prd update", "workspaces create", "projects create":
			return runCrudCommand(cmd+" "+rest[0], rest[1:], apiBase)
		}
	}
	switch cmd {
	case "ideas", "evidence":
		return runCrudCommand(cmd, rest, apiBase)
	case "roadmap":
		return runRoadmapCommand(rest, envOr("INSIDEOUT_API", defaultAPIBase))
	case "build", "expand":
		return runAgentCommand(cmd, rest, envOr("INSIDEOUT_API", defaultAPIBase))
	case "repo", "sync":
		return runGithubCommand(cmd, rest, envOr("INSIDEOUT_API", defaultAPIBase))
	case "guide":
		return runGuideCommand(rest, envOr("INSIDEOUT_API", defaultAPIBase))
	case "commit", "versions", "readiness", "view", "agent-context", "checkpoint", "propose", "revisions", "snapshot":
		return runVersionCommands(cmd, rest, envOr("INSIDEOUT_API", defaultAPIBase))
	case "presence":
		return runRoadmapCommand(append([]string{"presence"}, rest...), envOr("INSIDEOUT_API", defaultAPIBase))
	case "idea":
		return runIdeaCommand(rest, envOr("INSIDEOUT_API", defaultAPIBase))
	case "login", "register", "forgot-password", "reset-password":
		return runAuthCommand(cmd, rest, envOr("INSIDEOUT_API", defaultAPIBase))
	case "invite", "invites", "signup-mode":
		return runAdminCommand(cmd, rest, envOr("INSIDEOUT_API", defaultAPIBase))
	case "whoami", "workspaces", "projects", "prd":
	default:
		return false, 0
	}

	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	api := fs.String("api", envOr("INSIDEOUT_API", defaultAPIBase), "API base URL")
	if err := fs.Parse(rest); err != nil {
		return true, 2
	}
	c := apiclient.New(*api)
	c.SetToken(os.Getenv("INSIDEOUT_TOKEN"))

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

func credentials(args []string) (string, string, error) {
	if len(args) != 1 {
		return "", "", fmt.Errorf("usage: insideout login <email>  (password is read from stdin)")
	}
	pw := prompt("password: ")
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

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
