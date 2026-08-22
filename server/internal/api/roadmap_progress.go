package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/roadmaptime"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// handleRoadmapProgress is PRODUCT.md's default Progress view: Now
// dominant (deadlined work only), at most three justified Next, Done
// counted; in-progress work without a deadline surfaces as
// needsDeadline — time is a first-class constraint.
func (s *Server) handleRoadmapProgress(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	pid, ok := pathUUID(w, r, "pid")
	if !ok {
		return
	}
	nodes, err := s.store.ListRoadmap(r.Context(), userID, pid)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "project not found", "", nil)
		return
	}
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "you must be a member of this workspace", "", nil)
		return
	}
	if err != nil {
		s.log.Error("roadmap progress", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	parents := make(map[string]bool, len(nodes))
	for _, n := range nodes {
		if n.ParentID != nil {
			parents[n.ParentID.String()] = true
		}
	}
	in := make([]roadmaptime.Node, 0, len(nodes))
	for _, n := range nodes {
		in = append(in, roadmaptime.Node{ID: n.ID.String(), Title: n.Title, Status: n.Status, Deadline: n.Deadline, Leaf: !parents[n.ID.String()]})
	}
	httpx.WriteJSON(w, http.StatusOK, roadmaptime.Assemble(in, time.Now()))
}
