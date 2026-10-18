package main

// `insideout help`, `insideout version` and the unknown-verb answer. They
// run before config.Load(): the first thing a new user types
// (insideout --help) must not meet "DATABASE_URL is required" (rv4-USER-02,
// FULL-03, PM-04). The listing comes from the cliVerbs table in cli.go.

import (
	"fmt"
	"io"
)

// runFrontDoor answers help, version and unknown verbs. handled=false means
// "go on": no verb is the API server (the container runs `/insideout` bare),
// and the server verbs (migrate, seed, promote-admin) are run by main() once
// it has loaded config.
func runFrontDoor(args []string, stdout, stderr io.Writer) (handled bool, exitCode int) {
	if len(args) == 0 {
		return false, 0
	}
	switch args[0] {
	case "help", "-h", "--help":
		printHelp(stdout)
		return true, 0
	case "version", "--version":
		fmt.Fprintln(stdout, "insideout", version)
		return true, 0
	}
	for _, v := range cliVerbs {
		if v.group == groupServer && v.name == args[0] {
			return false, 0
		}
	}
	fmt.Fprintf(stderr, "insideout: unknown command %q\nrun `insideout help` for the list of commands\n", args[0])
	return true, 2
}

func printHelp(w io.Writer) {
	fmt.Fprint(w, "insideout - track projects, capture ideas, refine them into PRDs\n\n"+
		"Usage:\n  insideout                run the API server\n  insideout <command> [args]\n")
	group := ""
	for _, v := range cliVerbs {
		if v.group != group {
			group = v.group
			fmt.Fprintf(w, "\n%s:\n", group)
		}
		fmt.Fprintf(w, "  %-16s %s\n", v.name, v.summary)
	}
	fmt.Fprint(w, "\nClient commands talk to INSIDEOUT_API (default: the hosted API) with INSIDEOUT_TOKEN;\n"+
		"a command given the wrong arguments prints its own usage.\n")
}
