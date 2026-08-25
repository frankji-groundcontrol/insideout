package main

// Admin CLI verbs: invitation management and the signup-mode switch.
// Raw-JSON output like the rest of the CLI — the API's answer is the
// truth; the create response (with the one-time code) prints verbatim.

import (
	"flag"
	"fmt"
	"os"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

func runAdminCommand(cmd string, args []string, api string) (bool, int) {
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	ttl := fs.Int("ttl", 0, "invitation lifetime in hours (default 168 = 7 days)")
	if err := fs.Parse(args); err != nil {
		return true, 2
	}
	c := apiclient.New(api)
	c.SetToken(os.Getenv("INSIDEOUT_TOKEN"))

	switch cmd {
	case "invite":
		if len(fs.Args()) != 1 {
			fmt.Fprintln(os.Stderr, "usage: insideout invite [--ttl hours] <email>")
			return true, 2
		}
		return printJSON(c.CreateInvitation(fs.Args()[0], *ttl))
	case "invites":
		return printJSON(c.ListInvitations())
	case "signup-mode":
		if len(fs.Args()) == 0 {
			mode, err := c.AuthConfig()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return true, 1
			}
			fmt.Println(mode)
			return true, 0
		}
		mode := fs.Args()[0]
		if mode != "invitation_only" && mode != "open" && mode != "closed" {
			fmt.Fprintln(os.Stderr, "signup-mode: must be invitation_only, open, or closed")
			return true, 2
		}
		return printJSON(c.SetSignupMode(mode))
	}
	return false, 0
}
