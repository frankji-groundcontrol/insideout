package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ErrUnauthorized is what a request refused with 401 wraps, after the session
// had its one chance to renew itself (SetRenewal).
var ErrUnauthorized = errors.New("unauthorized")

// Renewal is how a Client keeps a saved session alive past its access
// token's short life.
type Renewal struct {
	// Reload returns the session as currently stored: another command may
	// have renewed it since this client read it. ok is false when nothing
	// usable is stored.
	Reload func() (access, refresh string, ok bool)
	// Save stores a rotated pair and reports its own failure: the old
	// refresh token is already dead on the server by then.
	Save func(access, refresh string)

	hint string // what a final 401 says to do instead of the default (SetUnauthorizedHint)
}

// armed reports whether r can renew a session. A Renewal without both hooks is
// only a carrier of the hint.
func (r *Renewal) armed() bool { return r != nil && r.Reload != nil && r.Save != nil }

// SetRenewal arms refresh-on-401 for a client that carries the saved session;
// refresh is that session's refresh token. A client holding any other
// credential (INSIDEOUT_TOKEN, an in-process login) is never armed with the
// saved session: renewing it with that would swap one identity for another. A
// holder of its own refresh token (an in-process login) may arm it with hooks
// over its own storage: apiclient never touches a file, only the hooks do.
func (c *Client) SetRenewal(refresh string, r Renewal) {
	if c.renew != nil {
		r.hint = c.renew.hint // a hint set before stays
	}
	c.refresh, c.renew = refresh, &r
}

// SetUnauthorizedHint replaces the advice a final 401 gives this client, which
// is to run `insideout login`: advice that cannot cure a client whose credential
// is not the saved login (an in-process sign-in in the MCP server). hint is the
// whole advice, "" restores the default. It belongs to this client only:
// AgentScoped does not copy it, so set it on the client AgentScoped returns
// (rv4-FR-F152).
func (c *Client) SetUnauthorizedHint(hint string) {
	if c.renew == nil {
		c.renew = &Renewal{}
	}
	c.renew.hint = hint
}

// renewing serializes renewals in a process. The refresh token is one-time,
// so a burst of calls that all found the access token expired renews once and
// the rest adopt the stored result.
var renewing sync.Mutex

// renewTransport carries every request of a Client. A 401 on an authenticated,
// non-auth request is answered once: renew the session, replay the request
// with the new access token, and return whatever that gets.
type renewTransport struct{ c *Client }

func (t renewTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := http.DefaultTransport.RoundTrip(req)
	c := t.c
	replayable := req.Body == nil || req.Body == http.NoBody || req.GetBody != nil
	if err != nil || resp.StatusCode != http.StatusUnauthorized || !c.renew.armed() || !replayable ||
		req.Header.Get("Authorization") == "" || strings.Contains(req.URL.Path, "/auth/") {
		return resp, err
	}
	access, err := c.renewSession(req.Context())
	if err != nil {
		resp.Body.Close()
		return nil, err
	}
	if access == "" {
		return resp, nil // the refusal stands
	}
	resp.Body.Close()
	retry := req.Clone(req.Context())
	retry.Header.Set("Authorization", "Bearer "+access)
	if req.GetBody != nil {
		if retry.Body, err = req.GetBody(); err != nil {
			return nil, err
		}
	}
	return http.DefaultTransport.RoundTrip(retry)
}

// renewSession returns the access token to replay a refused request with, or
// "" when the refusal stands. An error means the renewal itself did not happen:
// the request never got through (network), or the server declined to answer it
// (429, 5xx). Neither is a verdict on the session, so the client stays armed.
func (c *Client) renewSession(ctx context.Context) (string, error) {
	renewing.Lock()
	defer renewing.Unlock()
	if a, r, ok := c.renew.Reload(); ok && r != c.refresh {
		c.token, c.refresh = a, r // another command or call renewed it already
		return a, nil
	}
	used := c.refresh
	access, refresh, status, err := c.postRefresh(ctx, used)
	switch {
	case err != nil:
		return "", err
	case status == http.StatusOK:
		c.token, c.refresh = access, refresh
		c.renew.Save(access, refresh)
		return access, nil
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		// The one-time token is gone: another command won it, and its rotated
		// pair reaches the store a moment after the server answered it.
		if a, r, ok := c.awaitRotation(ctx, used); ok {
			c.token, c.refresh = a, r
			return a, nil
		}
		c.renew.Reload, c.renew.Save = nil, nil // rejected for good: later requests of this client must not retry it
		return "", nil
	}
	// The per-IP throttle (429), a deploy in progress (503), any 5xx: nothing
	// about the session. Say so, rather than let the original 401 read as the
	// verdict (rv4-FR-F151, rv4-FR-F164). The client stays armed.
	return "", fmt.Errorf("apiclient: POST /auth/refresh: could not renew the session (HTTP %d): "+
		"try again; run `insideout login` only if this persists", status)
}

