package apiclient

// sweepRV5 are the escape-sweep rows of the review 5 verbs: one project, its
// updates, one workspace, its PRDs and members, the profile, and the
// human-only writes. TestSweepCoversEveryClientMethod holds them to the same
// "every method has a row" rule as the rows in escape_test.go.
var sweepRV5 = []sweepCase{
	{"Project", "GET", "/projects/{h}", func(c *Client, h string) error { return drop(c.Project(h)) }},
	{"UpdateProject", "PATCH", "/projects/{h}", func(c *Client, h string) error {
		return drop(c.UpdateProject(h, ProjectPatch{Title: &h, Description: &h, Status: &h, OwnerID: &h}))
	}},
	{"DeleteProject", "DELETE", "/projects/{h}", func(c *Client, h string) error { return c.DeleteProject(h) }},
	{"ProjectUpdates", "GET", "/projects/{h}/updates", func(c *Client, h string) error { return drop(c.ProjectUpdates(h, UpdatesQuery{})) }},
	{"ProjectUpdates", "GET", "/projects/{h}/updates?before={h}&limit=5", func(c *Client, h string) error { return drop(c.ProjectUpdates(h, UpdatesQuery{Before: h, Limit: 5})) }},
	{"ProjectUpdates", "GET", "/projects/{h}/updates?node={h}&pending=true", func(c *Client, h string) error {
		return drop(c.ProjectUpdates(h, UpdatesQuery{Node: h, Pending: true}))
	}},
	{"AddProjectUpdate", "POST", "/projects/{h}/updates", func(c *Client, h string) error {
		return drop(c.AddProjectUpdate(h, ProjectUpdate{Kind: h, Content: h}))
	}},
	{"EditProjectUpdate", "PATCH", "/updates/{h}", func(c *Client, h string) error { return drop(c.EditProjectUpdate(h, h)) }},
	{"DeleteProjectUpdate", "DELETE", "/updates/{h}", func(c *Client, h string) error { return c.DeleteProjectUpdate(h) }},
	{"DeleteWorkspace", "DELETE", "/workspaces/{h}", func(c *Client, h string) error { return c.DeleteWorkspace(h) }},
	{"UpdateMember", "PATCH", "/workspaces/{h}/members/{h}", func(c *Client, h string) error { return drop(c.UpdateMember(h, h, h)) }},
	{"RemoveMember", "DELETE", "/workspaces/{h}/members/{h}", func(c *Client, h string) error { return c.RemoveMember(h, h) }},
	{"UpdateMe", "PATCH", "/me", func(c *Client, h string) error {
		return drop(c.UpdateMe(MePatch{Username: &h, Bio: &h, AvatarURL: &h}))
	}},
	{"Workspace", "GET", "/workspaces/{h}", func(c *Client, h string) error { return drop(c.Workspace(h)) }},
	{"UpdateWorkspace", "PATCH", "/workspaces/{h}", func(c *Client, h string) error {
		return drop(c.UpdateWorkspace(h, WorkspacePatch{Title: &h, Description: &h, CoverURL: &h}))
	}},
	{"Prds", "GET", "/workspaces/{h}/prds", func(c *Client, h string) error { return drop(c.Prds(h)) }},
	{"DeleteMe", "DELETE", "/me", func(c *Client, h string) error { return c.DeleteMe(h) }},
	{"AddObservation", "POST", "/roadmap/{h}/evidence", func(c *Client, h string) error {
		return drop(c.AddObservation(h, Observation{Detail: h, SourceURL: h, ObservedAt: h}))
	}},
	{"ValidateNode", "POST", "/roadmap/{h}/validate", func(c *Client, h string) error { return drop(c.ValidateNode(h, h)) }},
}

func init() { sweep = append(sweep, sweepRV5...) }
