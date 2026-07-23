// Package github fetches recent commits for a linked repo so a project's
// progress timeline can sync real development activity. Public repos need no
// credentials (GitHub's unauthenticated rate limit is plenty for a manual
// sync); set GITHUB_TOKEN to raise it. See docs/plans/2026-07-22-idea-to-reality.md.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// Commit is the subset of a GitHub commit the progress timeline records.
type Commit struct {
	SHA     string
	Message string
	Author  string
	Date    time.Time
}

// ParseRepoURL extracts owner/repo from a github.com URL, a bare "owner/repo",
// with or without scheme, .git suffix, or trailing slash.
func ParseRepoURL(raw string) (owner, repo string, err error) {
	s := strings.TrimSpace(raw)
	s = strings.TrimSuffix(s, "/")
	s = strings.TrimSuffix(s, ".git")
	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+3:]
	}
	if i := strings.Index(s, "github.com/"); i >= 0 {
		s = s[i+len("github.com/"):]
	}
	parts := strings.Split(s, "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("github: cannot parse repo %q — want owner/repo or a github.com URL", raw)
	}
	return parts[0], parts[1], nil
}

// FetchCommits returns the most recent commits (newest first) on the repo's
// default branch.
func FetchCommits(ctx context.Context, repoURL string, limit int) ([]Commit, error) {
	owner, repo, err := ParseRepoURL(repoURL)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits?per_page=%d", owner, repo, limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("github: build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "insideout-app") // GitHub rejects UA-less requests
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github: fetch commits: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github: commits for %s/%s returned status %d (private repo, bad name, or rate-limited)", owner, repo, resp.StatusCode)
	}

	var raw []struct {
		SHA    string `json:"sha"`
		Commit struct {
			Message string `json:"message"`
			Author  struct {
				Name string    `json:"name"`
				Date time.Time `json:"date"`
			} `json:"author"`
		} `json:"commit"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("github: decode commits: %w", err)
	}

	out := make([]Commit, 0, len(raw))
	for _, c := range raw {
		msg := c.Commit.Message
		if i := strings.IndexByte(msg, '\n'); i >= 0 {
			msg = msg[:i] // subject line only
		}
		out = append(out, Commit{SHA: c.SHA, Message: strings.TrimSpace(msg), Author: c.Commit.Author.Name, Date: c.Commit.Author.Date})
	}
	return out, nil
}
