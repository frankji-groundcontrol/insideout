package api

import (
	"errors"
	"net/http"

	"github.com/frankji-groundcontrol/insideout/server/internal/guide"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// handleProjectGuide returns the scaffolded insideout.yaml matching
// guide for a project's roadmap — the artifact users commit to their
// repo so GitHub events can attach evidence to roadmap leaves.
func (s *Server) handleProjectGuide(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	pid, ok := pathUUID(w, r, "pid")
	if !ok {
		return
	}
	p, err := s.store.GetProjectForMember(r.Context(), pid, userID)
	if err != nil {
		s.writeStoreError(w, "guide: get project", err, errMsgs{
			store.ErrNotFound:  "project not found",
			store.ErrForbidden: "you must be a member of this workspace",
		})
		return
	}
	// Kept on purpose: a project with no roadmap yet reads as an empty one.
	nodes, err := s.store.ListRoadmap(r.Context(), userID, pid)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		s.writeStoreError(w, "guide: list roadmap", err, nil)
		return
	}
	parents := make(map[string]bool, len(nodes))
	for _, n := range nodes {
		if n.ParentID != nil {
			parents[n.ParentID.String()] = true
		}
	}
	gn := make([]guide.Node, 0, len(nodes))
	for _, n := range nodes {
		gn = append(gn, guide.Node{ID: n.ID.String(), Title: n.Title, Leaf: !parents[n.ID.String()]})
	}
	w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(guide.Generate(p.Title, gn))); err != nil {
		s.log.Error("guide: write", "error", err)
	}
}
