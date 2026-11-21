package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Scope is what an access token may do (docs/architecture/agent-token-scope.md).
// It is the signed "scope" claim. ScopeFull is the zero value because it is the
// claim being absent: every token issued before scopes existed, every browser
// cookie and every saved CLI login is a full session. A scope this server does
// not know is rejected outright, so a scope added later never degrades to full
// power on an older server.
type Scope string

const (
	ScopeFull  Scope = ""
	ScopeAgent Scope = "agent"
)

type TokenIssuer struct {
	secret    []byte
	accessTTL time.Duration
}

func NewTokenIssuer(secret string, accessTTL time.Duration) *TokenIssuer {
	return &TokenIssuer{secret: []byte(secret), accessTTL: accessTTL}
}

// accessClaims is the payload of an access token: the registered claims plus
// the scope. A full session carries no scope claim at all.
type accessClaims struct {
	jwt.RegisteredClaims
	Scope Scope `json:"scope,omitempty"`
	// Root confines an agent token to one roadmap node and the nodes below
	// it (rv5-TREEB-05). Only an agent token carries one.
	Root string `json:"root,omitempty"`
}

func (t *TokenIssuer) mint(userID uuid.UUID, scope Scope, root *uuid.UUID) (string, time.Time, error) {
	now := time.Now()
	expires := now.Add(t.accessTTL)
	claims := accessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expires),
		},
		Scope: scope,
	}
	if root != nil {
		claims.Root = root.String()
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("auth: sign access token: %w", err)
	}
	return signed, expires, nil
}

// MintAccessToken issues a short-lived HS256 JWT with sub/exp/iat claims: a
// full session.
func (t *TokenIssuer) MintAccessToken(userID uuid.UUID) (string, error) {
	token, _, err := t.mint(userID, ScopeFull, nil)
	return token, err
}

// MintAgentToken issues an agent-scope access token for the user, with the same
// lifetime as a full one, and the instant it expires. Only the server routes
// that the MCP tools call accept it (api.agentRoutePolicy). A non-nil root
// confines its node writes to that node's subtree.
func (t *TokenIssuer) MintAgentToken(userID uuid.UUID, root *uuid.UUID) (string, time.Time, error) {
	return t.mint(userID, ScopeAgent, root)
}

// Verified is what a valid access token says.
type Verified struct {
	UserID uuid.UUID
	Scope  Scope
	Root   *uuid.UUID // the agent token's root node, nil when unconfined
}

// VerifyToken parses and validates an access token, returning the subject user
// id and the token's scope. A scope other than full or agent is an error.
func (t *TokenIssuer) VerifyToken(raw string) (uuid.UUID, Scope, error) {
	v, err := t.Verify(raw)
	if err != nil {
		return uuid.UUID{}, ScopeFull, err
	}
	return v.UserID, v.Scope, nil
}

// Verify is VerifyToken with the root claim. A root on a token that is not an
// agent token, or one that is not a node id, is an error.
func (t *TokenIssuer) Verify(raw string) (Verified, error) {
	claims := &accessClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(tok *jwt.Token) (interface{}, error) {
		if _, ok := tok.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", tok.Header["alg"])
		}
		return t.secret, nil
	})
	if err != nil {
		return Verified{}, fmt.Errorf("auth: parse access token: %w", err)
	}
	if !token.Valid {
		return Verified{}, fmt.Errorf("auth: invalid access token")
	}
	if claims.Scope != ScopeFull && claims.Scope != ScopeAgent {
		return Verified{}, fmt.Errorf("auth: unknown token scope %q", claims.Scope)
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return Verified{}, fmt.Errorf("auth: invalid subject claim: %w", err)
	}
	v := Verified{UserID: userID, Scope: claims.Scope}
	if claims.Root != "" {
		root, err := uuid.Parse(claims.Root)
		if err != nil || claims.Scope != ScopeAgent {
			return Verified{}, fmt.Errorf("auth: a root claim needs an agent token and a node id")
		}
		v.Root = &root
	}
	return v, nil
}

// VerifyAccessToken parses and validates a full-session access token, returning
// the subject user id. A scoped token is refused here, so no caller can take an
// agent token for a full session; use VerifyToken to see the scope.
func (t *TokenIssuer) VerifyAccessToken(raw string) (uuid.UUID, error) {
	userID, scope, err := t.VerifyToken(raw)
	if err != nil {
		return uuid.UUID{}, err
	}
	if scope != ScopeFull {
		return uuid.UUID{}, fmt.Errorf("auth: a %q-scope token is not a full session", scope)
	}
	return userID, nil
}
