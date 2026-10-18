// Package apiclient is the shared HTTP client for every non-web
// projection of the product (CLI today, MCP next). It speaks the same
// /api/v1 contract as the Flutter client — bearer auth, JSON bodies —
// and deliberately returns raw JSON for resource reads so surfaces
// cannot drift from the API's truth.
package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// A Client serves one caller at a time.
type Client struct {
	base    string
	token   string
	refresh string
	renew   *Renewal        // nil: a 401 is final (see SetRenewal)
	ctx     context.Context // nil: context.Background (see SetContext)
	hc      *http.Client
}

func New(base string) *Client {
	c := &Client{base: strings.TrimRight(base, "/")}
	// No Timeout here: every request carries the deadline of its route (see
	// deadlineFor), which a client-wide number can not do.
	c.hc = &http.Client{Transport: renewTransport{c}}
	return c
}

// SetContext binds ctx to every request this client makes from now on, the
// renewal of an expired session and the agent-token trade included: cancelling
// it (a cancelled MCP call, a Ctrl-C in the CLI) ends the request in flight. It
// is set in place, not copied: a Client serves one caller at a time, so there is
// no second view to fall out of step with the first one's session.
func (c *Client) SetContext(ctx context.Context) { c.ctx = ctx }

// Context returns the context SetContext bound, or context.Background.
func (c *Client) Context() context.Context {
	if c.ctx == nil {
		return context.Background()
	}
	return c.ctx
}

// newRequest builds a request for path on the client's context, bounded by the
// deadline its route needs (deadlineFor). Every request the client makes starts
// here. The caller defers cancel once it has read the response body, so the
// deadline covers the body too and its timer is released.
func (c *Client) newRequest(method, path string, body io.Reader) (*http.Request, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(c.Context(), deadlineFor(method, path))
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, body)
	if err != nil {
		cancel()
		return nil, nil, fmt.Errorf("apiclient: build request: %w", err)
	}
	return req, cancel, nil
}

// SetToken installs a previously obtained access token (INSIDEOUT_TOKEN).
func (c *Client) SetToken(token string) { c.token = token }

// Token exposes the current access token (after Login) so callers such
// as `insideout login` can hand it to the user.
func (c *Client) Token() string { return c.token }

// RefreshToken exposes the refresh token the last sign-in returned, so a
// caller can persist it next to the access token.
func (c *Client) RefreshToken() string { return c.refresh }

// Login exchanges credentials for an access token and a refresh token
// (POST /auth/login — the same route the web client uses).
func (c *Client) Login(email, password string) error {
	var out authBody
	if err := c.do(http.MethodPost, "/auth/login", map[string]string{
		"email":    email,
		"password": password,
	}, &out); err != nil {
		return err
	}
	if out.AccessToken == "" {
		return fmt.Errorf("login: no accessToken in response%s", errSuffix(out.Error))
	}
	c.token, c.refresh = out.AccessToken, out.RefreshToken
	return nil
}

// Whoami returns GET /me.
func (c *Client) Whoami() (json.RawMessage, error) { return c.get("/me") }

// Workspaces returns GET /workspaces.
func (c *Client) Workspaces() (json.RawMessage, error) { return c.get("/workspaces") }

// Projects returns GET /workspaces/{id}/projects.
func (c *Client) Projects(workspaceID string) (json.RawMessage, error) {
	return c.get(apiPath("workspaces", workspaceID, "projects"))
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
	req, cancel, err := c.newRequest(method, path, reader)
	if err != nil {
		return err
	}
	defer cancel()
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
	if resp.StatusCode == http.StatusUnauthorized && !strings.Contains(path, "/auth/") {
		return c.unauthorized(method + " " + path)
	} // on /auth/* a 401 is the flow's verdict (wrong password), shown with the server's words below
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

// getBytes GETs a path whose answer is not JSON (export markdown/HTML)
// and returns the body verbatim.
func (c *Client) getBytes(path string) ([]byte, error) {
	req, cancel, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer cancel()
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("apiclient: GET %s: %w", path, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("apiclient: read %s: %w", path, err)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, c.unauthorized("GET " + path)
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("apiclient: GET %s: %s: %s", path, resp.Status, strings.TrimSpace(string(raw)))
	}
	return raw, nil
}

// postRaw / getRaw are the raw-JSON request helpers for projections that
// print the API's answer verbatim (admin verbs).
func (c *Client) postRaw(path string, body any) (json.RawMessage, error) {
	return c.requestJSON(http.MethodPost, path, body)
}

func (c *Client) getRaw(path string) (json.RawMessage, error) {
	return c.requestJSON(http.MethodGet, path, nil)
}

func (c *Client) requestJSON(method, path string, body any) (json.RawMessage, error) {
	var out json.RawMessage
	if err := c.do(method, path, body, &out); err != nil {
		return nil, err
	}
	return out, nil
}
