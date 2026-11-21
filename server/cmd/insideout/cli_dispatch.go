package main

// The client-verb dispatcher: it maps a command line to the verb function
// that runs it. cliVerbs (cli_verbs.go) is the inventory it must match.

import (
	"flag"
	"fmt"
	"os"
)

// runClientCommand reports whether the argument was a client command
// (already executed; the caller should exit with the returned code).
func runClientCommand(args []string) (handled bool, exitCode int) {
	if len(args) == 0 {
		return false, 0
	}
	cmd, rest := args[0], args[1:]

	// Two-word verbs whose first word is also a verb of its own.
	if len(rest) > 0 {
		switch cmd + " " + rest[0] {
		case "prd update":
			return runPrdUpdate(rest[1:])
		case "workspaces create":
			return runWorkspacesCreate(rest[1:])
		case "projects create":
			return runProjectsCreate(rest[1:])
		case "project update", "update add", "workspace update":
			return runProjectCommand(cmd+" "+rest[0], rest[1:])
		case "project delete", "workspace delete", "member role", "member remove", "update edit", "update delete", "profile update":
			return runPeopleCommand(cmd+" "+rest[0], rest[1:])
		case "account delete", "evidence add", "roadmap validate":
			return runHumanCommand(cmd+" "+rest[0], rest[1:])
		case "proposal decide":
			return runProposalDecide(rest[1:])
		}
	}
	switch cmd {
	case "project", "updates", "update", "workspace", "prds":
		return runProjectCommand(cmd, rest)
	case "account":
		return runHumanCommand(cmd, rest)
	case "member", "profile":
		return runPeopleCommand(cmd, rest)
	case "proposal": // anything but `proposal decide` prints its usage
		return runProposalDecide(nil)
	case "export":
		return runExportCommand(rest)
	case "setup":
		return runSetupCommand(rest)
	case "health":
		// Container probe: no API client, no flags — see cli_health.go.
		return true, runHealthCommand()
	case "ideas":
		return runIdeas(rest)
	case "evidence":
		return runEvidence(rest)
	case "join":
		return runJoin(rest)
	case "roadmap":
		return runRoadmapCommand(rest)
	case "build", "expand", "build-confirm", "expand-confirm":
		return runDraftCommand(cmd, rest)
	case "repo", "sync":
		return runGithubCommand(cmd, rest)
	case "guide":
		return runGuideCommand(rest)
	case "commit", "versions", "revisions", "snapshot":
		return runPrdVersionCommand(cmd, rest)
	case "view":
		return runPrdView(rest)
	case "readiness":
		return runPrdReadiness(rest)
	case "agent-context", "checkpoint", "propose":
		return runAgentCommand(cmd, rest)
	case "presence":
		return runRoadmapCommand(append([]string{"presence"}, rest...))
	case "members":
		return runMembersCommand(rest)
	case "idea":
		return runIdeaCommand(rest)
	case "login", "register", "forgot-password", "reset-password", "logout":
		return runAuthCommand(cmd, rest)
	case "invite", "invites", "signup-mode":
		return runAdminCommand(cmd, rest)
	case "whoami", "workspaces", "projects", "prd":
		return runReadCommand(cmd, rest)
	}
	return false, 0
}

// runReadCommand runs whoami, workspaces, projects or prd: one GET, its JSON
// printed as the API sends it.
func runReadCommand(cmd string, rest []string) (bool, int) {
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	api := apiFlag(fs)
	if err := fs.Parse(rest); err != nil {
		return true, 2
	}
	c := clientFromFlag(*api)

	switch cmd {
	case "whoami":
		return printJSON(c.Whoami())
	case "workspaces":
		return printJSON(c.Workspaces())
	case "projects":
		if len(fs.Args()) != 1 {
			fmt.Fprintln(os.Stderr, "usage: insideout projects <workspace-id>")
			return true, 2
		}
		return printJSON(c.Projects(fs.Args()[0]))
	}
	if len(fs.Args()) != 1 {
		fmt.Fprintln(os.Stderr, "usage: insideout prd <prd-id>")
		return true, 2
	}
	return printJSON(c.Prd(fs.Args()[0]))
}
