package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/frankji-groundcontrol/insideout/server/internal/github"
	"github.com/frankji-groundcontrol/insideout/server/internal/guide"
	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

func (s *Server) registerGithubWebhookRoute(mux *http.ServeMux) {
	// No requireAuth: deliveries are authenticated by the HMAC signature,
	// not by a user session.
	mux.HandleFunc("POST /api/v1/hooks/github", s.handleGithubWebhook)
}

// handleGithubWebhook accepts GitHub App deliveries
// (https://insideout.yalotein.net/api/v1/hooks/github). Signature first,
// always; events other than push/pull_request are acknowledged and
// ignored. push/pull_request resolve the repo's projects and re-run the
// normal per-project sync as each project owner, so every write stays
// inside RLS.
func (s *Server) handleGithubWebhook(w http.ResponseWriter, r *http.Request) {
	if s.cfg.GithubWebhookSecret == "" {
		httpx.WriteError(w, http.StatusServiceUnavailable, "webhook not configured", "", nil)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 8<<20))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "read body", "", nil)
		return
	}
	if !github.VerifyWebhookSignature(s.cfg.GithubWebhookSecret, body, r.Header.Get("X-Hub-Signature-256")) {
		httpx.WriteError(w, http.StatusUnauthorized, "bad signature", "", nil)
		return
	}

	event := r.Header.Get("X-GitHub-Event")
	switch event {
	case "ping":
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true, "event": event})
	case "push", "pull_request":
		name, err := github.WebhookRepository(body)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "unparseable repository", "", nil)
			return
		}
		targets, err := s.store.ProjectsByRepo(r.Context(), github.RepoURLFor(name))
		if err != nil {
			s.log.Error("webhook: projects by repo", "repo", name, "error", err)
			httpx.WriteError(w, http.StatusInternalServerError, "lookup failed", "", nil)
			return
		}
		commits := 0
		for _, po := range targets {
			added, _, err := s.syncGithubProject(r.Context(), po.OwnerID, po.ProjectID)
			if err != nil {
				// One project's failure must not block the others; GitHub
				// retries the whole delivery only on 5xx.
				s.log.Error("webhook: sync", "project", po.ProjectID, "error", err)
				continue
			}
			commits += added
		}
		matched := s.attachGuideEvidence(r.Context(), event, name, body, targets)
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"event": event, "repo": name, "projects": len(targets), "commits": commits, "evidence": matched,
		})
	default:
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"ignored": event})
	}
}

