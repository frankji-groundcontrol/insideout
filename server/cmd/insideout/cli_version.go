package main

// `insideout commit` and `insideout versions` — the human Commit and
// the version list (docs/plans/2026-08-21-version-commit.md).

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

func runVersionCommands(cmd string, args []string) (bool, int) {
	if cmd == "revisions" {
		fs := flag.NewFlagSet("revisions", flag.ContinueOnError)
		api := apiFlag(fs)
		if err := fs.Parse(args); err != nil || len(fs.Args()) != 1 {
			fmt.Fprintln(os.Stderr, "usage: insideout revisions <prd-id>")
			return true, 2
		}
		return printJSON(clientFromFlag(*api).PrdRevisions(fs.Args()[0]))
	}
	if cmd == "snapshot" {
		fs := flag.NewFlagSet("snapshot", flag.ContinueOnError)
		api := apiFlag(fs)
		note := fs.String("note", "", "optional note")
		if err := fs.Parse(args); err != nil || len(fs.Args()) != 1 {
			fmt.Fprintln(os.Stderr, "usage: insideout snapshot [--note N] <prd-id>")
			return true, 2
		}
		return printJSON(clientFromFlag(*api).SnapshotPrd(fs.Args()[0], *note))
	}
	if cmd == "agent-context" {
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
	if cmd == "checkpoint" {
		fs := flag.NewFlagSet("checkpoint", flag.ContinueOnError)
		api := apiFlag(fs)
		node := fs.String("node", "", "roadmap node id (optional)")
		if err := fs.Parse(args); err != nil || len(fs.Args()) != 2 {
			fmt.Fprintln(os.Stderr, "usage: insideout checkpoint [--node id] <project-id> <summary>")
			return true, 2
		}
		return printJSON(clientFromFlag(*api).AgentCheckpoint(fs.Args()[0], *node, fs.Args()[1]))
	}
	if cmd == "propose" {
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
	if cmd == "view" {
		fs := flag.NewFlagSet("view", flag.ContinueOnError)
		api := apiFlag(fs)
		audience := fs.String("audience", "decision", "decision|management|delivery|validation")
		out := fs.String("export", "", "write the audience markdown view to this file")
		if err := fs.Parse(args); err != nil || len(fs.Args()) != 1 {
			fmt.Fprintln(os.Stderr, "usage: insideout view [--audience A] [--export FILE] <prd-id>")
			return true, 2
		}
		c := clientFromFlag(*api)
		if *out != "" {
			md, err := c.PrdExportAudience(fs.Args()[0], *audience)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return true, 1
			}
			if err := os.WriteFile(*out, md, 0o644); err != nil {
				fmt.Fprintln(os.Stderr, "write:", err)
				return true, 1
			}
			fmt.Fprintf(os.Stderr, "wrote %s"+"\n", *out)
			return true, 0
		}
		return printJSON(c.PrdView(fs.Args()[0], *audience))
	}
	if cmd == "readiness" {
		fs := flag.NewFlagSet("readiness", flag.ContinueOnError)
		api := apiFlag(fs)
		if err := fs.Parse(args); err != nil || len(fs.Args()) != 1 {
			fmt.Fprintln(os.Stderr, "usage: insideout readiness <prd-id>")
			return true, 2
		}
		return printJSON(clientFromFlag(*api).PrdReadiness(fs.Args()[0]))
	}
	if cmd == "versions" {
		fs := flag.NewFlagSet("versions", flag.ContinueOnError)
		api := apiFlag(fs)
		if err := fs.Parse(args); err != nil || len(fs.Args()) != 1 {
			fmt.Fprintln(os.Stderr, "usage: insideout versions <prd-id>")
			return true, 2
		}
		return printJSON(clientFromFlag(*api).PrdVersions(fs.Args()[0]))
	}
	// commit
	fs := flag.NewFlagSet("commit", flag.ContinueOnError)
	api := apiFlag(fs)
	name := fs.String("name", "", "version name (required)")
	audience := fs.String("audience", "", "decision|management|delivery|validation (required)")
	summary := fs.String("summary", "", "change summary")
	note := fs.String("note", "", "decision log note")
	var unresolved multiFlag
	fs.Var(&unresolved, "unresolved", "carried open item (repeatable)")
	if err := fs.Parse(args); err != nil || len(fs.Args()) != 1 || *name == "" || *audience == "" {
		fmt.Fprintln(os.Stderr, "usage: insideout commit --name N --audience A [--summary S] [--unresolved U]... [--note D] <prd-id>")
		return true, 2
	}
	return printJSON(clientFromFlag(*api).CommitPrd(fs.Args()[0], *name, *audience, *summary, unresolved, *note))
}

// multiFlag collects repeated string flags.
type multiFlag []string

func (m *multiFlag) String() string     { return "" }
func (m *multiFlag) Set(v string) error { *m = append(*m, v); return nil }
