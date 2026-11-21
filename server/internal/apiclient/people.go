package apiclient

import (
	"encoding/json"
	"net/http"
)

// The workspace, member, update and profile writes (rv5-PARA-06). The server
// keeps them for a human's session, so they are CLI verbs; MCP has no tool.

// DeleteWorkspace deletes a workspace and everything in it (DELETE
// /workspaces/{id}; its creator only).
func (c *Client) DeleteWorkspace(workspaceID string) error {
	var discard json.RawMessage
	return c.do(http.MethodDelete, apiPath("workspaces", workspaceID), nil, &discard)
}

// UpdateMember sets a member's role, admin or member
// (PATCH /workspaces/{id}/members/{userId}; workspace admins only).
func (c *Client) UpdateMember(workspaceID, userID, role string) (json.RawMessage, error) {
	return c.requestJSON(http.MethodPatch, apiPath("workspaces", workspaceID, "members", userID), map[string]string{"role": role})
}

// RemoveMember removes a member from a workspace
// (DELETE /workspaces/{id}/members/{userId}).
func (c *Client) RemoveMember(workspaceID, userID string) error {
	var discard json.RawMessage
	return c.do(http.MethodDelete, apiPath("workspaces", workspaceID, "members", userID), nil, &discard)
}

// DeleteProjectUpdate deletes one project update (DELETE /updates/{id}; its
// author or a workspace admin).
func (c *Client) DeleteProjectUpdate(updateID string) error {
	var discard json.RawMessage
	return c.do(http.MethodDelete, apiPath("updates", updateID), nil, &discard)
}

// MePatch is the profile edit the CLI fills and the PATCH /me body: nil
// leaves a field as it is, a pointer to "" clears the bio or the avatar (fill
// those with PatchText). The server reads an absent avatarUrl as "keep".
type MePatch struct {
	Username  *string  `json:"username,omitempty"`
	Bio       *string  `json:"bio,omitempty"`
	Keywords  []string `json:"keywords,omitempty"`
	AvatarURL *string  `json:"avatarUrl,omitempty"`
}

// Over fills the fields PATCH /me rewrites on every call (username, bio,
// keywords) from current, the user as Whoami read it.
// shortcut: read then write, as ProjectPatch.Over.
func (p MePatch) Over(current json.RawMessage) (MePatch, error) {
	var cur struct {
		Username, Bio string
		Keywords      []string
	}
	if err := json.Unmarshal(current, &cur); err != nil {
		return p, err
	}
	p.Username = keep(p.Username, cur.Username)
	p.Bio = keep(p.Bio, cur.Bio)
	if p.Keywords == nil {
		p.Keywords = cur.Keywords
	}
	return p, nil
}

// UpdateMe sends patch as it is to PATCH /me.
func (c *Client) UpdateMe(patch MePatch) (json.RawMessage, error) {
	return c.requestJSON(http.MethodPatch, "/me", patch)
}

// Apply reads the signed-in user, fills what p leaves out (Over) and sends
// the PATCH through c, as ProjectPatch.Apply.
func (p MePatch) Apply(c *Client) (json.RawMessage, error) {
	cur, err := c.Whoami()
	if err == nil {
		p, err = p.Over(cur)
	}
	if err != nil {
		return nil, err
	}
	return c.UpdateMe(p)
}