// renewalGrace is how long a renewal that lost the race waits for the winner's
// pair to be stored before it takes the refusal for the session's end. A var
// so tests can shorten it.
// ponytail: polls the store instead of locking across processes. The server now
// revokes a token family on reuse, but only after refreshReuseGrace (10 s,
// internal/api/auth.go), which covers this wait with room to spare; if that
// grace is ever shortened below this wait, or bursts still lose, serialize
// renewals with a lock file beside the config.
var renewalGrace = 500 * time.Millisecond

// awaitRotation polls the store until it holds a pair other than used, or the
// grace is over.
func (c *Client) awaitRotation(ctx context.Context, used string) (access, refresh string, ok bool) {
	for deadline := time.Now().Add(renewalGrace); ; {
		if a, r, ok := c.renew.Reload(); ok && r != used {
			return a, r, true
		}
		if !time.Now().Before(deadline) {
			return "", "", false
		}
		select {
		case <-ctx.Done():
			return "", "", false
		case <-time.After(25 * time.Millisecond):
		}
	}
}

// renewalDrain is how long a renewal in flight may outlive the cancellation of
// the call that started it. The server rotates the refresh token as it reads the
// request, so dropping the request before its answer is in hand leaves the
// stored login holding a spent token and the user signed out (rv4-FR-F66). A var
// so tests can change it.
// ponytail: a ceiling, not a guarantee: an answer slower than this is still lost.
// Only a two-step rotation on the server (confirm before the old token dies)
// removes it.
var renewalDrain = 750 * time.Millisecond

// WaitForRenewal returns when no session renewal is in flight in this process,
// or after 2*renewalDrain (a cancelled renewal ends within renewalDrain, the rest
// is its save). A process about to exit calls it so it does not end between the
// server rotating the refresh token and the new pair being stored.
func WaitForRenewal() {
	for end := time.Now().Add(2 * renewalDrain); time.Now().Before(end); time.Sleep(10 * time.Millisecond) {
		if renewing.TryLock() {
			renewing.Unlock()
			return
		}
	}
}

// postRefresh trades a refresh token for a new pair (POST /auth/refresh takes
// it in the body when there is no cookie). status is 0 when err is set. A caller
// that is cancelled mid-way waits renewalDrain for the answer instead of
// dropping it; one that is cancelled before the request is sent sends nothing,
// so no token is spent for a caller that has gone.
func (c *Client) postRefresh(ctx context.Context, refresh string) (access, next string, status int, err error) {
	if err := ctx.Err(); err != nil {
		return "", "", 0, fmt.Errorf("apiclient: POST /auth/refresh: %w", err)
	}
	dl, ok := ctx.Deadline()
	if !ok {
		dl = time.Now().Add(deadlineFor(http.MethodPost, "/auth/refresh"))
	}
	rctx, cancel := context.WithDeadline(context.WithoutCancel(ctx), dl)
	defer cancel()
	drain := renewalDrain // read here: the callback below runs on a goroutine of its own
	stop := context.AfterFunc(ctx, func() { time.AfterFunc(drain, cancel) })
	defer stop()
	body, _ := json.Marshal(map[string]string{"refreshToken": refresh})
	req, err := http.NewRequestWithContext(rctx, http.MethodPost, c.base+"/auth/refresh", bytes.NewReader(body))
	if err != nil {
		return "", "", 0, fmt.Errorf("apiclient: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return "", "", 0, fmt.Errorf("apiclient: POST /auth/refresh: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return "", "", resp.StatusCode, nil
	}
	var out authBody
	if json.Unmarshal(raw, &out) != nil || out.AccessToken == "" || out.RefreshToken == "" {
		return "", "", 0, fmt.Errorf("apiclient: POST /auth/refresh: no tokens in response")
	}
	return out.AccessToken, out.RefreshToken, http.StatusOK, nil
}

// unauthorized is the final answer to a 401: what it means for this client's
// credential, and what to do about it.
func (c *Client) unauthorized(what string) error {
	if c.renew != nil && c.renew.hint != "" {
		return fmt.Errorf("apiclient: %s: %w: %s", what, ErrUnauthorized, c.renew.hint)
	}
	if c.token == "" {
		return fmt.Errorf("apiclient: %s: %w: not signed in; run `insideout login`, or set INSIDEOUT_TOKEN", what, ErrUnauthorized)
	}
	return fmt.Errorf("apiclient: %s: %w: the session expired or was revoked; run `insideout login` to sign in again "+
		"(an INSIDEOUT_TOKEN in the environment overrides the saved login: replace or unset it)", what, ErrUnauthorized)
}
