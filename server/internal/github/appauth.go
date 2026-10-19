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
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

// ErrNotInstalled means the GitHub App has no installation covering the repo.
// GitHub answers the same 404 for a private repo and for one that does not exist.
var ErrNotInstalled = errors.New("github: app is not installed on this repo")

// InstallationForRepo asks GitHub which installation of this App covers
// owner/repo (GET /repos/{o}/{r}/installation, authenticated by the App JWT).
// The id comes from GitHub for the repo itself, never from the caller, so a
// token minted from it reads only what that installation was granted.
// ErrNotInstalled when none does.
func (t *InstallationTokens) InstallationForRepo(ctx context.Context, owner, repo string) (int64, error) {
	// The App JWT rides on this request: nothing but a plain owner/repo pair
	// may shape its path.
	if !isRepoSegment(owner) || !isRepoSegment(repo) {
		return 0, fmt.Errorf("github: %q/%q is not a repo name", owner, repo)
	}
	jwt, err := MintAppJWT(t.appID, t.key, time.Now())
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/repos/%s/%s/installation", apiBaseURL, owner, repo), nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := t.hc.Do(req)
	if err != nil {
		return 0, fmt.Errorf("github: installation lookup: %w", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return 0, ErrNotInstalled
	case http.StatusForbidden, http.StatusTooManyRequests:
		return 0, &RateLimitError{RetryAfterSeconds: retryAfterSeconds(resp.Header)}
	default:
		return 0, fmt.Errorf("github: installation lookup: %s", resp.Status)
	}
	var out struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil || out.ID == 0 {
		return 0, fmt.Errorf("github: installation lookup: decode")
	}
	return out.ID, nil
}

// RepoToken is the installation token that can read owner/repo, or "" when the
// App is not installed on it (a public repo still reads anonymously).
func (t *InstallationTokens) RepoToken(ctx context.Context, owner, repo string) (string, error) {
	id, err := t.InstallationForRepo(ctx, owner, repo)
	if errors.Is(err, ErrNotInstalled) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return t.Token(ctx, id)
}

// RepoVisible reports whether this server can read owner/repo at all: the App
// is installed on it, or it is public. GitHub answers the same 404 for a
// private repo and a missing one, so false means "not to us", and a lookup
// that fails is an error, never false. Binding a repo to a project asks this
// first, because the link is what later hands the repo's installation token to
// the project's syncs (rv4-COMP-05).
func (t *InstallationTokens) RepoVisible(ctx context.Context, owner, repo string) (bool, error) {
	_, err := t.InstallationForRepo(ctx, owner, repo)
	if err == nil {
		return true, nil
	}
	if !errors.Is(err, ErrNotInstalled) {
		return false, err
	}
	return RepoIsPublic(ctx, owner, repo)
}

// FetchGuideFile downloads insideout.yaml from the repo at a ref.
// token may be empty (public repos work unauthenticated).
func FetchGuideFile(ctx context.Context, token, owner, repo, ref string) ([]byte, error) {
	// PathEscape leaves "." and ".." alone and an empty segment collapses into
	// "//": either reshapes the path of a request that carries the installation
	// token, and the names come from a delivery payload (SEC-06). Other odd
	// characters stay escaped below.
	if strings.Trim(owner, ".") == "" || strings.Trim(repo, ".") == "" {
		return nil, fmt.Errorf("github: %q/%q is not a repo name", owner, repo)
	}
	// Escape what gets interpolated: a raw '#' in a segment would start a
	// fragment and a '&' in the ref would inject query parameters (SEC-06).
	url := fmt.Sprintf("%s/repos/%s/%s/contents/insideout.yaml?ref=%s",
		apiBaseURL, url.PathEscape(owner), url.PathEscape(repo), url.QueryEscape(ref))
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
	// Read at most cap+1 bytes so an oversized guide is detected while it
	// streams and the rest of the body is never buffered (SEC-02, same
	// class as maxCommitPageBytes F19).
	const maxGuideFileBytes = 1 << 20
	buf, err := io.ReadAll(io.LimitReader(resp.Body, maxGuideFileBytes+1))
	if err != nil {
		return nil, err
	}
	if len(buf) > maxGuideFileBytes {
		return nil, fmt.Errorf("github: guide too large")
	}
	return buf, nil
}

// ErrGuideNotFound means the repo has no insideout.yaml (yet).
var ErrGuideNotFound = fmt.Errorf("guide not found")

func b64(v any) string {
	raw, _ := json.Marshal(v)
	return base64.RawURLEncoding.EncodeToString(raw)
}
