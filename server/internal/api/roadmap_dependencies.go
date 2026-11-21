package api

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// Node dependencies (2026-11-22 plan, "Dependency contract"): a node waits on
// other nodes of its project. The writer of the waiting node {nid} adds and
// removes links; a root-scoped agent token needs {nid} inside its root.

func (s *Server) registerDependencyRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/roadmap/{nid}/dependencies", s.requireAuth(s.handleAddDependency))
	mux.HandleFunc("DELETE /api/v1/roadmap/{nid}/dependencies/{did}", s.requireAuth(s.handleRemoveDependency))
}

// uuidStrings is ids as strings, [] (never nil) for none.
func uuidStrings(ids []uuid.UUID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out
}

// handleAddDependency answers 201 with the view of {nid}; an existing link is
// no change.
func (s *Server) handleAddDependency(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	nid, ok := pathUUID(w, r, "nid")
	if !ok {
		return
	}
	var req struct {
		DependsOnID string `json:"dependsOnId"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	did, ok := parseUUIDParam(w, "dependsOnId", req.DependsOnID)
	if !ok || s.agentRootRefuses(w, r, &nid) {
		return
	}
	n, err := s.store.AddRoadmapDependency(r.Context(), userID, nid, did)
	if err != nil {
		s.writeStoreError(w, "add roadmap dependency", err, errMsgs{
			store.ErrNotFound:  "node not found",
			store.ErrForbidden: nodeWriteDenied,
		})
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, roadmapNodeResponse(*n, s.now()))
}

// handleRemoveDependency answers 204, or 404 when there is no such link.
func (s *Server) handleRemoveDependency(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	nid, ok := pathUUID(w, r, "nid")
	if !ok {
		return
	}
	did, ok := pathUUID(w, r, "did")
	if !ok || s.agentRootRefuses(w, r, &nid) {
		return
	}
	if err := s.store.RemoveRoadmapDependency(r.Context(), userID, nid, did); err != nil {
		s.writeStoreError(w, "remove roadmap dependency", err, errMsgs{
			store.ErrNotFound:  "dependency not found",
			store.ErrForbidden: nodeWriteDenied,
		})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
