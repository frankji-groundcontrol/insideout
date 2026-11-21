package main

// cliVerb is one row of the verb inventory; cliVerbs is that inventory, in
// `insideout help` print order. help.go prints it and answers "unknown
// command" against it, and TestVerbTableMatchesDispatch fails when it and
// the dispatch switches (runClientCommand in cli_dispatch.go, main, runFrontDoor) drift
// apart — hand-written verb lists went stale (review-3 ARCH-04).
type cliVerb struct{ group, name, summary string }

const (
	groupServer  = "Server operator commands (need DATABASE_URL and INSIDEOUT_JWT_SECRET)"
	groupSetup   = "Setup and diagnostics"
	groupAccount = "Account and sign-in"
	groupWork    = "Workspaces, ideas and projects"
	groupPRD     = "PRDs and versions"
	groupRoadmap = "Roadmap and agent"
	groupAdmin   = "Admin (needs an admin token)"
)

var cliVerbs = []cliVerb{
	{groupServer, "migrate", "apply pending SQL migrations and exit"},
	{groupServer, "seed", "create demo data for local development"},
	{groupServer, "promote-admin", "make an account admin, creating it (password on stdin) if absent"},

	{groupSetup, "setup", "numbered setup: managed service, a service you host, or a server on this machine"},
	{groupSetup, "health", "probe /healthz over loopback (the container HEALTHCHECK)"},
	{groupSetup, "help", "print this list (also -h, --help)"},
	{groupSetup, "version", "print the build version (also --version)"},

	{groupAccount, "register", "create an account (password on stdin; --invite when signup is invitation-only)"},
	{groupAccount, "login", "sign in, save the login and print the token (--code or --magic for emailed sign-in)"},
	{groupAccount, "logout", "revoke the saved login on the server and delete it"},
	{groupAccount, "forgot-password", "email a password-reset link"},
	{groupAccount, "reset-password", "finish a reset: paste the link, then the new password"},
	{groupAccount, "whoami", "show the signed-in user"},
	{groupAccount, "account", "account delete: delete your account (asks for the current password)"},
	{groupAccount, "profile", "profile update: edit your username, bio, keywords or avatar"},

	{groupWork, "workspaces", "list workspaces (workspaces create --title T)"},
	{groupWork, "workspace", "show a workspace (workspace update edits it, workspace delete --yes removes it)"},
	{groupWork, "join", "join a workspace with its workspace code"},
	{groupWork, "projects", "list a workspace's projects (projects create --title T <workspace-id>)"},
	{groupWork, "project", "show a project (project update edits it, project delete --yes removes it)"},
	{groupWork, "updates", "page through a project's updates (--before the nextCursor)"},
	{groupWork, "update", "post, edit or delete a project update (update add, update edit, update delete)"},
	{groupWork, "ideas", "list a workspace's ideas"},
	{groupWork, "idea", "create, update, delete or convert an idea"},
	{groupWork, "evidence", "show the evidence behind a roadmap node (evidence add records an observation)"},
	{groupWork, "repo", "link a GitHub repository to a project (repo set <project-id> <url>)"},
	{groupWork, "sync", "pull a project's GitHub activity into its roadmap"},
	{groupWork, "guide", "print or write a project's insideout.yaml guide"},

	{groupPRD, "prds", "list a workspace's PRDs, newest first"},
	{groupPRD, "prd", "show a PRD (prd update edits its title, sections or status)"},
	{groupPRD, "view", "render a PRD for an audience (--export FILE writes markdown)"},
	{groupPRD, "export", "export the whole PRD: markdown, one audience, or print HTML (--out FILE)"},
	{groupPRD, "readiness", "report how ready a PRD is to commit"},
	{groupPRD, "commit", "freeze a named, audience-specific version of a PRD"},
	{groupPRD, "versions", "list a PRD's committed versions"},
	{groupPRD, "revisions", "list a PRD's revisions"},
	{groupPRD, "snapshot", "snapshot a PRD's current revision (--note)"},

	{groupRoadmap, "roadmap", "list, show, add, update, move or delete nodes; mine; delegate, handback; validate; progress"},
	{groupRoadmap, "members", "list a workspace's members and their user ids (for delegate and --owner)"},
	{groupRoadmap, "member", "member role --role admin|member <ws> <user>; member remove <ws> <user>"},
	{groupRoadmap, "presence", "show who is active on a project's roadmap"},
	{groupRoadmap, "build", "draft a roadmap from an approved PRD (apply it with build-confirm)"},
	{groupRoadmap, "build-confirm", "apply a drafted roadmap (-force to erase evidence and done nodes)"},
	{groupRoadmap, "expand", "draft children for a roadmap node (apply with expand-confirm)"},
	{groupRoadmap, "expand-confirm", "apply drafted children"},
	{groupRoadmap, "agent-context", "fetch the agent briefing for a project (--mode, --focus)"},
	{groupRoadmap, "checkpoint", "record an agent progress checkpoint (--node)"},
	{groupRoadmap, "propose", "propose a roadmap change for review (--kind, --node, --item)"},
	{groupRoadmap, "proposal", "proposal decide: accept (applying its items unless --no-apply) or reject a proposal"},

	{groupAdmin, "invite", "invite one email address (invite revoke <id> cancels)"},
	{groupAdmin, "invites", "list invitations"},
	{groupAdmin, "signup-mode", "show or set invitation_only | open | closed"},
}
