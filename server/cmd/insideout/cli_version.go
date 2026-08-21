package main

// `insideout commit` and `insideout versions` — the human Commit and
// the version list (docs/plans/2026-08-21-version-commit.md).

import (
	"flag"
	"fmt"
	"os"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

func runVersionCommands(cmd string, args []string, api string) (bool, int) {
	c := apiclient.New(api)
	c.SetToken(os.Getenv("INSIDEOUT_TOKEN"))
	if cmd == "versions" {
		fs := flag.NewFlagSet("versions", flag.ContinueOnError)
		applyAPIFlag(fs, &api)
		if err := fs.Parse(args); err != nil || len(fs.Args()) != 1 {
			fmt.Fprintln(os.Stderr, "usage: insideout versions <prd-id>")
			return true, 2
		}
		return printJSON(c.PrdVersions(fs.Args()[0]))
	}
	// commit
	fs := flag.NewFlagSet("commit", flag.ContinueOnError)
	applyAPIFlag(fs, &api)
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
	return printJSON(c.CommitPrd(fs.Args()[0], *name, *audience, *summary, unresolved, *note))
}

// multiFlag collects repeated string flags.
type multiFlag []string

func (m *multiFlag) String() string     { return "" }
func (m *multiFlag) Set(v string) error { *m = append(*m, v); return nil }
