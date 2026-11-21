package main

// Projects beyond list and create, their updates feed, one workspace and a
// workspace's PRDs (rv5-PARB-04, PARA-06, PARB-07, PARB-03). An edit reads the
// row first, so a flag left out keeps its value; the word clear clears one
// (docs/usage/cli-mcp-and-guidance.md, "Absent, empty and clear").

import (
	"flag"
	"fmt"
	"os"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

var projectUsage = map[string]string{
	"project": "insideout project <project-id>",
	"project update": "insideout project update [--title T] [--description D|clear] " +
		"[--status planning|active|paused|done|archived] [--owner user-id|me|clear] <project-id>",
	"updates":          "insideout updates [--before cursor] [--limit n] <project-id>",
	"update add":       "insideout update add --kind progress|blocker|note --content C <project-id>",
	"workspace":        "insideout workspace <workspace-id>",
	"workspace update": "insideout workspace update [--title T] [--description D|clear] [--cover URL|clear] <workspace-id>",
	"prds":             "insideout prds <workspace-id>",
}

// runProjectCommand runs one of the projectUsage verbs.
func runProjectCommand(verb string, args []string) (bool, int) {
	usage, ok := projectUsage[verb]
	if !ok { // `update` with no sub-verb, or an unknown one
		usage = projectUsage["update add"] + "\n       " + peopleUsage["update edit"] + "\n       " + peopleUsage["update delete"]
	}
	fs := flag.NewFlagSet(verb, flag.ContinueOnError)
	api := apiFlag(fs)
	var title, desc, status, owner, cover, before, kind, content *string
	var limit *int
	switch verb {
	case "project update", "workspace update":
		title = fs.String("title", "", "new title")
		desc = fs.String("description", "", "new description, or clear")
		if verb == "project update" {
			status = fs.String("status", "", "planning|active|paused|done|archived")
			owner = fs.String("owner", "", "owner: a member's user id, me, or clear")
		} else {
			cover = fs.String("cover", "", "cover image URL, or clear")
		}
	case "updates":
		before = fs.String("before", "", "the nextCursor of the page before")
		limit = fs.Int("limit", 0, "updates per page, 1 to 100 (default 50)")
	case "update add":
		kind = fs.String("kind", "", "progress|blocker|note")
		content = fs.String("content", "", "the update text")
	}
	if !ok || fs.Parse(args) != nil || len(fs.Args()) != 1 || (kind != nil && (*kind == "" || *content == "")) {
		fmt.Fprintln(os.Stderr, "usage: "+usage)
		return true, 2
	}
	c, id := clientFromFlag(*api), fs.Args()[0]
	switch verb {
	case "project":
		return printJSON(c.Project(id))
	case "updates":
		return printJSON(c.ProjectUpdates(id, *before, *limit))
	case "update add":
		return printJSON(c.AddProjectUpdate(id, apiclient.ProjectUpdate{Kind: *kind, Content: *content}))
	case "workspace":
		return printJSON(c.Workspace(id))
	case "prds":
		return printJSON(c.Prds(id))
	case "project update":
		patch := apiclient.ProjectPatch{Title: apiclient.NonEmpty(*title), Description: apiclient.PatchText(*desc),
			Status: apiclient.NonEmpty(*status), OwnerID: apiclient.PatchText(*owner)}
		if patch == (apiclient.ProjectPatch{}) {
			fmt.Fprintln(os.Stderr, "usage: "+usage)
			return true, 2
		}
		if *owner == "me" {
			me, err := signedInUserID(c)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return true, 1
			}
			patch.OwnerID = &me
		}
		return printJSON(patch.Apply(c, id))
	}
	// workspace update
	patch := apiclient.WorkspacePatch{Title: apiclient.NonEmpty(*title), Description: apiclient.PatchText(*desc),
		CoverURL: apiclient.PatchText(*cover)}
	if patch == (apiclient.WorkspacePatch{}) {
		fmt.Fprintln(os.Stderr, "usage: "+usage)
		return true, 2
	}
	return printJSON(patch.Apply(c, id))
}