// attachGuideEvidence loads the repo's insideout.yaml (installation
// token when the app credentials are configured and the delivery names
// an installation; unauthenticated otherwise — public repos), matches
// the event, and appends leaf evidence to the resolved projects.
func (s *Server) attachGuideEvidence(ctx context.Context, event, repoName string, body []byte, targets []store.ProjectOwner) int {
	if len(targets) == 0 {
		return 0
	}
	var ev struct {
		Installation struct {
			ID int64 `json:"id"`
		} `json:"installation"`
		Ref     string `json:"ref"` // push: refs/heads/x
		Commits []struct {
			Added    []string `json:"added"`
			Modified []string `json:"modified"`
			Removed  []string `json:"removed"`
			SHA      string   `json:"id"`
			Message  string   `json:"message"`
		} `json:"commits"`
		PullRequest struct {
			Head struct {
				Ref string `json:"ref"`
			} `json:"head"`
			Labels []struct {
				Name string `json:"name"`
			} `json:"labels"`
			HTMLURL  string `json:"html_url"`
			Title    string `json:"title"`
			Merged   bool   `json:"merged"`
			MergedAt string `json:"merged_at"`
		} `json:"pull_request"`
	}
	if err := json.Unmarshal(body, &ev); err != nil {
		return 0
	}

	token := ""
	if s.ghTokens != nil && ev.Installation.ID != 0 {
		if t, err := s.ghTokens.Token(ctx, ev.Installation.ID); err != nil {
			s.log.Warn("webhook: installation token", "error", err)
		} else {
			token = t
		}
	}
	owner, repo, _ := strings.Cut(repoName, "/")
	branch := strings.TrimPrefix(ev.Ref, "refs/heads/")
	kind, detail, labels, paths, source := "activity", "", []string(nil), []string(nil), ""
	if event == "push" {
		if branch == "" {
			return 0
		}
		for _, c := range ev.Commits {
			paths = append(paths, c.Added...)
			paths = append(paths, c.Modified...)
			paths = append(paths, c.Removed...)
		}
		if len(paths) > 200 {
			paths = paths[:200]
		}
		if len(ev.Commits) > 0 {
			c := ev.Commits[len(ev.Commits)-1]
			sha := c.SHA
			if len(sha) > 7 {
				sha = sha[:7]
			}
			detail = fmt.Sprintf("push %s: %s", sha, firstLine(c.Message))
			source = fmt.Sprintf("https://github.com/%s/commit/%s", repoName, c.SHA)
		} else {
			detail = "push to " + branch
		}
	} else { // pull_request
		branch = ev.PullRequest.Head.Ref
		for _, l := range ev.PullRequest.Labels {
			labels = append(labels, l.Name)
		}
		switch {
		case ev.PullRequest.Merged:
			kind = "implementation"
		default:
			kind = "review"
		}
		detail = fmt.Sprintf("PR %s", firstLine(ev.PullRequest.Title))
		source = ev.PullRequest.HTMLURL
	}

	raw, err := github.FetchGuideFile(ctx, token, owner, repo, branch)
	if err != nil {
		if !errors.Is(err, github.ErrGuideNotFound) {
			s.log.Warn("webhook: guide fetch", "repo", repoName, "error", err)
		}
		return 0
	}
	nodes, err := guide.Parse(raw)
	if err != nil {
		s.log.Warn("webhook: guide parse", "repo", repoName, "error", err)
		return 0
	}
	// leaf check per target project: a matched id must be a real, parent-
	// less node of that project.
	leafFn := func(po store.ProjectOwner) func(string) bool {
		return func(id string) bool {
			nodeID, err := uuid.Parse(id)
			if err != nil {
				return false
			}
			return s.store.IsRoadmapLeaf(ctx, po.OwnerID, po.ProjectID, nodeID)
		}
	}
	added := 0
	for _, po := range targets {
		for _, id := range guide.Match(nodes, branch, labels, paths, leafFn(po)) {
			nodeID, err := uuid.Parse(id)
			if err != nil {
				continue
			}
			if err := s.store.AddRoadmapEvidence(ctx, po.OwnerID, nodeID, kind, detail, source); err != nil {
				s.log.Warn("webhook: evidence", "node", id, "error", err)
				continue
			}
			added++
		}
	}
	return added
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	if len(s) > 120 {
		return s[:120]
	}
	return s
}

// syncGithubProject is the per-project sync core shared by the
// user-triggered route and the webhook: pull commits since the stored
// cursor and append them to the project timeline in one transaction.
// userID is the acting owner (the requesting user, or the owner the
// webhook resolved).
func (s *Server) syncGithubProject(ctx context.Context, userID, projectID uuid.UUID) (added int, repoURL string, err error) {
	repoURL, lastSHA, err := s.store.ProjectRepoSync(ctx, userID, projectID)
	if err != nil {
		return 0, "", err
	}
	if repoURL == "" {
		return 0, "", errNoRepoLinked
	}
	fresh, err := github.FetchCommitsSince(ctx, repoURL, lastSHA, 20, 5)
	if err != nil {
		return 0, repoURL, err
	}
	if len(fresh) == 0 {
		return 0, repoURL, nil
	}
	contents := make([]string, 0, len(fresh))
	for i := len(fresh) - 1; i >= 0; i-- {
		c := fresh[i]
		sha := c.SHA
		if len(sha) > 7 {
			sha = sha[:7]
		}
		contents = append(contents, fmt.Sprintf("[github %s] %s — %s", sha, c.Message, c.Author))
	}
	added, err = s.store.SyncRepoCommits(ctx, userID, projectID, contents, fresh[0].SHA)
	if err != nil {
		return 0, repoURL, err
	}
	return added, repoURL, nil
}

var errNoRepoLinked = errors.New("no repo linked")
