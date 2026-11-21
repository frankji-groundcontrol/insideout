package main

// The deletes, member roles and the profile (rv5-PARA-06). The server keeps
// them for a human's session, so MCP has no tool for them; deleting a project
// or a workspace takes everything under it and asks for --yes. A profile edit reads the user first, so a flag
// left out keeps its value; the word clear clears the bio or the avatar
// (docs/usage/cli-mcp-and-guidance.md, "Absent, empty and clear").

import (
	"flag"
	"fmt"
	"os"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

var peopleUsage = map[string]string{
	"project delete":   "insideout project delete --yes <project-id>  (deletes its roadmap, updates and evidence too)",
	"workspace delete": "insideout workspace delete --yes <workspace-id>  (deletes its projects, ideas and PRDs too)",
	"member role":      "insideout member role --role admin|member <workspace-id> <user-id>",
	"member remove":    "insideout member remove <workspace-id> <user-id>",
	"update edit":      "insideout update edit --content C <update-id>",
	"update delete":    "insideout update delete <update-id>",
	"profile update":   "insideout profile update [--username U] [--bio B|clear] [--avatar URL|clear]",
}

// runPeopleCommand runs one of the peopleUsage verbs; a bare `member` or
// `profile` prints its usage.
func runPeopleCommand(verb string, args []string) (bool, int) {
	usage, ok := peopleUsage[verb]
	if !ok {
		usage = peopleUsage["member role"] + "\n       " + peopleUsage["member remove"]
		if verb == "profile" {
			usage = peopleUsage["profile update"]
		}
	}
	fs := flag.NewFlagSet(verb, flag.ContinueOnError)
	api := apiFlag(fs)
	var yes *bool
	var role, content, username, bio, avatar *string
	wantIDs := 1
	switch verb {
	case "project delete", "workspace delete":
		yes = fs.Bool("yes", false, "confirm the delete")
	case "member role":
		role = fs.String("role", "", "admin|member")
		wantIDs = 2
	case "member remove":
		wantIDs = 2
	case "update edit":
		content = fs.String("content", "", "the new text")
	case "profile update":
		username = fs.String("username", "", "new display name")
		bio = fs.String("bio", "", "new bio, or clear")
		avatar = fs.String("avatar", "", "avatar image URL, or clear")
		wantIDs = 0
	}
	bad := !ok || fs.Parse(args) != nil || len(fs.Args()) != wantIDs ||
		(yes != nil && !*yes) || (role != nil && *role == "") || (content != nil && *content == "")
	var patch apiclient.MePatch
	if username != nil {
		patch = apiclient.MePatch{Username: apiclient.NonEmpty(*username), Bio: apiclient.PatchText(*bio),
			AvatarURL: apiclient.PatchText(*avatar)}
		bad = bad || (patch.Username == nil && patch.Bio == nil && patch.AvatarURL == nil)
	}
	if bad {
		fmt.Fprintln(os.Stderr, "usage: "+usage)
		return true, 2
	}
	c, ids := clientFromFlag(*api), fs.Args()
	switch verb {
	case "member role":
		return printJSON(c.UpdateMember(ids[0], ids[1], *role))
	case "update edit":
		return printJSON(c.EditProjectUpdate(ids[0], *content))
	case "profile update":
		return printJSON(patch.Apply(c))
	}
	var err error
	done := "deleted"
	switch verb {
	case "project delete":
		err = c.DeleteProject(ids[0])
	case "workspace delete":
		err = c.DeleteWorkspace(ids[0])
	case "member remove":
		err, done = c.RemoveMember(ids[0], ids[1]), "removed"
	default: // update delete
		err = c.DeleteProjectUpdate(ids[0])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return true, 1
	}
	fmt.Println(done)
	return true, 0
}
