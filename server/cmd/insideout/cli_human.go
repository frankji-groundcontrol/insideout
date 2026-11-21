package main

// The human-only verbs (rv5-PARA-03, PARC-04): account deletion, a validation
// observation and the validate confirm. The server refuses an agent token on
// all three, so MCP has no tool for them; the CLI runs on the human's login.

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
	"github.com/frankji-groundcontrol/insideout/server/internal/clientconfig"
)

var humanUsage = map[string]string{
	"account delete":   "insideout account delete  (the current password is read from stdin, never a flag)",
	"evidence add":     "insideout evidence add --detail D [--source URL] [--observed YYYY-MM-DD] <node-id>",
	"roadmap validate": "insideout roadmap validate --reason R <node-id>",
}

// runHumanCommand runs one of the humanUsage verbs; any other verb (a bare
// `account`) prints the account usage.
func runHumanCommand(verb string, args []string) (bool, int) {
	usage, ok := humanUsage[verb]
	if !ok {
		usage = humanUsage["account delete"]
	}
	fs := flag.NewFlagSet(verb, flag.ContinueOnError)
	api := apiFlag(fs)
	var detail, source, observed, reason *string
	wantIDs := 1
	switch verb {
	case "evidence add":
		detail = fs.String("detail", "", "what was observed")
		source = fs.String("source", "", "where it was observed: a link")
		observed = fs.String("observed", time.Now().Format(time.DateOnly), "the day it was observed")
	case "roadmap validate":
		reason = fs.String("reason", "", "why the node counts as validated")
	default:
		wantIDs = 0
	}
	bad := !ok || fs.Parse(args) != nil || len(fs.Args()) != wantIDs
	if detail != nil {
		_, dateErr := time.Parse(time.DateOnly, *observed)
		bad = bad || *detail == "" || dateErr != nil
	}
	if bad || (reason != nil && *reason == "") {
		fmt.Fprintln(os.Stderr, "usage: "+usage)
		return true, 2
	}
	c := clientFromFlag(*api)
	switch verb {
	case "evidence add":
		return printJSON(c.AddObservation(fs.Args()[0], apiclient.Observation{Detail: *detail, SourceURL: *source, ObservedAt: *observed}))
	case "roadmap validate":
		return printJSON(c.ValidateNode(fs.Args()[0], *reason))
	}
	password, err := readSecret("current password: ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return true, 1
	}
	if password == "" {
		fmt.Fprintln(os.Stderr, "account delete: empty password")
		return true, 2
	}
	if err := c.DeleteMe(password); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return true, 1
	}
	// Every session died with the account: drop the saved one too.
	if creds, err := clientconfig.Resolve(*api); err == nil {
		if err := forgetSavedLogin(creds.Base); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
	fmt.Println("account deleted")
	return true, 0
}
