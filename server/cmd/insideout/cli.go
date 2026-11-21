package main

// Product-facing CLI subcommands — the CLI projection of the /api/v1
// contract (docs/plans/2026-08-21-cli-mcp-parity.md). These run before
// server config and DB pools: a CLI user needs only INSIDEOUT_API
// (default: hosted API) and INSIDEOUT_TOKEN. Read output is the API's
// raw JSON, indented, so the CLI can never drift from the truth. This file
// holds the helpers every verb shares; cli_dispatch.go routes the verbs.

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
	"github.com/frankji-groundcontrol/insideout/server/internal/clientconfig"
)

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
