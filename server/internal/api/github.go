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

	commits, err := github.FetchCommits(r.Context(), repoURL, 20)
	if err != nil {
		s.log.Error("github fetch", "error", err)
		httpx.WriteError(w, http.StatusBadGateway, err.Error(), "GITHUB_UPSTREAM", nil)
		return
	}

	// New commits are those before the cursor in the newest-first list.
	var fresh []github.Commit
	for _, c := range commits {
		if c.SHA == lastSHA {
			break
		}
		fresh = append(fresh, c)
	}

	// Insert oldest-first so created_at ordering matches commit order.
	added := 0
	for i := len(fresh) - 1; i >= 0; i-- {
		c := fresh[i]
		sha := c.SHA
		if len(sha) > 7 {
			sha = sha[:7]
		}
		content := fmt.Sprintf("[github %s] %s — %s", sha, c.Message, c.Author)
		if _, err := s.store.AddProjectUpdate(r.Context(), userID, pid, "progress", content); err != nil {
			s.log.Error("insert synced update", "error", err)
			httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
			return
		}
		added++
	}

	if len(fresh) > 0 {
		if err := s.store.RecordRepoSyncSHA(r.Context(), userID, pid, fresh[0].SHA); err != nil {
			s.log.Error("record sync cursor", "error", err)
			httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
			return
		}
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"added": added, "repoUrl": repoURL})
}
