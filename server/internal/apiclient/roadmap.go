package apiclient

import (
	"encoding/json"
	"net/http"
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
// (POST /prds/{id}/build/confirm). force discards nodes that have evidence or
// are done or validated, which the server otherwise refuses with 409
// roadmap_replace_has_history.
func (c *Client) ConfirmBuildFromPrd(prdID string, tree json.RawMessage, expectedCount int, force bool) (json.RawMessage, error) {
	body := map[string]any{"tree": json.RawMessage(tree)}
	if expectedCount > 0 {
		body["expectedCount"] = expectedCount
	}
	if force {
		body["force"] = true
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
// (GET /projects/{id}/roadmap/progress); a root other than "" limits it to
// that node's subtree (?root=).
func (c *Client) RoadmapProgress(projectID, root string) (json.RawMessage, error) {
	return c.get(withRoot(apiPath("projects", projectID, "roadmap", "progress"), root))
}

// RoadmapList returns GET /projects/{id}/roadmap; a root other than "" lists
// that node and the nodes below it (?root=).
func (c *Client) RoadmapList(projectID, root string) (json.RawMessage, error) {
	return c.get(withRoot(apiPath("projects", projectID, "roadmap"), root))
}

func withRoot(path, root string) string {
	if root == "" {
		return path
	}
	return withQuery(path, "root", root)
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

// RoadmapPatch is a partial node update (PATCH /roadmap/{id}). A nil pointer
// leaves its field alone; a pointer to "" sends "", which clears description
// and acceptance. Deadline is RFC3339 ("" leaves it) and ClearDeadline
// removes it; OwnerID is a member's user id ("" leaves it) and ClearOwner
// removes the owner. The server refuses an owner change to an agent token.
type RoadmapPatch struct {
	Title, Description, Status, Acceptance, CompletionReason *string
	Deadline                                                 string
	ClearDeadline                                            bool
	OwnerID                                                  string
	ClearOwner                                               bool
}

// RoadmapUpdate sends patch to PATCH /roadmap/{id}.
func (c *Client) RoadmapUpdate(nodeID string, patch RoadmapPatch) (json.RawMessage, error) {
	body := map[string]any{}
	for key, v := range map[string]*string{"title": patch.Title, "description": patch.Description,
		"status": patch.Status, "acceptance": patch.Acceptance, "completionReason": patch.CompletionReason} {
		if v != nil {
			body[key] = *v
		}
	}
	if patch.ClearDeadline {
		body["deadline"] = ""
	} else if patch.Deadline != "" {
		body["deadline"] = patch.Deadline
	}
	if patch.ClearOwner {
		body["ownerId"] = ""
	} else if patch.OwnerID != "" {
		body["ownerId"] = patch.OwnerID
	}
	var out json.RawMessage
	if err := c.do(http.MethodPatch, apiPath("roadmap", nodeID), body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// RoadmapMove re-parents and/or repositions a branch
// (POST /roadmap/{id}/move). nil parentID keeps the current parent; a pointer
// to "" makes the node a root (sent as an explicit null).
func (c *Client) RoadmapMove(nodeID string, parentID *string, position *int) (json.RawMessage, error) {
	body := map[string]any{}
	if parentID != nil && *parentID == "" {
		body["parentId"] = nil
	} else if parentID != nil {
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
// (GET /projects/{id}/guide, text/yaml) for the user to commit. It rides the
// shared raw GET, so a 401 is the same actionable ErrUnauthorized as everywhere
// (rv4-FR-F169).
func (c *Client) Guide(projectID string) ([]byte, error) {
	return c.getBytes(apiPath("projects", projectID, "guide"))
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
