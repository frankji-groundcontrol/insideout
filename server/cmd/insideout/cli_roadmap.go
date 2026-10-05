package main

// Roadmap CLI verbs — the roadmap projection of the /api/v1 contract
// (docs/plans/2026-08-21-roadmap-parity-and-github.md). Agent planning
// (build, expand), branch CRUD (roadmap add/update/move/delete), and
// GitHub binding live here; reads print the API's raw JSON.

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

// runRoadmapCommand handles `insideout roadmap
// <list|add|update|move|delete|progress|presence> …`.
// Returns (handled, exitCode).
func runRoadmapCommand(args []string, api string) (bool, int) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: insideout roadmap <list|add|update|move|delete|progress|presence> …")
		return true, 2
	}
	sub, rest := args[0], args[1:]
	c := apiclient.New(api)
	c.SetToken(os.Getenv("INSIDEOUT_TOKEN"))

	switch sub {
	case "progress":
		fs := flag.NewFlagSet("roadmap progress", flag.ContinueOnError)
		applyAPIFlag(fs, &api)
		if err := fs.Parse(rest); err != nil || len(fs.Args()) != 1 {
			fmt.Fprintln(os.Stderr, "usage: insideout roadmap progress <project-id>")
			return true, 2
		}
		return printJSON(c.RoadmapProgress(fs.Args()[0]))
	case "presence":
		fs := flag.NewFlagSet("presence", flag.ContinueOnError)
		applyAPIFlag(fs, &api)
		if err := fs.Parse(rest); err != nil || len(fs.Args()) != 1 {
			fmt.Fprintln(os.Stderr, "usage: insideout presence <project-id>")
			return true, 2
		}
		return printJSON(c.ProjectPresence(fs.Args()[0]))
	case "list":
		fs := flag.NewFlagSet("roadmap list", flag.ContinueOnError)
		applyAPIFlag(fs, &api)
		if err := fs.Parse(rest); err != nil || len(fs.Args()) != 1 {
			fmt.Fprintln(os.Stderr, "usage: insideout roadmap list <project-id>")
			return true, 2
		}
		return printJSON(c.RoadmapList(fs.Args()[0]))
	case "add":
		fs := flag.NewFlagSet("roadmap add", flag.ContinueOnError)
		applyAPIFlag(fs, &api)
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
		return printJSON(c.RoadmapAdd(fs.Args()[0], *title, *description, parentID))
	case "update":
		fs := flag.NewFlagSet("roadmap update", flag.ContinueOnError)
		applyAPIFlag(fs, &api)
		title := fs.String("title", "", "new title")
		description := fs.String("description", "", "new description")
		status := fs.String("status", "", "locked|pending|in_progress|done")
		deadline := fs.String("deadline", "", "RFC3339 deadline, or clear")
		if err := fs.Parse(rest); err != nil || len(fs.Args()) != 1 ||
			(*title == "" && *description == "" && *status == "" && *deadline == "") {
			fmt.Fprintln(os.Stderr, "usage: insideout roadmap update [--title T] [--description D] [--status S] [--deadline RFC3339|clear] <node-id>")
			return true, 2
		}
		var dl *string
		if *deadline == "clear" {
			empty := ""
			dl = &empty
		} else if *deadline != "" {
			if _, err := time.Parse(time.RFC3339, *deadline); err != nil {
				fmt.Fprintln(os.Stderr, "deadline must be RFC3339 or clear")
				return true, 2
			}
			dl = deadline
		}
		return printJSON(c.RoadmapUpdate(fs.Args()[0], strPtrIfSet(*title), strPtrIfSet(*description), strPtrIfSet(*status), dl))
	case "move":
		fs := flag.NewFlagSet("roadmap move", flag.ContinueOnError)
		applyAPIFlag(fs, &api)
		parent := fs.String("parent", "", "new parent id (empty = root)")
		position := fs.Int("position", -1, "position among siblings")
		if err := fs.Parse(rest); err != nil || len(fs.Args()) != 1 {
			fmt.Fprintln(os.Stderr, "usage: insideout roadmap move [--parent id] [--position n] <node-id>")
			return true, 2
		}
		var parentID *string
		if *parent != "" {
			parentID = parent
		}
		var pos *int
		if *position >= 0 {
			pos = position
		}
		return printJSON(c.RoadmapMove(fs.Args()[0], parentID, pos))
	case "delete":
		fs := flag.NewFlagSet("roadmap delete", flag.ContinueOnError)
		applyAPIFlag(fs, &api)
		if err := fs.Parse(rest); err != nil || len(fs.Args()) != 1 {
			fmt.Fprintln(os.Stderr, "usage: insideout roadmap delete <node-id>")
			return true, 2
		}
		if err := c.RoadmapDelete(fs.Args()[0]); err != nil {
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
func runAgentCommand(cmd string, args []string, api string) (bool, int) {
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	applyAPIFlag(fs, &api)
	count := 0
	draft := ""
	switch cmd {
	case "build-confirm":
		fs.IntVar(&count, "expected-count", 0, "live node count; required when replacing an existing roadmap")
		fs.StringVar(&draft, "tree", "", "drafted tree JSON from build")
	case "expand-confirm":
		fs.StringVar(&draft, "children", "", "drafted children JSON from expand")
	}
	usage := map[string]string{
		"build":          " <prd-id>",
		"expand":         " <node-id>",
		"build-confirm":  " <prd-id> -tree '<json>'",
		"expand-confirm": " <node-id> -children '<json>'",
	}
	if err := fs.Parse(args); err != nil || len(fs.Args()) != 1 {
		fmt.Fprintf(os.Stderr, "usage: insideout %s%s\n", cmd, usage[cmd])
		return true, 2
	}
	if (cmd == "build-confirm" || cmd == "expand-confirm") && !json.Valid([]byte(draft)) {
		fmt.Fprintln(os.Stderr, "confirm needs the drafted JSON from the matching draft command")
		return true, 2
	}
	c := apiclient.New(api)
	c.SetToken(os.Getenv("INSIDEOUT_TOKEN"))
	switch cmd {
	case "build":
		return printJSON(c.BuildFromPrd(fs.Args()[0]))
	case "build-confirm":
		return printJSON(c.ConfirmBuildFromPrd(fs.Args()[0], json.RawMessage(draft), count))
	case "expand-confirm":
		return printJSON(c.ConfirmExpandNode(fs.Args()[0], json.RawMessage(draft)))
	default:
		return printJSON(c.ExpandNode(fs.Args()[0]))
	}
}

// runGithubCommand handles `insideout repo set <project-id> <repo-url>`
// and `insideout sync <project-id>`.
func runGithubCommand(cmd string, args []string, api string) (bool, int) {
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	applyAPIFlag(fs, &api)
	if err := fs.Parse(args); err != nil {
		return true, 2
	}
	c := apiclient.New(api)
	c.SetToken(os.Getenv("INSIDEOUT_TOKEN"))
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

func strPtrIfSet(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func applyAPIFlag(fs *flag.FlagSet, api *string) {
	fs.StringVar(api, "api", envOr("INSIDEOUT_API", defaultAPIBase), "API base URL")
}
