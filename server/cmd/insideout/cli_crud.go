package main

// CRUD-completion verbs (docs/plans/2026-08-25-agent-crud-tool-surface.md):
// the idea-inbox lifecycle, PRD section authoring, evidence reading, and
// bootstrapping — same raw-JSON output convention as the rest of the CLI.

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

// stringList is a repeatable string flag (--section k=v --section k2=v2).
type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ", ") }
func (s *stringList) Set(v string) error {
	*s = append(*s, v)
	return nil
}

func runCrudCommand(cmd string, args []string, api string) (bool, int) {
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	title := fs.String("title", "", "title")
	content := fs.String("content", "", "content (idea body / section text)")
	desc := fs.String("desc", "", "description")
	status := fs.String("status", "", "PRD status transition (applies after section edits)")
	var sections stringList
	fs.Var(&sections, "section", "section key=value (repeatable)")
	if err := fs.Parse(args); err != nil {
		return true, 2
	}
	positional := fs.Args()
	c := apiclient.New(api)
	c.SetToken(os.Getenv("INSIDEOUT_TOKEN"))

	need := func(n int, usage string) bool {
		if len(positional) != n {
			fmt.Fprintln(os.Stderr, "usage: "+usage)
			return false
		}
		return true
	}

	switch cmd {
	case "ideas":
		if !need(1, "insideout ideas <workspace-id>") {
			return true, 2
		}
		return printJSON(c.Ideas(positional[0]))
	case "idea update":
		if !need(1, "insideout idea update --title T [--content C] <id>") {
			return true, 2
		}
		return printJSON(c.UpdateIdea(positional[0], *title, *content))
	case "idea delete":
		if !need(1, "insideout idea delete <id>") {
			return true, 2
		}
		if err := c.DeleteIdea(positional[0]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return true, 1
		}
		fmt.Println("dropped (soft — idea_list keeps it with status \"dropped\")")
		return true, 0
	case "prd update":
		if !need(1, "insideout prd update [--title T] [--section key=value]… [--status S] <id>") {
			return true, 2
		}
		var sec map[string]string
		if len(sections) > 0 {
			sec = map[string]string{}
			for _, s := range sections {
				k, v, ok := strings.Cut(s, "=")
				if !ok {
					fmt.Fprintf(os.Stderr, "prd update: --section wants key=value, got %q\n", s)
					return true, 2
				}
				sec[k] = v
			}
		}
		var t *string
		if *title != "" {
			t = title
		}
		out, err := c.UpdatePrd(positional[0], t, sec)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return true, 1
		}
		if *status != "" {
			return printJSON(c.SetPrdStatus(positional[0], *status))
		}
		return printJSON(out, nil)
	case "evidence":
		if !need(1, "insideout evidence <node-id>") {
			return true, 2
		}
		return printJSON(c.NodeEvidence(positional[0]))
	case "workspaces create":
		if *title == "" {
			fmt.Fprintln(os.Stderr, "usage: insideout workspaces create --title T [--desc D]")
			return true, 2
		}
		return printJSON(c.CreateWorkspace(*title, *desc))
	case "join":
		if !need(1, "insideout join <workspace-code>") {
			return true, 2
		}
		return printJSON(c.JoinWorkspace(positional[0]))
	case "projects create":
		if !need(1, "insideout projects create --title T [--desc D] <workspace-id>") || *title == "" {
			return true, 2
		}
		return printJSON(c.CreateProject(positional[0], *title, *desc))
	}
	return false, 0
}
