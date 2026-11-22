package main

// `roadmap steps`: one level of the roadmap in dependency order, the Canvas's
// view (docs/architecture/roadmap-tree.md, "Steps"). The flag precedes the id.

import (
	"flag"
	"fmt"
	"os"
)

// runRoadmapSteps runs `roadmap steps`; handled is false for any other sub.
func runRoadmapSteps(sub string, args []string) (bool, int) {
	if sub != "steps" {
		return false, 0
	}
	fs := flag.NewFlagSet("roadmap steps", flag.ContinueOnError)
	api := apiFlag(fs)
	root := fs.String("root", "", "open this node (default: an agent token's root, else the project's one root)")
	if err := fs.Parse(args); err != nil || len(fs.Args()) != 1 {
		fmt.Fprintln(os.Stderr, "usage: insideout roadmap steps [--root node-id] <project-id>")
		return true, 2
	}
	return printJSON(clientFromFlag(*api).RoadmapSteps(fs.Args()[0], *root))
}
