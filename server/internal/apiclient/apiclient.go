// Package apiclient is the shared HTTP client for every non-web
// projection of the product (CLI today, MCP next). It speaks the same
// /api/v1 contract as the Flutter client — bearer auth, JSON bodies —
// and deliberately returns raw JSON for resource reads so surfaces
// cannot drift from the API's truth.
package apiclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	base  string
	token string
	hc    *http.Client
}

func New(base string) *Client {
	return &Client{
		base: strings.TrimRight(base, "/"),
		hc:   &http.Client{Timeout: 20 * time.Second},
	}
}

// SetToken installs a previously obtained access token (INSIDEOUT_TOKEN).
func (c *Client) SetToken(token string) { c.token = token }

// Token exposes the current access token (after Login) so callers such
// as `insideout login` can hand it to the user.
func (c *Client) Token() string { return c.token }

// Login exchanges credentials for an access token (POST /auth/login —
// the same route the web client uses; the refresh token stays unused by
// CLI/MCP sessions, which simply log in again when the token expires).
func (c *Client) Login(email, password string) error {
	var out struct {
		AccessToken string `json:"accessToken"`
		Error       string `json:"error"`
	}
	if err := c.do(http.MethodPost, "/auth/login", map[string]string{
		"email":    email,
		"password": password,
	}, &out); err != nil {
		return err
	}
	if out.AccessToken == "" {
		return fmt.Errorf("login: no accessToken in response%s", errSuffix(out.Error))
	}
	c.token = out.AccessToken
	return nil
}

// Whoami returns GET /me.
func (c *Client) Whoami() (json.RawMessage, error) { return c.get("/me") }

// Workspaces returns GET /workspaces.
func (c *Client) Workspaces() (json.RawMessage, error) { return c.get("/workspaces") }

// Projects returns GET /workspaces/{id}/projects.
func (c *Client) Projects(workspaceID string) (json.RawMessage, error) {
	return c.get("/workspaces/" + workspaceID + "/projects")
}

// Prd returns GET /prds/{id}.
func (c *Client) Prd(id string) (json.RawMessage, error) { return c.get("/prds/" + id) }

// BuildFromPrd runs the agent planner: PRD → project with a branched
// roadmap (POST /prds/{id}/build). expectedCount <= 0 omits the hint.
func (c *Client) BuildFromPrd(prdID string, expectedCount int) (json.RawMessage, error) {
	body := map[string]any{}
	if expectedCount > 0 {
		body["expectedCount"] = expectedCount
	}
	var out json.RawMessage
	if err := c.do(http.MethodPost, "/prds/"+prdID+"/build", body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ExpandNode grows one roadmap node into AI-proposed children
// (POST /roadmap/{id}/expand).
func (c *Client) ExpandNode(nodeID string) (json.RawMessage, error) {
	var out json.RawMessage
	if err := c.do(http.MethodPost, "/roadmap/"+nodeID+"/expand", map[string]any{}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// RoadmapList returns GET /projects/{id}/roadmap.
func (c *Client) RoadmapList(projectID string) (json.RawMessage, error) {
	return c.get("/projects/" + projectID + "/roadmap")
}

// RoadmapAdd creates a node (POST /projects/{id}/roadmap); parentID may
// be empty for a root, or "root" is not accepted — pass nil for root.
func (c *Client) RoadmapAdd(projectID, title, description string, parentID *string) (json.RawMessage, error) {
	body := map[string]any{"title": title, "description": description}
	if parentID != nil {
		body["parentId"] = *parentID
	}
	var out json.RawMessage
	if err := c.do(http.MethodPost, "/projects/"+projectID+"/roadmap", body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// RoadmapUpdate partially updates a node (PATCH /roadmap/{id}); nil
// fields are left untouched server-side.
func (c *Client) RoadmapUpdate(nodeID string, title, description, status *string) (json.RawMessage, error) {
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
	var out json.RawMessage
	if err := c.do(http.MethodPatch, "/roadmap/"+nodeID, body, &out); err != nil {
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
	if err := c.do(http.MethodPost, "/roadmap/"+nodeID+"/move", body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// RoadmapDelete removes a node (DELETE /roadmap/{id}); any response
// body is discarded.
func (c *Client) RoadmapDelete(nodeID string) error {
	var discard json.RawMessage
	return c.do(http.MethodDelete, "/roadmap/"+nodeID, nil, &discard)
}

// SetRepo binds a GitHub repository to a project (PUT /projects/{id}/repo).
func (c *Client) SetRepo(projectID, repoURL string) (json.RawMessage, error) {
	var out json.RawMessage
	if err := c.do(http.MethodPut, "/projects/"+projectID+"/repo", map[string]string{"repoUrl": repoURL}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// SyncGithub runs the pull-based GitHub evidence sync
// (POST /projects/{id}/sync-github).
func (c *Client) SyncGithub(projectID string) (json.RawMessage, error) {
	var out json.RawMessage
	if err := c.do(http.MethodPost, "/projects/"+projectID+"/sync-github", map[string]any{}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Guide returns the scaffolded insideout.yaml matching guide
// (GET /projects/{id}/guide, text/yaml) for the user to commit.
func (c *Client) Guide(projectID string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, c.base+"/projects/"+projectID+"/guide", nil)
	if err != nil {
		return nil, fmt.Errorf("apiclient: build request: %w", err)
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

func (c *Client) get(path string) (json.RawMessage, error) {
	var out json.RawMessage
	if err := c.do(http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) do(method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("apiclient: encode body: %w", err)
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequest(method, c.base+path, reader)
	if err != nil {
		return fmt.Errorf("apiclient: build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("apiclient: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return fmt.Errorf("apiclient: read response: %w", err)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("apiclient: %s %s: unauthorized (check INSIDEOUT_TOKEN or login)", method, path)
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("apiclient: %s %s: %s: %s", method, path, resp.Status, strings.TrimSpace(string(raw)))
	}
	if len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("apiclient: %s %s: decode: %w", method, path, err)
	}
	return nil
}

func errSuffix(msg string) string {
	if msg == "" {
		return ""
	}
	return ": " + msg
}

// CommitPrd performs the human Commit: freeze the working PRD as an
// immutable named version (POST /prds/{id}/commit).
func (c *Client) CommitPrd(prdID, name, audience, summary string, unresolved []string, decisionNote string) (json.RawMessage, error) {
	if unresolved == nil {
		unresolved = []string{}
	}
	body := map[string]any{
		"name": name, "primaryAudience": audience, "changeSummary": summary,
		"unresolved": unresolved, "decisionNote": decisionNote,
	}
	var out json.RawMessage
	if err := c.do(http.MethodPost, "/prds/"+prdID+"/commit", body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// PrdVersions lists a PRD's commits, newest first
// (GET /prds/{id}/commits).
func (c *Client) PrdVersions(prdID string) (json.RawMessage, error) {
	return c.get("/prds/" + prdID + "/commits")
}

// PrdReadiness returns the per-audience gap disclosure
// (GET /prds/{id}/readiness) for "form a version now".
func (c *Client) PrdReadiness(prdID string) (json.RawMessage, error) {
	return c.get("/prds/" + prdID + "/readiness")
}
