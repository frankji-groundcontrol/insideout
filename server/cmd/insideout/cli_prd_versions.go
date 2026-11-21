package main

// A PRD's history on the CLI: `insideout commit` (the human Commit of a
// named, audience-specific version), `versions`, `revisions` and `snapshot`
// (docs/plans/2026-08-21-version-commit.md).

import (
	"flag"
	"fmt"
	"os"
)

// runPrdVersionCommand runs commit, versions, revisions or snapshot.
func runPrdVersionCommand(cmd string, args []string) (bool, int) {
	switch cmd {
	case "commit":
		return runPrdCommit(args)
	case "snapshot":
		return runPrdSnapshot(args)
	}
	// versions and revisions: one PRD id, the list printed as the API sends it.
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	api := apiFlag(fs)
	if err := fs.Parse(args); err != nil || len(fs.Args()) != 1 {
		fmt.Fprintf(os.Stderr, "usage: insideout %s <prd-id>\n", cmd)
		return true, 2
	}
	c := clientFromFlag(*api)
	if cmd == "revisions" {
		return printJSON(c.PrdRevisions(fs.Args()[0]))
	}
	return printJSON(c.PrdVersions(fs.Args()[0]))
}

// runPrdSnapshot is `insideout snapshot [--note N] <prd-id>`.
func runPrdSnapshot(args []string) (bool, int) {
	fs := flag.NewFlagSet("snapshot", flag.ContinueOnError)
	api := apiFlag(fs)
	note := fs.String("note", "", "optional note")
	if err := fs.Parse(args); err != nil || len(fs.Args()) != 1 {
		fmt.Fprintln(os.Stderr, "usage: insideout snapshot [--note N] <prd-id>")
		return true, 2
	}
	return printJSON(clientFromFlag(*api).SnapshotPrd(fs.Args()[0], *note))
}

// runPrdCommit is `insideout commit --name N --audience A … <prd-id>`.
func runPrdCommit(args []string) (bool, int) {
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
