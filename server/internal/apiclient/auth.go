package apiclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// The auth verbs — one per public /auth route — so every projection of the
// product (CLI, MCP) can authenticate all five ways the web can: password
// login, register, magic link, password reset, and code login. Every verb
// that opens a session keeps the access token and the refresh token it was
// given (same contract as Login).

// Register creates an account and stores its session token. inviteCode
// is required when the deployment is invitation-only (GET /auth/config).
func (c *Client) Register(email, password, username, inviteCode string) error {
	return c.authSession("/auth/register", map[string]string{
		"email":      email,
		"password":   password,
		"username":   username,
		"inviteCode": inviteCode,
	})
}

// RequestMagicLink emails a one-time sign-in link (uniform 200 contract —
// the API never reveals whether the account exists).
func (c *Client) RequestMagicLink(email string) error {
	return c.authRequest("/auth/magic-link", map[string]string{"email": email})
}

// ConsumeMagicLink exchanges an emailed link token for a session.
func (c *Client) ConsumeMagicLink(token string) error {
	return c.authSession("/auth/magic-link/consume", map[string]string{"token": token})
}

// RequestLoginCode emails a 6-digit sign-in code (uniform 200 contract).
func (c *Client) RequestLoginCode(email string) error {
	return c.authRequest("/auth/login-code", map[string]string{"email": email})
}

// VerifyLoginCode exchanges an emailed code for a session.
func (c *Client) VerifyLoginCode(email, code string) error {
	return c.authSession("/auth/login-code/verify", map[string]string{
		"email": email,
		"code":  code,
	})
}

// ForgotPassword emails a one-time reset link (uniform 200 contract).
func (c *Client) ForgotPassword(email string) error {
	return c.authRequest("/auth/forgot-password", map[string]string{"email": email})
}

// ResetPassword sets a new password from an emailed reset token; the API
// revokes every session of the account, so the next step is a fresh login.
func (c *Client) ResetPassword(token, password string) error {
	return c.authRequest("/auth/reset-password", map[string]string{
		"token":    token,
		"password": password,
	})
}

// authSession POSTs a credential exchange and stores the access and refresh
// tokens on success — the shared shape of Register and the two consume/verify
// verbs.
func (c *Client) authSession(path string, body any) error {
	out, err := c.authPost(path, body)
	if err != nil {
		return err
	}
	if out.AccessToken == "" {
		return fmt.Errorf("POST %s: no accessToken in response%s", path, errSuffix(out.Error))
	}
	c.token, c.refresh = out.AccessToken, out.RefreshToken
	return nil
}

// authRequest POSTs a fire-and-forget auth request (uniform {"ok":true}).
func (c *Client) authRequest(path string, body any) error {
	_, err := c.authPost(path, body)
	return err
}

// authPost performs the pre-auth POST common to every auth verb: no bearer
// header, and a >= 300 answer surfaces the API's own error body — a 401
// here is the flow's verdict ("invalid or expired code"), not a missing
// INSIDEOUT_TOKEN.
func (c *Client) authPost(path string, body any) (authBody, error) {
	encoded, err := json.Marshal(body)
	if err != nil {
		return authBody{}, fmt.Errorf("apiclient: encode body: %w", err)
	}
	req, err := c.newRequest(http.MethodPost, path, bytes.NewReader(encoded))
	if err != nil {
		return authBody{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.hc.Do(req)
	if err != nil {
		return authBody{}, fmt.Errorf("apiclient: POST %s: %w", path, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return authBody{}, fmt.Errorf("apiclient: read response: %w", err)
	}
	var out authBody
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	if resp.StatusCode >= 300 {
		msg := strings.TrimSpace(out.Error)
		if msg == "" {
			msg = strings.TrimSpace(string(raw))
		}
		return authBody{}, fmt.Errorf("POST %s: %s: %s", path, resp.Status, msg)
	}
	return out, nil
}

type authBody struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	Email        string `json:"email"`
	Error        string `json:"error"`
}

// AuthConfig returns the public signup mode (GET /auth/config) — the
// register form needs to know whether an invitation code is required.
func (c *Client) AuthConfig() (mode string, err error) {
	var out struct {
		SignupMode string `json:"signupMode"`
	}
	if err = c.do(http.MethodGet, "/auth/config", nil, &out); err != nil {
		return "", err
	}
	return out.SignupMode, nil
}

// Admin: invitation management and the signup-mode switch. These ride
// the bearer token of an admin session.

// CreateInvitation mints an email-bound invitation code; the response
// (JSON: invitation + code + signupUrl + mailSent) is the only place the
// code ever appears. mailSent false means nothing was e-mailed.
func (c *Client) CreateInvitation(email string, ttlHours int) (json.RawMessage, error) {
	body := map[string]any{"email": email}
	if ttlHours > 0 {
		body["ttlHours"] = ttlHours
	}
	return c.postRaw("/admin/invitations", body)
}

// ListInvitations returns the invitation ledger (no codes).
func (c *Client) ListInvitations() (json.RawMessage, error) {
	return c.getRaw("/admin/invitations")
}

// RevokeInvitation kills a still-usable invitation.
func (c *Client) RevokeInvitation(id string) error {
	_, err := c.postRaw(apiPath("admin", "invitations", id, "revoke"), map[string]any{})
	return err
}

// SetSignupMode switches registration policy: invitation_only, open, closed.
func (c *Client) SetSignupMode(mode string) (json.RawMessage, error) {
	return c.requestJSON(http.MethodPut, "/admin/settings", map[string]string{"signupMode": mode})
}
