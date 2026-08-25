package apiclient

import (
	"net/http"

	"encoding/json"
)

// The CRUD completion verbs — projections of existing API routes that
// had no CLI/MCP tool (docs/plans/2026-08-25-agent-crud-tool-surface.md):
// the idea inbox lifecycle, PRD section authoring, reading GitHub
// evidence, and workspace/project bootstrapping. Reads return raw JSON
// so surfaces cannot drift from the API's truth.

// Ideas lists a workspace's idea inbox (full content included).
func (c *Client) Ideas(workspaceID string) (json.RawMessage, error) {
	return c.getRaw("/workspaces/" + workspaceID + "/ideas")
}

// UpdateIdea rewrites an idea's title and content (title is required by
// the API on update).
func (c *Client) UpdateIdea(id, title, content string) (json.RawMessage, error) {
	return c.requestJSON(http.MethodPatch, "/ideas/"+id, map[string]string{
		"title":   title,
		"content": content,
	})
}

// DeleteIdea drops an idea — a soft delete: the row stays in the inbox
// with status "dropped" (ListIdeas returns every status; filter
// client-side).
func (c *Client) DeleteIdea(id string) error {
	var discard json.RawMessage
	return c.do(http.MethodDelete, "/ideas/"+id, nil, &discard)
}

// UpdatePrd patches a PRD's title (nil = unchanged) and/or sections;
// a nil/empty sections map sends no sections key.
func (c *Client) UpdatePrd(id string, title *string, sections map[string]string) (json.RawMessage, error) {
	body := map[string]any{}
	if title != nil {
		body["title"] = *title
	}
	if len(sections) > 0 {
		body["sections"] = sections
	}
	return c.requestJSON(http.MethodPatch, "/prds/"+id, body)
}

// SetPrdStatus transitions a PRD's status.
func (c *Client) SetPrdStatus(id, status string) (json.RawMessage, error) {
	return c.requestJSON(http.MethodPost, "/prds/"+id+"/status", map[string]string{"status": status})
}

// NodeEvidence returns the GitHub evidence attached to a roadmap node —
// the read half of the bind → sync → checkpoint loop.
func (c *Client) NodeEvidence(nodeID string) (json.RawMessage, error) {
	return c.getRaw("/roadmap/" + nodeID + "/evidence")
}

// CreateWorkspace bootstraps a workspace (creator becomes owner).
func (c *Client) CreateWorkspace(title, description string) (json.RawMessage, error) {
	return c.postRaw("/workspaces", map[string]string{"title": title, "description": description})
}

// CreateProject bootstraps a project inside a workspace.
func (c *Client) CreateProject(workspaceID, title, description string) (json.RawMessage, error) {
	return c.postRaw("/workspaces/"+workspaceID+"/projects", map[string]string{
		"title": title, "description": description,
	})
}
