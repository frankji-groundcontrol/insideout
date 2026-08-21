package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/frankji-groundcontrol/insideout/server/internal/github"
	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
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
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"event": event, "repo": name, "projects": len(targets), "commits": commits,
		})
	default:
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"ignored": event})
	}
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
