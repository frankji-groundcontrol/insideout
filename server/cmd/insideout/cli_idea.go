package main

// `insideout idea create|convert` — the idea write verbs
// (docs/plans/2026-08-21-cli-mcp-parity.md Stage 3).

import (
	"flag"
	"fmt"
	"os"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

func runIdeaCommand(args []string, api string) (bool, int) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: insideout idea <create|convert> …")
		return true, 2
	}
	sub, rest := args[0], args[1:]
	c := apiclient.New(api)
	c.SetToken(os.Getenv("INSIDEOUT_TOKEN"))

	switch sub {
	case "create":
		fs := flag.NewFlagSet("idea create", flag.ContinueOnError)
		applyAPIFlag(fs, &api)
		title := fs.String("title", "", "idea title (required)")
		content := fs.String("content", "", "idea body")
		if err := fs.Parse(rest); err != nil || len(fs.Args()) != 1 || *title == "" {
			fmt.Fprintln(os.Stderr, "usage: insideout idea create --title T [--content C] <workspace-id>")
			return true, 2
		}
		return printJSON(c.IdeaCreate(fs.Args()[0], *title, *content))
	case "proposal-decide":
		fs := flag.NewFlagSet("proposal-decide", flag.ContinueOnError)
		applyAPIFlag(fs, &api)
		accept := fs.Bool("accept", false, "accept the proposal")
		reject := fs.Bool("reject", false, "reject the proposal")
		reason := fs.String("reason", "", "why")
		if err := fs.Parse(rest); err != nil || len(fs.Args()) != 1 || *accept == *reject {
			fmt.Fprintln(os.Stderr, "usage: insideout idea proposal-decide --accept|--reject [--reason R] <update-id>")
			return true, 2
		}
		decision := "rejected"
		if *accept {
			decision = "accepted"
		}
		return printJSON(c.DecideProposal(fs.Args()[0], decision, *reason))
	case "convert":
		fs := flag.NewFlagSet("idea convert", flag.ContinueOnError)
		applyAPIFlag(fs, &api)
		if err := fs.Parse(rest); err != nil || len(fs.Args()) != 1 {
			fmt.Fprintln(os.Stderr, "usage: insideout idea convert <idea-id>")
			return true, 2
		}
		return printJSON(c.IdeaConvert(fs.Args()[0]))
	}
	fmt.Fprintln(os.Stderr, "unknown idea subcommand:", sub)
	return true, 2
}
