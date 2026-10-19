package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/frankji-groundcontrol/insideout/server/internal/github"
	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

func (s *Server) registerGithubRoutes(mux *http.ServeMux) {
	mux.HandleFunc("PUT /api/v1/projects/{pid}/repo", s.requireAuth(s.handleSetProjectRepo))
	mux.HandleFunc("POST /api/v1/projects/{pid}/sync-github", s.requireAuth(s.handleSyncGithub))
	s.registerGithubWebhookRoute(mux)
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
	// Allow clearing with an empty value; otherwise it must parse as a repo,
	// and what is stored is the canonical URL, not the raw input: the sync and
	// the webhook lookup both read that exact string (rv4-SEC-06).
	if req.RepoURL != "" {
		owner, repo, err := github.ParseRepoURL(req.RepoURL)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error(), "", nil)
			return
		}
		req.RepoURL = github.CanonicalRepoURL(owner, repo)
		// Who may link comes before what GitHub shows: a caller who may not link
		// this project must not be able to spend the App's rate limit or learn
		// which repos it is installed on (rv4-COMP-05).
		if _, _, err := s.store.ProjectRepoSync(r.Context(), userID, pid); err != nil {
			s.writeRepoStoreError(w, err)
			return
		}
		if err := s.requireRepoVisible(r.Context(), owner, repo); err != nil {
			s.writeGithubError(w, err, "GitHub could not be asked about this repository")
			return
		}
	}

	p, err := s.store.SetProjectRepo(r.Context(), userID, pid, req.RepoURL)
	if err != nil {
		s.writeRepoStoreError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, projectResponse(store.ProjectWithLatest{Project: *p}, s.now()))
}

// errRepoNotVisible means GitHub shows the repo to nobody this server can act
// as: the App is not installed on it and it is not public.
var errRepoNotVisible = errors.New("github: repository is not visible to the server")

// requireRepoVisible refuses a link GitHub does not show to this server. The
// link is what later lets the project's syncs read the repo with the App's
// installation token, so it must be a repo the App is installed on or one that
// is public. A deployment without the App holds no credential a link could hand
// out, so there is nothing to verify and the link is kept as typed.
func (s *Server) requireRepoVisible(ctx context.Context, owner, repo string) error {
	if s.ghTokens == nil {
		return nil
	}
	visible, err := s.ghTokens.RepoVisible(ctx, owner, repo)
	if err != nil {
		return err
	}
	if !visible {
		return errRepoNotVisible
	}
	return nil
}

// writeRepoStoreError answers a failed link of a repo to a project. A refused
// second link says the repo is taken, and not by whom, to a caller who may read
// the holder; to anyone else it is answered as a repo GitHub does not show, so
// the answer does not tell which repos are linked in other workspaces
// (rv4-FR-F199).
func (s *Server) writeRepoStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "project not found", "", nil)
	case errors.Is(err, store.ErrForbidden):
		httpx.WriteError(w, http.StatusForbidden, "only the project owner or a workspace admin can link a repo", "", nil)
	case errors.Is(err, store.ErrRepoHeldOutOfReach):
		s.writeGithubError(w, errRepoNotVisible, "")
	case errors.Is(err, store.ErrRepoTaken):
		httpx.WriteError(w, http.StatusConflict, "this repository is already linked to another project", "GITHUB_REPO_ALREADY_LINKED", nil)
	default:
		s.log.Error("set project repo", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
	}
}

// writeGithubError answers a failed call about a repo to GitHub. Transport
// detail is logged and never returned, so upstream internals (URLs, headers)
// do not reach the client (F12); upstreamMsg is what the client sees instead.
func (s *Server) writeGithubError(w http.ResponseWriter, err error, upstreamMsg string) {
	var rle *github.RateLimitError
	var refused *github.AccessRefusedError
	switch {
	case errors.As(err, &rle):
		httpx.WriteError(w, http.StatusTooManyRequests, "GitHub rate limit hit — try again shortly", "GITHUB_RATE_LIMITED", map[string]any{"retry_after_seconds": rle.RetryAfterSeconds})
	case errors.As(err, &refused):
		// GitHub said no and waiting will not change it (a blocked repository, a
		// permission the App was not granted, a suspended installation): not a
		// gateway failure to retry, so a 400 that carries GitHub's own words
		// (cleaned and capped by the client) for the user to act on at GitHub.
		s.log.Warn("github refused access", "error", err)
		httpx.WriteError(w, http.StatusBadRequest, "GitHub refused access to this repository: "+refused.Message, "GITHUB_ACCESS_REFUSED", nil)
	case errors.Is(err, github.ErrRepoNotFound):
		httpx.WriteError(w, http.StatusNotFound, "GitHub repository not found — private repo, or check the name", "GITHUB_NOT_FOUND", nil)
	case errors.Is(err, errRepoURLInvalid):
		httpx.WriteError(w, http.StatusBadRequest, "the linked repository URL is not a GitHub repository — link the repository again", "GITHUB_REPO_URL_INVALID", nil)
	case errors.Is(err, errRepoNotVisible):
		httpx.WriteError(w, http.StatusBadRequest, "GitHub does not show this repository to InsideOut — install the InsideOut GitHub App on it, or link a public repository", "GITHUB_REPO_NOT_VISIBLE", nil)
	default:
		s.log.Error("github call", "error", err)
		httpx.WriteError(w, http.StatusBadGateway, upstreamMsg, "GITHUB_UPSTREAM", nil)
	}
}

// handleSyncGithub pulls recent commits from the project's linked repo into
// its progress timeline, advancing a per-project cursor so repeat syncs only
// add what's new. Owner/admin (it writes the projects sync cursor). The
// pull core lives in syncGithubProject, shared with the webhook.
func (s *Server) handleSyncGithub(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	pid, ok := pathUUID(w, r, "pid")
	if !ok {
		return
	}

	added, repoURL, err := s.syncGithubProject(r.Context(), userID, pid, 0)
	if errors.Is(err, errNoRepoLinked) {
		httpx.WriteError(w, http.StatusBadRequest, "no GitHub repo linked to this project yet", "", nil)
		return
	}
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "project not found", "", nil)
		return
	}
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "only the project owner or a workspace admin can sync", "", nil)
		return
	}
	if err != nil {
		s.writeGithubError(w, err, "GitHub sync failed")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"added": added, "repoUrl": repoURL})
}
