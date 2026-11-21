package main

// GitHub binding on the CLI (docs/plans/2026-08-21-roadmap-parity-and-github.md):
// link a repository to a project and pull its activity into the roadmap.

import (
	"flag"
	"fmt"
	"os"
)

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
