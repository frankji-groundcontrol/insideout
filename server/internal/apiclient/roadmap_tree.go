package apiclient

import (
	"encoding/json"
	"net/http"
)

// The task tree (docs/architecture/roadmap-tree.md): one node with its
// breadcrumb, the caller's own nodes, delegation and its history, the
// workspace members a node can be handed to, and the nodes one waits on.

// RoadmapNode reads one node with its ancestors and project
// (GET /roadmap/{id}).
func (c *Client) RoadmapNode(nodeID string) (json.RawMessage, error) {
	return c.get(apiPath("roadmap", nodeID))
}

// RoadmapMine lists the topmost nodes the caller is in charge of in a
// workspace (GET /workspaces/{id}/roadmap/mine), or in every workspace of
// theirs when workspaceID is "" (GET /roadmap/mine).
func (c *Client) RoadmapMine(workspaceID string) (json.RawMessage, error) {
	if workspaceID == "" {
		return c.get(apiPath("roadmap", "mine"))
	}
	return c.get(apiPath("workspaces", workspaceID, "roadmap", "mine"))
}

// RoadmapDelegate hands a node to a workspace member
// (POST /roadmap/{id}/delegate); note may be "".
func (c *Client) RoadmapDelegate(nodeID, toUserID, note string) (json.RawMessage, error) {
	body := map[string]any{"toUserId": toUserID}
	if note != "" {
		body["note"] = note
	}
	return c.requestJSON(http.MethodPost, apiPath("roadmap", nodeID, "delegate"), body)
}

// RoadmapHandBack returns a node the caller owns to whoever delegated it
// (POST /roadmap/{id}/handback); note may be "".
func (c *Client) RoadmapHandBack(nodeID, note string) (json.RawMessage, error) {
	body := map[string]any{}
	if note != "" {
		body["note"] = note
	}
	return c.requestJSON(http.MethodPost, apiPath("roadmap", nodeID, "handback"), body)
}

// RoadmapAssignments lists who held a node, newest first
// (GET /roadmap/{id}/assignments).
func (c *Client) RoadmapAssignments(nodeID string) (json.RawMessage, error) {
	return c.get(apiPath("roadmap", nodeID, "assignments"))
}

// Members lists a workspace's members with their user ids
// (GET /workspaces/{id}/members).
func (c *Client) Members(workspaceID string) (json.RawMessage, error) {
	return c.get(apiPath("workspaces", workspaceID, "members"))
}

// AddRoadmapDependency makes a node wait on another node of its project
// (POST /roadmap/{id}/dependencies) and returns the node's view; an existing
// link is no change.
func (c *Client) AddRoadmapDependency(nodeID, dependsOnID string) (json.RawMessage, error) {
	return c.requestJSON(http.MethodPost, apiPath("roadmap", nodeID, "dependencies"), map[string]any{"dependsOnId": dependsOnID})
}

// RemoveRoadmapDependency drops the link
// (DELETE /roadmap/{id}/dependencies/{dependsOnId}).
func (c *Client) RemoveRoadmapDependency(nodeID, dependsOnID string) error {
	var discard json.RawMessage
	return c.do(http.MethodDelete, apiPath("roadmap", nodeID, "dependencies", dependsOnID), nil, &discard)
}
