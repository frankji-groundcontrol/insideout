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
