package main

// The task tree on the CLI (docs/architecture/roadmap-tree.md): a project's
// tree or one subtree, one node with its breadcrumb, the caller's own nodes,
// node edits with owner and acceptance, delegation and its history, and the
// workspace members a node can be handed to. Flags precede the positional id.

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

// treeUsage is each tree verb's usage line.
var treeUsage = map[string]string{
	"list":        "insideout roadmap list [--root node-id] <project-id>",
	"progress":    "insideout roadmap progress [--root node-id] <project-id>",
	"show":        "insideout roadmap show <node-id>",
	"mine":        "insideout roadmap mine <workspace-id>",
	"assignments": "insideout roadmap assignments <node-id>",
	"delegate":    "insideout roadmap delegate --to <user-id> [--note N] <node-id>",
	"handback":    "insideout roadmap handback [--note N] <node-id>",
	"update": "insideout roadmap update [--title T] [--description D] [--status S] [--deadline RFC3339|clear] " +
		"[--acceptance A] [--owner user-id|me|none] [--completion-reason SENTENCE] <node-id>",
}

// runRoadmapTreeCommand runs `roadmap <sub>` when sub is a tree verb; handled
// is false for any other sub.
func runRoadmapTreeCommand(sub string, args []string) (handled bool, exitCode int) {
	usage, ok := treeUsage[sub]
	if !ok {
		return false, 0
	}
	fs := flag.NewFlagSet("roadmap "+sub, flag.ContinueOnError)
	api := apiFlag(fs)
	var root, to, note *string
	switch sub {
	case "list", "progress":
		root = fs.String("root", "", "limit to this node and the nodes below it")
	case "delegate":
		to = fs.String("to", "", "user id of the workspace member who takes the node (see `insideout members`)")
		note = fs.String("note", "", "a note for the record (at most 500 characters)")
	case "handback":
		note = fs.String("note", "", "a note for the record (at most 500 characters)")
	case "update":
		return runRoadmapUpdate(fs, api, args)
	}
	if err := fs.Parse(args); err != nil || len(fs.Args()) != 1 || (to != nil && *to == "") {
		fmt.Fprintln(os.Stderr, "usage: "+usage)
		return true, 2
	}
	c, id := clientFromFlag(*api), fs.Args()[0]
	switch sub {
	case "list":
		return printJSON(c.RoadmapList(id, *root))
	case "progress":
		return printJSON(c.RoadmapProgress(id, *root))
	case "show":
		return printJSON(c.RoadmapNode(id))
	case "mine":
		return printJSON(c.RoadmapMine(id))
	case "assignments":
		return printJSON(c.RoadmapAssignments(id))
	case "delegate":
		return printJSON(c.RoadmapDelegate(id, *to, *note))
	default: // handback
		return printJSON(c.RoadmapHandBack(id, *note))
	}
}

// runRoadmapUpdate is `roadmap update`. --acceptance "" clears the acceptance;
// --owner takes a user id, me (the signed-in user) or none (no owner).
func runRoadmapUpdate(fs *flag.FlagSet, api *string, args []string) (bool, int) {
	title := fs.String("title", "", "new title")
	description := fs.String("description", "", "new description")
	status := fs.String("status", "", "locked|pending|in_progress|done")
	deadline := fs.String("deadline", "", "RFC3339 deadline, or clear")
	acceptance := fs.String("acceptance", "", `what done means ("" clears it)`)
	owner := fs.String("owner", "", "owner: a member's user id, me, or none")
	reason := fs.String("completion-reason", "", "one sentence when marking done with no evidence")
	err := fs.Parse(args)
	acceptanceSet := false
	fs.Visit(func(f *flag.Flag) { acceptanceSet = acceptanceSet || f.Name == "acceptance" })
	if err != nil || len(fs.Args()) != 1 || (*title == "" && *description == "" && *status == "" &&
		*deadline == "" && !acceptanceSet && *owner == "" && *reason == "") {
		fmt.Fprintln(os.Stderr, "usage: "+treeUsage["update"])
		return true, 2
	}
	patch := apiclient.RoadmapPatch{Title: strPtrIfSet(*title), Description: strPtrIfSet(*description),
		Status: strPtrIfSet(*status), CompletionReason: strPtrIfSet(*reason)}
	if acceptanceSet {
		patch.Acceptance = acceptance
	}
	switch *deadline {
	case "":
	case "clear":
		patch.ClearDeadline = true
	default:
		if _, err := time.Parse(time.RFC3339, *deadline); err != nil {
			fmt.Fprintln(os.Stderr, "deadline must be RFC3339 or clear")
			return true, 2
		}
		patch.Deadline = *deadline
	}
	c := clientFromFlag(*api)
	switch *owner {
	case "none":
		patch.ClearOwner = true
	case "me":
		id, err := signedInUserID(c)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return true, 1
		}
		patch.OwnerID = id
	default:
		patch.OwnerID = *owner
	}
	return printJSON(c.RoadmapUpdate(fs.Args()[0], patch))
}

// signedInUserID is the id GET /me answers for this client's login.
func signedInUserID(c *apiclient.Client) (string, error) {
	raw, err := c.Whoami()
	if err != nil {
		return "", err
	}
	var me struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(raw, &me) != nil || me.ID == "" {
		return "", fmt.Errorf("--owner me: GET /me did not answer a user id")
	}
	return me.ID, nil
}

// runMembersCommand is `insideout members <workspace-id>`: the members of a
// workspace with the user ids `roadmap delegate --to` and `--owner` take.
func runMembersCommand(args []string) (bool, int) {
	fs := flag.NewFlagSet("members", flag.ContinueOnError)
	api := apiFlag(fs)
	if err := fs.Parse(args); err != nil || len(fs.Args()) != 1 {
		fmt.Fprintln(os.Stderr, "usage: insideout members <workspace-id>")
		return true, 2
	}
	return printJSON(clientFromFlag(*api).Members(fs.Args()[0]))
}
