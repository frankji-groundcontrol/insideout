package github

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// How long RepoIsPublic remembers an answer (rv4-FR-F177): the server asks
// without a credential from one address whose anonymous quota is 60 requests an
// hour for everyone, and a public repo stays public, so that is kept five
// minutes. A "no" is kept a minute, so a repo just made public is not refused long.
const (
	publicTTL    = 5 * time.Minute
	notPublicTTL = time.Minute
	publicMax    = 512 // answers kept at most; a full map is emptied
)

type publicAnswer struct {
	public bool
	until  time.Time
}

// ponytail: one process-wide map, emptied when full (a loop over distinct names
// costs the others one more question each); two first asks for one repo at once
// both reach GitHub. Add an LRU or singleflight only if that ever shows.
var (
	publicMu      sync.Mutex
	publicAnswers = map[string]publicAnswer{}
)

// RepoIsPublic reports whether anyone can read owner/repo. It asks without a
// credential: GitHub answers an anonymous request 200 for a public repo and 404
// for a private one and a missing one alike, so false means "not visible to
// us", never "missing". A question GitHub cannot answer (limit, refusal, outage)
// is an error, never false, and is not remembered; an answer is, per repo and
// whatever the case of its name.
func RepoIsPublic(ctx context.Context, owner, repo string) (bool, error) {
	if !isRepoSegment(owner) || !isRepoSegment(repo) {
		return false, fmt.Errorf("github: %q/%q is not a repo name", owner, repo)
	}
	key := strings.ToLower(owner + "/" + repo)
	publicMu.Lock()
	a, ok := publicAnswers[key]
	publicMu.Unlock()
	if ok && time.Now().Before(a.until) {
		return a.public, nil
	}
	public, err := askRepoPublic(ctx, owner, repo)
	if err != nil {
		return false, err
	}
	ttl := notPublicTTL
	if public {
		ttl = publicTTL
	}
	publicMu.Lock()
	if len(publicAnswers) >= publicMax {
		clear(publicAnswers)
	}
	publicAnswers[key] = publicAnswer{public: public, until: time.Now().Add(ttl)}
	publicMu.Unlock()
	return public, nil
}

// askRepoPublic is the question RepoIsPublic remembers the answer to.
func askRepoPublic(ctx context.Context, owner, repo string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/repos/%s/%s", apiBaseURL, owner, repo), nil)
	if err != nil {
		return false, fmt.Errorf("github: build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "insideout-app")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return false, fmt.Errorf("github: repo lookup: %w", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	case http.StatusForbidden, http.StatusTooManyRequests:
		return false, limitOrRefusal(resp)
	}
	return false, fmt.Errorf("github: repo lookup returned status %d", resp.StatusCode)
}
