package main

// Roadmap CLI verbs — the roadmap projection of the /api/v1 contract
// (docs/plans/2026-08-21-roadmap-parity-and-github.md). Agent planning
// (build, expand), branch CRUD (roadmap add/update/move/delete), and
// GitHub binding live here; reads print the API's raw JSON.

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

// runRoadmapCommand handles `insideout roadmap <list|add|update|move|delete> …`.
// Returns (handled, exitCode).
func runRoadmapCommand(args []string, api string) (bool, int) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: insideout roadmap <list|add|update|move|delete> …")
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

// runAgentCommand handles `insideout build <prd-id>` and
// `insideout expand <node-id>` — the agent-planning verbs.
func runAgentCommand(cmd string, args []string, api string) (bool, int) {
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	applyAPIFlag(fs, &api)
	count := 0
	if cmd == "build" {
		fs.IntVar(&count, "expected-count", 0, "optional hint for the number of root nodes")
	}
	if err := fs.Parse(args); err != nil || len(fs.Args()) != 1 {
		fmt.Fprintf(os.Stderr, "usage: insideout %s <id>%s\n", cmd, map[string]string{"build": "  (prd id)", "expand": "  (roadmap node id)"}[cmd])
		return true, 2
	}
	c := apiclient.New(api)
	c.SetToken(os.Getenv("INSIDEOUT_TOKEN"))
	if cmd == "build" {
		return printJSON(c.BuildFromPrd(fs.Args()[0], count))
	}
	return printJSON(c.ExpandNode(fs.Args()[0]))
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
