package apiclient

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// BuildFromPrd drafts a branched roadmap and writes nothing
// (POST /prds/{id}/build). ConfirmBuildFromPrd persists that tree;
// expectedCount <= 0 omits the replace hint.
func (c *Client) BuildFromPrd(prdID string) (json.RawMessage, error) {
	var out json.RawMessage
	if err := c.do(http.MethodPost, apiPath("prds", prdID, "build"), map[string]any{}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ConfirmBuildFromPrd writes a previously drafted tree
// (POST /prds/{id}/build/confirm).
func (c *Client) ConfirmBuildFromPrd(prdID string, tree json.RawMessage, expectedCount int) (json.RawMessage, error) {
	body := map[string]any{"tree": json.RawMessage(tree)}
	if expectedCount > 0 {
		body["expectedCount"] = expectedCount
	}
	var out json.RawMessage
	if err := c.do(http.MethodPost, apiPath("prds", prdID, "build", "confirm"), body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ExpandNode drafts subtasks for one node and writes nothing
// (POST /roadmap/{id}/expand).
func (c *Client) ExpandNode(nodeID string) (json.RawMessage, error) {
	var out json.RawMessage
	if err := c.do(http.MethodPost, apiPath("roadmap", nodeID, "expand"), map[string]any{}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ConfirmExpandNode writes previously drafted subtasks
// (POST /roadmap/{id}/expand/confirm).
func (c *Client) ConfirmExpandNode(nodeID string, children json.RawMessage) (json.RawMessage, error) {
	var out json.RawMessage
	body := map[string]any{"children": json.RawMessage(children)}
	if err := c.do(http.MethodPost, apiPath("roadmap", nodeID, "expand", "confirm"), body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// RoadmapProgress returns the Progress view
// (GET /projects/{id}/roadmap/progress).
func (c *Client) RoadmapProgress(projectID string) (json.RawMessage, error) {
	return c.get(apiPath("projects", projectID, "roadmap", "progress"))
}

// RoadmapList returns GET /projects/{id}/roadmap.
func (c *Client) RoadmapList(projectID string) (json.RawMessage, error) {
	return c.get(apiPath("projects", projectID, "roadmap"))
}

// RoadmapAdd creates a node (POST /projects/{id}/roadmap); parentID may
// be empty for a root, or "root" is not accepted — pass nil for root.
func (c *Client) RoadmapAdd(projectID, title, description string, parentID *string) (json.RawMessage, error) {
	body := map[string]any{"title": title, "description": description}
	if parentID != nil {
		body["parentId"] = *parentID
	}
	var out json.RawMessage
	if err := c.do(http.MethodPost, apiPath("projects", projectID, "roadmap"), body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// RoadmapUpdate partially updates a node (PATCH /roadmap/{id}); nil
// fields are left untouched server-side. deadline: non-nil sets
// (RFC3339) or clears (empty string).
func (c *Client) RoadmapUpdate(nodeID string, title, description, status, deadline *string) (json.RawMessage, error) {
	body := map[string]any{}
	if title != nil {
		body["title"] = *title
	}
	if description != nil {
		body["description"] = *description
	}
	if status != nil {
		body["status"] = *status
	}
	if deadline != nil {
		body["deadline"] = *deadline
	}
	var out json.RawMessage
	if err := c.do(http.MethodPatch, apiPath("roadmap", nodeID), body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// RoadmapMove re-parents and/or repositions a branch
// (POST /roadmap/{id}/move). nil parentID makes it a root.
func (c *Client) RoadmapMove(nodeID string, parentID *string, position *int) (json.RawMessage, error) {
	body := map[string]any{}
	if parentID != nil {
		body["parentId"] = *parentID
	}
	if position != nil {
		body["position"] = *position
	}
	var out json.RawMessage
	if err := c.do(http.MethodPost, apiPath("roadmap", nodeID, "move"), body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// RoadmapDelete removes a node (DELETE /roadmap/{id}); any response
// body is discarded.
func (c *Client) RoadmapDelete(nodeID string) error {
	var discard json.RawMessage
	return c.do(http.MethodDelete, apiPath("roadmap", nodeID), nil, &discard)
}

// SetRepo binds a GitHub repository to a project (PUT /projects/{id}/repo).
func (c *Client) SetRepo(projectID, repoURL string) (json.RawMessage, error) {
	var out json.RawMessage
	if err := c.do(http.MethodPut, apiPath("projects", projectID, "repo"), map[string]string{"repoUrl": repoURL}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// SyncGithub runs the pull-based GitHub evidence sync
// (POST /projects/{id}/sync-github).
func (c *Client) SyncGithub(projectID string) (json.RawMessage, error) {
	var out json.RawMessage
	if err := c.do(http.MethodPost, apiPath("projects", projectID, "sync-github"), map[string]any{}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Guide returns the scaffolded insideout.yaml matching guide
// (GET /projects/{id}/guide, text/yaml) for the user to commit.
func (c *Client) Guide(projectID string) ([]byte, error) {
	req, err := c.newRequest(http.MethodGet, apiPath("projects", projectID, "guide"), nil)
	if err != nil {
		return nil, err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("apiclient: GET guide: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("apiclient: read guide: %w", err)
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("apiclient: GET guide: %s: %s", resp.Status, strings.TrimSpace(string(raw)))
	}
	return raw, nil
}

// Cursor moves this session's canvas cursor (POST /projects/{id}/cursor).
func (c *Client) Cursor(projectID string, x, y float64) error {
	return c.do(http.MethodPost, apiPath("projects", projectID, "cursor"),
		map[string]float64{"x": x, "y": y}, nil)
}

// ProjectPresence returns who is currently viewing a project
// (GET /projects/{id}/presence).
func (c *Client) ProjectPresence(projectID string) (json.RawMessage, error) {
	return c.get(apiPath("projects", projectID, "presence"))
}
