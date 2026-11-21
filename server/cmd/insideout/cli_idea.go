package main

// The idea inbox on the CLI: `insideout ideas` lists a workspace's ideas and
// `insideout idea create|update|delete|convert` writes them
// (docs/plans/2026-08-21-cli-mcp-parity.md Stage 3,
// docs/plans/2026-08-25-agent-crud-tool-surface.md).

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
	case "update":
		return runIdeaUpdate(rest)
	case "delete":
		return runIdeaDelete(rest)
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

// runIdeas is `insideout ideas <workspace-id>`.
func runIdeas(args []string) (bool, int) {
	f, ok := parseWriteFlags("ideas", args)
	if !ok || !f.need(1, "insideout ideas <workspace-id>") {
		return true, 2
	}
	return printJSON(clientFromFlag(*f.api).Ideas(f.positional[0]))
}

// runIdeaUpdate is `insideout idea update`. An absent or empty --content
// keeps the body (PARA-02); the word clear clears it.
func runIdeaUpdate(args []string) (bool, int) {
	f, ok := parseWriteFlags("idea update", args)
	if !ok || !f.need(1, "insideout idea update --title T [--content C|clear] <id>") {
		return true, 2
	}
	return printJSON(clientFromFlag(*f.api).UpdateIdea(f.positional[0], *f.title, apiclient.PatchText(*f.content)))
}

// runIdeaDelete is `insideout idea delete <id>`, a soft delete.
func runIdeaDelete(args []string) (bool, int) {
	f, ok := parseWriteFlags("idea delete", args)
	if !ok || !f.need(1, "insideout idea delete <id>") {
		return true, 2
	}
	if err := clientFromFlag(*f.api).DeleteIdea(f.positional[0]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return true, 1
	}
	fmt.Println("dropped (soft — it no longer appears in your ideas list)")
	return true, 0
}
