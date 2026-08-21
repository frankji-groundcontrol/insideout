package main

// Product-facing CLI subcommands — the CLI projection of the /api/v1
// contract (docs/plans/2026-08-21-cli-mcp-parity.md). These run before
// server config and DB pools: a CLI user needs only INSIDEOUT_API
// (default: hosted API) and INSIDEOUT_TOKEN. Read output is the API's
// raw JSON, indented, so the CLI can never drift from the truth.

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

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
	switch cmd {
	case "roadmap":
		return runRoadmapCommand(rest, envOr("INSIDEOUT_API", defaultAPIBase))
	case "build", "expand":
		return runAgentCommand(cmd, rest, envOr("INSIDEOUT_API", defaultAPIBase))
	case "repo", "sync":
		return runGithubCommand(cmd, rest, envOr("INSIDEOUT_API", defaultAPIBase))
	case "guide":
		return runGuideCommand(rest, envOr("INSIDEOUT_API", defaultAPIBase))
	case "login", "whoami", "workspaces", "projects", "prd":
	default:
		return false, 0
	}

	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	api := fs.String("api", envOr("INSIDEOUT_API", defaultAPIBase), "API base URL")
	if err := fs.Parse(rest); err != nil {
		return true, 2
	}
	c := apiclient.New(*api)
	if cmd != "login" {
		c.SetToken(os.Getenv("INSIDEOUT_TOKEN"))
	}

	switch cmd {
	case "login":
		email, password, err := credentials(fs.Args())
		if err != nil {
			fmt.Fprintln(os.Stderr, "login:", err)
			return true, 2
		}
		if err := c.Login(email, password); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return true, 1
		}
		fmt.Println(c.Token())
		return true, 0
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
	fmt.Fprint(os.Stderr, "password: ")
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		return "", "", fmt.Errorf("read password: %w", err)
	}
	pw := strings.TrimSpace(line)
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
