package main

// Roadmap CLI verbs — the roadmap projection of the /api/v1 contract
// (docs/plans/2026-08-21-roadmap-parity-and-github.md). Agent planning
// (build, expand), branch CRUD (roadmap add/update/move/delete), and
// GitHub binding live here; reads print the API's raw JSON.

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

// runRoadmapCommand handles `insideout roadmap
// <list|show|mine|add|update|move|delete|delegate|handback|assignments|progress|presence> …`.
// Returns (handled, exitCode).
func runRoadmapCommand(args []string) (bool, int) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: insideout roadmap <list|show|mine|add|update|move|delete|delegate|handback|assignments|progress|presence> …")
		return true, 2
	}
	sub, rest := args[0], args[1:]
	if handled, code := runRoadmapTreeCommand(sub, rest); handled {
		return handled, code
	}

	switch sub {
	case "presence":
		fs := flag.NewFlagSet("presence", flag.ContinueOnError)
		api := apiFlag(fs)
		if err := fs.Parse(rest); err != nil || len(fs.Args()) != 1 {
			fmt.Fprintln(os.Stderr, "usage: insideout presence <project-id>")
			return true, 2
		}
		return printJSON(clientFromFlag(*api).ProjectPresence(fs.Args()[0]))
	case "add":
		fs := flag.NewFlagSet("roadmap add", flag.ContinueOnError)
		api := apiFlag(fs)
		title := fs.String("title", "", "node title (required)")
		description := fs.String("description", "", "node description")
		parent := fs.String("parent", "", "parent node id (empty = root)")
		if err := fs.Parse(rest); err != nil || len(fs.Args()) != 1 || *title == "" {
			fmt.Fprintln(os.Stderr, "usage: insideout roadmap add --title T [--description D] [--parent id] <project-id>")
			return true, 2
		}
		var parentID *string
		if *parent != "" {
			parentID = parent
		}
		return printJSON(clientFromFlag(*api).RoadmapAdd(fs.Args()[0], *title, *description, parentID))
	case "move":
		fs := flag.NewFlagSet("roadmap move", flag.ContinueOnError)
		api := apiFlag(fs)
		parent := fs.String("parent", "", "new parent id (omit to keep the parent)")
		root := fs.Bool("root", false, "make the node a root")
		position := fs.Int("position", -1, "position among siblings")
		if err := fs.Parse(rest); err != nil || len(fs.Args()) != 1 || (*root && *parent != "") {
			fmt.Fprintln(os.Stderr, "usage: insideout roadmap move [--parent id | --root] [--position n] <node-id>")
			return true, 2
		}
		var parentID *string // nil keeps the parent; "" asks for a root
		if *root || *parent != "" {
			parentID = parent
		}
		var pos *int
		if *position >= 0 {
			pos = position
		}
		return printJSON(clientFromFlag(*api).RoadmapMove(fs.Args()[0], parentID, pos))
	case "delete":
		fs := flag.NewFlagSet("roadmap delete", flag.ContinueOnError)
		api := apiFlag(fs)
		force := fs.Bool("force", false, "delete even when the branch holds evidence or done or validated nodes")
		if err := fs.Parse(rest); err != nil || len(fs.Args()) != 1 {
			fmt.Fprintln(os.Stderr, "usage: insideout roadmap delete [--force] <node-id>")
			return true, 2
		}
		err := clientFromFlag(*api).RoadmapDelete(fs.Args()[0], *force)
		var h *apiclient.DeleteHistoryError
		if errors.As(err, &h) {
			fmt.Fprintf(os.Stderr, "not deleted: deleting this branch would erase %d node(s) holding %d evidence row(s), %d done and %d validated.\nRe-run with --force to delete it anyway: insideout roadmap delete --force %s\n",
				h.LiveCount, h.EvidenceCount, h.DoneCount, h.ValidatedCount, fs.Args()[0])
			return true, 1
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return true, 1
		}
		fmt.Println("deleted")
		return true, 0
	}
	fmt.Fprintln(os.Stderr, "unknown roadmap subcommand:", sub)
	return true, 2
}

// runAgentCommand handles the draft verbs (`build`, `expand`) and the
// confirm verbs that write them (`build-confirm`, `expand-confirm`).
func runAgentCommand(cmd string, args []string) (bool, int) {
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

// runGithubCommand handles `insideout repo set <project-id> <repo-url>`
// and `insideout sync <project-id>`.
func runGithubCommand(cmd string, args []string) (bool, int) {
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	api := apiFlag(fs)
	if err := fs.Parse(args); err != nil {
		return true, 2
	}
	c := clientFromFlag(*api)
	switch cmd {
	case "repo":
		if len(fs.Args()) != 3 || fs.Args()[0] != "set" {
			fmt.Fprintln(os.Stderr, "usage: insideout repo set <project-id> <repo-url>")
			return true, 2
		}
		return printJSON(c.SetRepo(fs.Args()[1], fs.Args()[2]))
	case "sync":
		if len(fs.Args()) != 1 {
			fmt.Fprintln(os.Stderr, "usage: insideout sync <project-id>")
			return true, 2
		}
		return printJSON(c.SyncGithub(fs.Args()[0]))
	}
	return false, 0
}
