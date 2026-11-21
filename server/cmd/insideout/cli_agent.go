package main

// The agent's verbs on the CLI: roadmap drafts (`build`, `expand`) and the
// confirm verbs that write them, the agent briefing (`agent-context`),
// progress checkpoints, proposals, and the human's `proposal decide`.

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

// runDraftCommand handles the draft verbs (`build`, `expand`) and the
// confirm verbs that write them (`build-confirm`, `expand-confirm`).
func runDraftCommand(cmd string, args []string) (bool, int) {
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	api := apiFlag(fs)
	count := 0
	draft := ""
	force := false
	switch cmd {
	case "build-confirm":
		fs.IntVar(&count, "expected-count", 0, "live node count; required when replacing an existing roadmap")
		fs.BoolVar(&force, "force", false, "discard nodes that have evidence or are done or validated")
		fs.StringVar(&draft, "tree", "", "drafted tree JSON from build")
	case "expand-confirm":
		fs.StringVar(&draft, "children", "", "drafted children JSON from expand")
	}
	usage := map[string]string{
		"build":          " <prd-id>",
		"expand":         " <node-id>",
		"build-confirm":  " [-expected-count N] [-force] -tree '<json>' <prd-id>",
		"expand-confirm": " -children '<json>' <node-id>",
	}
	if err := fs.Parse(args); err != nil || len(fs.Args()) != 1 {
		fmt.Fprintf(os.Stderr, "usage: insideout %s%s\n", cmd, usage[cmd])
		return true, 2
	}
	if (cmd == "build-confirm" || cmd == "expand-confirm") && !json.Valid([]byte(draft)) {
		fmt.Fprintln(os.Stderr, "confirm needs the drafted JSON from the matching draft command")
		return true, 2
	}
	c := clientFromFlag(*api)
	switch cmd {
	case "build":
		return printJSON(c.BuildFromPrd(fs.Args()[0]))
	case "build-confirm":
		return printJSON(c.ConfirmBuildFromPrd(fs.Args()[0], json.RawMessage(draft), count, force))
	case "expand-confirm":
		return printJSON(c.ConfirmExpandNode(fs.Args()[0], json.RawMessage(draft)))
	default:
		return printJSON(c.ExpandNode(fs.Args()[0]))
	}
}

// runAgentCommand runs agent-context, checkpoint or propose.
func runAgentCommand(cmd string, args []string) (bool, int) {
	switch cmd {
	case "agent-context":
		return runAgentContext(args)
	case "checkpoint":
		return runAgentCheckpoint(args)
	}
	return runAgentPropose(args)
}

// runAgentContext is `insideout agent-context [--mode M] [--focus node] <project-id>`.
func runAgentContext(args []string) (bool, int) {
	fs := flag.NewFlagSet("agent-context", flag.ContinueOnError)
	api := apiFlag(fs)
	mode := fs.String("mode", "implementation", "brainstorming|implementation|review")
	focus := fs.String("focus", "", "focus node id")
	if err := fs.Parse(args); err != nil || len(fs.Args()) != 1 {
		fmt.Fprintln(os.Stderr, "usage: insideout agent-context [--mode M] [--focus node] <project-id>")
		return true, 2
	}
	return printJSON(clientFromFlag(*api).AgentContext(fs.Args()[0], *mode, *focus))
}

// runAgentCheckpoint is `insideout checkpoint [--node id] <project-id> <summary>`.
func runAgentCheckpoint(args []string) (bool, int) {
	fs := flag.NewFlagSet("checkpoint", flag.ContinueOnError)
	api := apiFlag(fs)
	node := fs.String("node", "", "roadmap node id (optional)")
	if err := fs.Parse(args); err != nil || len(fs.Args()) != 2 {
		fmt.Fprintln(os.Stderr, "usage: insideout checkpoint [--node id] <project-id> <summary>")
		return true, 2
	}
	return printJSON(clientFromFlag(*api).AgentCheckpoint(fs.Args()[0], *node, fs.Args()[1]))
}

// runAgentPropose is `insideout propose`: each --item is Title[@ParentTitle],
// split at the first "@".
func runAgentPropose(args []string) (bool, int) {
	fs := flag.NewFlagSet("propose", flag.ContinueOnError)
	api := apiFlag(fs)
	kind := fs.String("kind", "structure", "structure|scope|priority")
	detail := fs.String("detail", "", "extra detail")
	var items multiFlag
	fs.Var(&items, "item", "structured item Title[@ParentTitle], appliable on acceptance (repeatable)")
	if err := fs.Parse(args); err != nil || len(fs.Args()) != 2 {
		fmt.Fprintln(os.Stderr, "usage: insideout propose --kind K [--detail D] [--item Title[@Parent]]... <project-id> <summary>")
		return true, 2
	}
	proposed := make([]apiclient.ProposeItem, len(items))
	for i, it := range items {
		title, hint, _ := strings.Cut(it, "@")
		proposed[i] = apiclient.ProposeItem{Title: title, ParentHint: hint}
	}
	return printJSON(clientFromFlag(*api).AgentPropose(fs.Args()[0], *kind, fs.Args()[1], *detail, proposed))
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
