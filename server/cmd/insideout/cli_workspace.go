package main

// Bootstrapping a workspace on the CLI: `insideout workspaces create`,
// `insideout join <workspace-code>` and `insideout projects create`
// (docs/plans/2026-08-25-agent-crud-tool-surface.md). Reading and editing a
// workspace or project runs in cli_project.go.

import (
	"fmt"
	"os"
)

// runWorkspacesCreate is `insideout workspaces create --title T [--desc D]`.
func runWorkspacesCreate(args []string) (bool, int) {
	f, ok := parseWriteFlags("workspaces create", args)
	if !ok {
		return true, 2
	}
	if *f.title == "" {
		fmt.Fprintln(os.Stderr, "usage: insideout workspaces create --title T [--desc D]")
		return true, 2
	}
	return printJSON(clientFromFlag(*f.api).CreateWorkspace(*f.title, *f.desc))
}

// runJoin is `insideout join <workspace-code>`.
func runJoin(args []string) (bool, int) {
	f, ok := parseWriteFlags("join", args)
	if !ok || !f.need(1, "insideout join <workspace-code>") {
		return true, 2
	}
	return printJSON(clientFromFlag(*f.api).JoinWorkspace(f.positional[0]))
}

// runProjectsCreate is `insideout projects create --title T [--desc D] <workspace-id>`.
func runProjectsCreate(args []string) (bool, int) {
	f, ok := parseWriteFlags("projects create", args)
	if !ok || !f.need(1, "insideout projects create --title T [--desc D] <workspace-id>") || *f.title == "" {
		return true, 2
	}
	return printJSON(clientFromFlag(*f.api).CreateProject(f.positional[0], *f.title, *f.desc))
}
