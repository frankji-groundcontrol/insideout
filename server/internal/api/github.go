package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/frankji-groundcontrol/insideout/server/internal/github"
	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

func (s *Server) registerGithubRoutes(mux *http.ServeMux) {
	mux.HandleFunc("PUT /api/v1/projects/{pid}/repo", s.requireAuth(s.handleSetProjectRepo))
	mux.HandleFunc("POST /api/v1/projects/{pid}/sync-github", s.requireAuth(s.handleSyncGithub))
}

type setRepoRequest struct {
	RepoURL string `json:"repoUrl"`
}

func (s *Server) handleSetProjectRepo(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	pid, ok := pathUUID(w, r, "pid")
	if !ok {
		return
	}
	var req setRepoRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	req.RepoURL = strings.TrimSpace(req.RepoURL)
	// Allow clearing with an empty value; otherwise it must parse as a repo.
	if req.RepoURL != "" {
		if _, _, err := github.ParseRepoURL(req.RepoURL); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error(), "", nil)
			return
		}
	}

	p, err := s.store.SetProjectRepo(r.Context(), userID, pid, req.RepoURL)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "project not found", "", nil)
		return
	}
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "only the project owner or a workspace admin can link a repo", "", nil)
		return
	}
	if err != nil {
		s.log.Error("set project repo", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, projectResponse(store.ProjectWithLatest{Project: *p}))
}

// handleSyncGithub pulls recent commits from the project's linked repo into
// its progress timeline, advancing a per-project cursor so repeat syncs only
// add what's new. Owner/admin (it writes the projects sync cursor).
func (s *Server) handleSyncGithub(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	pid, ok := pathUUID(w, r, "pid")
	if !ok {
		return
	}

	repoURL, lastSHA, err := s.store.ProjectRepoSync(r.Context(), userID, pid)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "project not found", "", nil)
		return
	}
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "only the project owner or a workspace admin can sync", "", nil)
		return
	}
	if err != nil {
		s.log.Error("read repo sync state", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	if repoURL == "" {
		httpx.WriteError(w, http.StatusBadRequest, "no GitHub repo linked to this project yet", "", nil)
		return
	}

	// Walk pagination back to the last-synced cursor so a repo that landed
	// more than one page of commits since the last sync doesn't silently drop
	// the ones past the first page. fresh is newest-first.
	fresh, err := github.FetchCommitsSince(r.Context(), repoURL, lastSHA, 20, 5)
	if err != nil {
		var rle *github.RateLimitError
		switch {
		case errors.As(err, &rle):
			httpx.WriteError(w, http.StatusTooManyRequests, "GitHub rate limit hit — try again shortly", "GITHUB_RATE_LIMITED", map[string]any{"retry_after_seconds": rle.RetryAfterSeconds})
		case errors.Is(err, github.ErrRepoNotFound):
			httpx.WriteError(w, http.StatusNotFound, "GitHub repository not found — private repo, or check the name", "GITHUB_NOT_FOUND", nil)
		default:
			// Log the transport detail but return a generic message, so upstream
			// internals (URLs, headers) never reach the client (F12).
			s.log.Error("github fetch", "error", err)
			httpx.WriteError(w, http.StatusBadGateway, "GitHub sync failed", "GITHUB_UPSTREAM", nil)
		}
		return
	}
	if len(fresh) == 0 {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"added": 0, "repoUrl": repoURL})
		return
	}

	// Build the timeline entries oldest-first (so they read chronologically);
	// the newest SHA becomes the cursor.
	contents := make([]string, 0, len(fresh))
	for i := len(fresh) - 1; i >= 0; i-- {
		c := fresh[i]
		sha := c.SHA
		if len(sha) > 7 {
			sha = sha[:7]
		}
		contents = append(contents, fmt.Sprintf("[github %s] %s — %s", sha, c.Message, c.Author))
	}

	// One transaction: the batch and the cursor advance land together, so a
	// failure can't leave commits written with the cursor still behind them.
	added, err := s.store.SyncRepoCommits(r.Context(), userID, pid, contents, fresh[0].SHA)
	if err != nil {
		s.log.Error("sync repo commits", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"added": added, "repoUrl": repoURL})
}
