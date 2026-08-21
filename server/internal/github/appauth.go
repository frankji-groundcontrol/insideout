package github

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// LoadPrivateKey accepts the GitHub App .pem either \n-escaped in one
// line (env-var form) or via a file path (the _FILE variant; wins when
// set). PKCS#1 ("BEGIN RSA PRIVATE KEY") is what GitHub downloads.
func LoadPrivateKey(escaped, filePath string) (*rsa.PrivateKey, error) {
	var data []byte
	if filePath != "" {
		b, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("github: read private key file: %w", err)
		}
		data = b
	} else {
		data = []byte(strings.ReplaceAll(escaped, `\n`, "\n"))
	}
	block, _ := pem.Decode(bytes.TrimSpace(data))
	if block == nil {
		return nil, fmt.Errorf("github: private key is not PEM")
	}
	if k, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return k, nil
	}
	k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("github: parse private key: %w", err)
	}
	rk, ok := k.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("github: private key is %T, want RSA", k)
	}
	return rk, nil
}

// MintAppJWT produces the RS256 GitHub App JWT (iss = app id, 9-minute
// window) used to mint installation tokens.
func MintAppJWT(appID string, key *rsa.PrivateKey, now time.Time) (string, error) {
	header := b64(map[string]string{"alg": "RS256", "typ": "JWT"})
	claims := b64(map[string]any{
		"iat": now.Add(-time.Minute).Unix(),
		"exp": now.Add(9 * time.Minute).Unix(),
		"iss": appID,
	})
	signing := []byte(header + "." + claims)
	sum := sha256.Sum256(signing)
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		return "", fmt.Errorf("github: sign jwt: %w", err)
	}
	return string(signing) + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

// InstallationTokens mints and caches short-lived installation access
// tokens (POST /app/installations/{id}/access_tokens), one per
// installation, refreshed a minute before expiry.
type InstallationTokens struct {
	appID string
	key   *rsa.PrivateKey
	hc    *http.Client

	mu     sync.Mutex
	tokens map[int64]cachedToken
}

type cachedToken struct {
	token  string
	expiry time.Time
}

func NewInstallationTokens(appID string, key *rsa.PrivateKey) *InstallationTokens {
	return &InstallationTokens{
		appID:  appID,
		key:    key,
		hc:     &http.Client{Timeout: 15 * time.Second},
		tokens: map[int64]cachedToken{},
	}
}

// Token returns a live access token for the installation.
func (t *InstallationTokens) Token(ctx context.Context, installationID int64) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if c, ok := t.tokens[installationID]; ok && time.Until(c.expiry) > time.Minute {
		return c.token, nil
	}
	jwt, err := MintAppJWT(t.appID, t.key, time.Now())
	if err != nil {
		return "", err
	}
	url := fmt.Sprintf("%s/app/installations/%d/access_tokens", apiBaseURL, installationID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := t.hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("github: access token: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("github: access token: %s", resp.Status)
	}
	var out struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil || out.Token == "" {
		return "", fmt.Errorf("github: access token: decode")
	}
	t.tokens[installationID] = cachedToken{token: out.Token, expiry: out.ExpiresAt}
	return out.Token, nil
}

// FetchGuideFile downloads insideout.yaml from the repo at a ref.
// token may be empty (public repos work unauthenticated).
func FetchGuideFile(ctx context.Context, token, owner, repo, ref string) ([]byte, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/insideout.yaml?ref=%s", apiBaseURL, owner, repo, ref)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.raw+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("github: guide fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrGuideNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github: guide fetch: %s", resp.Status)
	}
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return nil, err
	}
	if buf.Len() > 1<<20 {
		return nil, fmt.Errorf("github: guide too large")
	}
	return buf.Bytes(), nil
}

// ErrGuideNotFound means the repo has no insideout.yaml (yet).
var ErrGuideNotFound = fmt.Errorf("guide not found")

func b64(v any) string {
	raw, _ := json.Marshal(v)
	return base64.RawURLEncoding.EncodeToString(raw)
}
