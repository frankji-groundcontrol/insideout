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
// login, register, magic link, password reset, and code login. The refresh
// token stays unused by CLI/MCP sessions, which simply log in again when
// the access token expires (same contract as Login).

// Register creates an account and stores its session token.
func (c *Client) Register(email, password, username string) error {
	return c.authSession("/auth/register", map[string]string{
		"email":    email,
		"password": password,
		"username": username,
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

// authSession POSTs a credential exchange and stores the access token on
// success — the shared shape of Register and the two consume/verify verbs.
func (c *Client) authSession(path string, body any) error {
	out, err := c.authPost(path, body)
	if err != nil {
		return err
	}
	if out.AccessToken == "" {
		return fmt.Errorf("POST %s: no accessToken in response%s", path, errSuffix(out.Error))
	}
	c.token = out.AccessToken
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
	req, err := http.NewRequest(http.MethodPost, c.base+path, bytes.NewReader(encoded))
	if err != nil {
		return authBody{}, fmt.Errorf("apiclient: build request: %w", err)
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
	AccessToken string `json:"accessToken"`
	Email       string `json:"email"`
	Error       string `json:"error"`
}
