package apiclient

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// The agent-token exchange (docs/architecture/agent-token-scope.md): the MCP
// binary trades the user's login for an access token that carries the agent scope
// and runs its tools on that token alone. The server refuses that token on every
// route a human must own (Commit, the proposal decision, the instance switches,
// deletes) and on the exchange itself, so it can never mint a wider one.

// scopeAgent is the "scope" claim of an agent-scope access token.
const scopeAgent = "agent"

// TokenClaims are the claims of a JWT access token as its holder can read them
// without the signing secret. Reading them verifies nothing, because the server
// checks every request: a client only uses them to choose how to carry a token.
type TokenClaims struct {
	Subject   string
	Scope     string // "" is a full session; "agent" is the agent scope
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// ParseTokenClaims reads the claims of a JWT. ok is false when raw is not one: a
// token that is not a JWT cannot carry a scope, and no InsideOut server accepts it.
func ParseTokenClaims(raw string) (TokenClaims, bool) {
	parts := strings.Split(raw, ".")
	if len(parts) != 3 || !jsonPart(parts[0], &map[string]any{}) {
		return TokenClaims{}, false
	}
	var c struct {
		Sub   string  `json:"sub"`
		Scope string  `json:"scope"`
		Iat   float64 `json:"iat"`
		Exp   float64 `json:"exp"`
	}
	if !jsonPart(parts[1], &c) {
		return TokenClaims{}, false
	}
	return TokenClaims{Subject: c.Sub, Scope: c.Scope, IssuedAt: unixTime(c.Iat), ExpiresAt: unixTime(c.Exp)}, true
}

// jsonPart decodes one base64url part of a JWT as JSON into v.
func jsonPart(part string, v any) bool {
	b, err := base64.RawURLEncoding.DecodeString(part)
	return err == nil && json.Unmarshal(b, v) == nil
}

func unixTime(sec float64) time.Time {
	if sec == 0 {
		return time.Time{}
	}
	return time.Unix(int64(sec), 0)
}

// AgentScoped returns the client a tool call should use for login. When login
// carries a full-scope JWT, the result carries an agent-scope token traded for it
// instead (minted once, then again about a third of its life from expiry), never
// the login itself. A failed trade is a client that fails every call with the
// reason, so a caller can not fall back to the full session. A token that already
// has a scope, and one that is not a JWT, are sent as they are: the server decides.
func AgentScoped(login *Client) *Client {
	claims, ok := ParseTokenClaims(login.token)
	if !ok || claims.Scope != "" {
		return login
	}
	token, err := heldAgentToken(login, claims.Subject)
	if err != nil {
		return Failing(err)
	}
	scoped := New(login.base)
	scoped.token = token // never renews: this token is not the saved session
	scoped.ctx = login.ctx
	return scoped
}

// held is the agent-scope token minted for the login in use.
var held struct {
	sync.Mutex
	key     string // the server and user the token was minted for; "" holds none
	token   string
	renewAt time.Time // mint again from here: about a third of the token's life is left
}

// ForgetAgentToken drops the held token. A process calls it when its login
// changes (a sign-in or a sign-out).
func ForgetAgentToken() {
	held.Lock()
	held.key = ""
	held.Unlock()
}

// heldAgentToken returns the held token for login's server and user, or mints one.
// The lock is held across the trade, so a burst of calls mints once.
// ponytail: one slot (a process serves one server and one user at a time); a
// client clock more than a token life off the server's mints on every call, or
// rides one expired token until the 401.
func heldAgentToken(login *Client, subject string) (string, error) {
	held.Lock()
	defer held.Unlock()
	key := login.base + "\n" + subject
	if held.key == key && time.Now().Before(held.renewAt) {
		return held.token, nil
	}
	token, claims, err := login.agentToken()
	if err != nil {
		held.key = ""
		return "", err
	}
	issued := claims.IssuedAt
	if issued.IsZero() {
		issued = time.Now()
	}
	held.key, held.token = key, token
	held.renewAt = claims.ExpiresAt.Add(-claims.ExpiresAt.Sub(issued) / 3)
	return token, nil
}

// agentToken trades this client's login for an agent-scope access token
// (POST /agent/token) and returns it with its claims. It only works from a full
// session; the server refuses the trade to an agent token. An answer that is not
// an agent-scope token is an error, so a caller can never mistake a full session
// for one.
func (c *Client) agentToken() (string, TokenClaims, error) {
	var out struct {
		AccessToken string `json:"accessToken"`
	}
	if err := c.do(http.MethodPost, "/agent/token", map[string]any{}, &out); err != nil {
		if strings.Contains(err.Error(), "/agent/token: 404") {
			err = fmt.Errorf("%w (this server predates agent tokens: update the server before the MCP binary)", err)
		}
		return "", TokenClaims{}, err
	}
	claims, ok := ParseTokenClaims(out.AccessToken)
	if !ok || claims.Scope != scopeAgent || claims.ExpiresAt.IsZero() {
		return "", TokenClaims{}, fmt.Errorf("apiclient: POST /agent/token: the answer is not an agent-scope access token")
	}
	return out.AccessToken, claims, nil
}
