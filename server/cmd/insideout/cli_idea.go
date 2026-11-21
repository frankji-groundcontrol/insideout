package main

// `insideout idea create|convert` — the idea write verbs
// (docs/plans/2026-08-21-cli-mcp-parity.md Stage 3; update and delete run in
// cli_crud.go) — and `insideout proposal decide`.

import (
	"flag"
	"fmt"
	"os"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

func runIdeaCommand(args []string) (bool, int) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: insideout idea <create|update|delete|convert|proposal-decide> …")
		return true, 2
	}
	sub, rest := args[0], args[1:]

	switch sub {
	case "proposal-decide": // the old spelling of `proposal decide`
		return runProposalDecide(rest)
	case "create":
		fs := flag.NewFlagSet("idea create", flag.ContinueOnError)
		api := apiFlag(fs)
		title := fs.String("title", "", "idea title (required)")
		content := fs.String("content", "", "idea body")
		if err := fs.Parse(rest); err != nil || len(fs.Args()) != 1 || *title == "" {
			fmt.Fprintln(os.Stderr, "usage: insideout idea create --title T [--content C] <workspace-id>")
			return true, 2
		}
		return printJSON(clientFromFlag(*api).IdeaCreate(fs.Args()[0], *title, *content))
	case "convert":
		fs := flag.NewFlagSet("idea convert", flag.ContinueOnError)
		api := apiFlag(fs)
		locale := fs.String("locale", "", "language the coach speaks, en or zh (default: LC_ALL, LC_MESSAGES or LANG, else en)")
		if err := fs.Parse(rest); err != nil || len(fs.Args()) != 1 {
			fmt.Fprintln(os.Stderr, "usage: insideout idea convert [--locale en|zh] <idea-id>")
			return true, 2
		}
		lang, err := apiclient.ConvertLocale(*locale)
		if err != nil {
			fmt.Fprintln(os.Stderr, "idea convert:", err)
			return true, 2
		}
		return printJSON(clientFromFlag(*api).IdeaConvert(fs.Args()[0], lang))
	}
	fmt.Fprintln(os.Stderr, "unknown idea subcommand:", sub)
	return true, 2
}

// runProposalDecide is `insideout proposal decide` (and the old
// `idea proposal-decide`): a human accepts or rejects an agent proposal.
// Acceptance applies the proposal's items unless --no-apply, the default the
// web carries too (rv5-PARB-05); --apply=false, the old spelling, still holds.
func runProposalDecide(args []string) (bool, int) {
	fs := flag.NewFlagSet("proposal decide", flag.ContinueOnError)
	api := apiFlag(fs)
	accept := fs.Bool("accept", false, "accept the proposal")
	reject := fs.Bool("reject", false, "reject the proposal")
	reason := fs.String("reason", "", "why")
	noApply := fs.Bool("no-apply", false, "accept without applying the proposal's items")
	apply := fs.Bool("apply", true, "apply the proposal's items on acceptance (the default)")
	if err := fs.Parse(args); err != nil || len(fs.Args()) != 1 || *accept == *reject {
		fmt.Fprintln(os.Stderr, "usage: insideout proposal decide --accept|--reject [--no-apply] [--reason R] <update-id>")
		return true, 2
	}
	decision := "rejected"
	if *accept {
		decision = "accepted"
	}
	return printJSON(clientFromFlag(*api).DecideProposal(fs.Args()[0], decision, *reason, *accept && *apply && !*noApply))
}
