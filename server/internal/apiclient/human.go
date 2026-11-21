package apiclient

import (
	"encoding/json"
	"net/http"
)

// The routes the server keeps for a human's own session (an agent token is
// refused): account deletion and the validation of a roadmap node
// (rv5-PARA-03, PARC-04). The CLI has verbs for them; MCP deliberately has none.

// DeleteMe tombstones the signed-in account (DELETE /me), which asks for the
// current password again and revokes every session.
func (c *Client) DeleteMe(password string) error {
	var discard json.RawMessage
	return c.do(http.MethodDelete, "/me", map[string]string{"password": password}, &discard)
}

// Observation is one human-entered validation observation
// (POST /roadmap/{id}/evidence): what was seen, where, and the day
// (YYYY-MM-DD).
type Observation struct {
	Detail     string `json:"detail"`
	SourceURL  string `json:"sourceUrl"`
	ObservedAt string `json:"observedAt"`
}

// AddObservation records o on a leaf node.
func (c *Client) AddObservation(nodeID string, o Observation) (json.RawMessage, error) {
	return c.postRaw(apiPath("roadmap", nodeID, "evidence"), o)
}

// ValidateNode confirms a done node as validated, with the reason it counts
// (POST /roadmap/{id}/validate).
func (c *Client) ValidateNode(nodeID, reason string) (json.RawMessage, error) {
	return c.postRaw(apiPath("roadmap", nodeID, "validate"), map[string]string{"reason": reason})
}
