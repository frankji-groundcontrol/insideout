package apiclient

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// Projects beyond list and create, their updates feed, one workspace and a
// workspace's PRDs (rv5-PARB-04, PARA-06, PARB-07, PARB-03). Reads return the
// API's JSON as it is.

// Project reads one project with its newest page of updates and the
// nextCursor for the rest (GET /projects/{id}).
func (c *Client) Project(projectID string) (json.RawMessage, error) {
	return c.get(apiPath("projects", projectID))
}

// ProjectPatch is the project edit the CLI fills and the PATCH /projects/{id}
// body: nil leaves a field as it is, a pointer to "" clears the description or
// the owner (fill those with PatchText).
type ProjectPatch struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Status      *string `json:"status,omitempty"`
	OwnerID     *string `json:"ownerId,omitempty"`
}

// Over fills the fields PATCH /projects/{id} rewrites on every call (title,
// description, status) from current, the project as Project read it, so a
// field the patch leaves out keeps its value.
// shortcut: read then write, so an edit landing between the two is lost;
// upgrade when the route keeps the stored value for an absent key.
func (p ProjectPatch) Over(current json.RawMessage) (ProjectPatch, error) {
	var cur struct{ Title, Description, Status string }
	if err := json.Unmarshal(current, &cur); err != nil {
		return p, err
	}
	p.Title = keep(p.Title, cur.Title)
	p.Description = keep(p.Description, cur.Description)
	p.Status = keep(p.Status, cur.Status)
	return p, nil
}

// UpdateProject sends patch as it is to PATCH /projects/{id}.
func (c *Client) UpdateProject(projectID string, patch ProjectPatch) (json.RawMessage, error) {
	return c.requestJSON(http.MethodPatch, apiPath("projects", projectID), patch)
}

// Apply is the project edit of the CLI and MCP: it reads the project, fills
// what p leaves out (Over) and sends the PATCH through c.
func (p ProjectPatch) Apply(c *Client, projectID string) (json.RawMessage, error) {
	cur, err := c.Project(projectID)
	if err == nil {
		p, err = p.Over(cur)
	}
	if err != nil {
		return nil, err
	}
	return c.UpdateProject(projectID, p)
}

// DeleteProject deletes a project and everything under it (DELETE
// /projects/{id}; workspace admins only, never an agent token).
func (c *Client) DeleteProject(projectID string) error {
	var discard json.RawMessage
	return c.do(http.MethodDelete, apiPath("projects", projectID), nil, &discard)
}

// UpdatesQuery narrows one page of a project's updates: Before is the
// nextCursor of the previous page ("" for the newest), Limit 1 to 100 (0 for
// the server's 50), Node keeps the updates about that node and the nodes
// below it, Pending only the proposals still waiting for the caller's
// decision.
type UpdatesQuery struct {
	Before, Node string
	Limit        int
	Pending      bool
}

// ProjectUpdates reads one page of a project's updates, newest first
// (GET /projects/{id}/updates).
func (c *Client) ProjectUpdates(projectID string, q UpdatesQuery) (json.RawMessage, error) {
	var kv []string
	if q.Before != "" {
		kv = append(kv, "before", q.Before)
	}
	if q.Limit > 0 {
		kv = append(kv, "limit", strconv.Itoa(q.Limit))
	}
	if q.Node != "" {
		kv = append(kv, "node", q.Node)
	}
	if q.Pending {
		kv = append(kv, "pending", "true")
	}
	return c.get(withQuery(apiPath("projects", projectID, "updates"), kv...))
}

// ProjectUpdate is a human's progress, blocker or note
// (POST /projects/{id}/updates).
type ProjectUpdate struct {
	Kind    string `json:"kind"`
	Content string `json:"content"`
}

// AddProjectUpdate posts u to a project's updates feed.
func (c *Client) AddProjectUpdate(projectID string, u ProjectUpdate) (json.RawMessage, error) {
	return c.postRaw(apiPath("projects", projectID, "updates"), u)
}

// EditProjectUpdate rewrites an update's text (PATCH /updates/{id}; its
// author or a workspace admin).
func (c *Client) EditProjectUpdate(updateID, content string) (json.RawMessage, error) {
	return c.requestJSON(http.MethodPatch, apiPath("updates", updateID), map[string]string{"content": content})
}

// Workspace reads one workspace (GET /workspaces/{id}).
func (c *Client) Workspace(workspaceID string) (json.RawMessage, error) {
	return c.get(apiPath("workspaces", workspaceID))
}

// WorkspacePatch is the workspace edit the CLI fills and the
// PATCH /workspaces/{id} body: nil leaves a field as it is, a pointer to ""
// clears the description or the cover.
type WorkspacePatch struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	CoverURL    *string `json:"coverUrl,omitempty"`
}

// Over fills title and description, which PATCH /workspaces/{id} rewrites on
// every call, from current, the workspace as Workspace read it.
// shortcut: read then write, as ProjectPatch.Over.
func (p WorkspacePatch) Over(current json.RawMessage) (WorkspacePatch, error) {
	var cur struct{ Title, Description string }
	if err := json.Unmarshal(current, &cur); err != nil {
		return p, err
	}
	p.Title = keep(p.Title, cur.Title)
	p.Description = keep(p.Description, cur.Description)
	return p, nil
}

// UpdateWorkspace sends patch as it is to PATCH /workspaces/{id}.
func (c *Client) UpdateWorkspace(workspaceID string, patch WorkspacePatch) (json.RawMessage, error) {
	return c.requestJSON(http.MethodPatch, apiPath("workspaces", workspaceID), patch)
}

// Apply reads the workspace, fills what p leaves out (Over) and sends the
// PATCH through c, as ProjectPatch.Apply.
func (p WorkspacePatch) Apply(c *Client, workspaceID string) (json.RawMessage, error) {
	cur, err := c.Workspace(workspaceID)
	if err == nil {
		p, err = p.Over(cur)
	}
	if err != nil {
		return nil, err
	}
	return c.UpdateWorkspace(workspaceID, p)
}

// Prds lists every PRD of a workspace, newest first
// (GET /workspaces/{id}/prds).
func (c *Client) Prds(workspaceID string) (json.RawMessage, error) {
	return c.get(apiPath("workspaces", workspaceID, "prds"))
}

// keep is v, or stored when v is nil.
func keep(v *string, stored string) *string {
	if v != nil {
		return v
	}
	return &stored
}
