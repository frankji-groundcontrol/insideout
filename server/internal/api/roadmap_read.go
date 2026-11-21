package api

import (
	"errors"
	"net/http"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// handleListRoadmap answers the project's nodes as a flat list, or with
// ?root={nid} only that node and the nodes below it (rv5-TREEB-03).
func (s *Server) handleListRoadmap(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	pid, ok := pathUUID(w, r, "pid")
	if !ok {
		return
	}
	var nodes []store.RoadmapNode
	var err error
	if raw := r.URL.Query().Get("root"); raw != "" {
		root, ok := parseUUIDParam(w, "root", raw)
		if !ok {
			return
		}
		nodes, err = s.store.ListRoadmapSubtree(r.Context(), userID, pid, root)
	} else {
		nodes, err = s.store.ListRoadmap(r.Context(), userID, pid)
	}
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "project or root node not found", "", nil)
		return
	}
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "you must be a member of this workspace", "", nil)
		return
	}
	if err != nil {
		s.log.Error("list roadmap", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	now := s.now()
	views := make([]roadmapNodeView, len(nodes))
	for i, n := range nodes {
		views[i] = roadmapNodeResponse(n, now)
	}
	httpx.WriteJSON(w, http.StatusOK, views)
}

type nodeRefView struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type roadmapNodeReadView struct {
	Node        roadmapNodeView `json:"node"`
	Ancestors   []nodeRefView   `json:"ancestors"`
	ProjectID   string          `json:"projectId"`
	ProjectName string          `json:"projectName"`
}

// handleReadRoadmapNode opens one node by id: the node, its ancestors root
// first, and its project (rv5-TREEX-06). A node the caller cannot see is 404.
func (s *Server) handleReadRoadmapNode(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	nid, ok := pathUUID(w, r, "nid")
	if !ok {
		return
	}
	read, err := s.store.ReadRoadmapNode(r.Context(), userID, nid)
	if errors.Is(err, store.ErrNotFound) || errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusNotFound, "node not found", "", nil)
		return
	}
	if err != nil {
		s.log.Error("read roadmap node", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
		return
	}
	out := roadmapNodeReadView{
		Node:        roadmapNodeResponse(read.Node, s.now()),
		Ancestors:   make([]nodeRefView, len(read.Ancestors)),
		ProjectID:   read.Node.ProjectID.String(),
		ProjectName: read.ProjectName,
	}
	for i, a := range read.Ancestors {
		out.Ancestors[i] = nodeRefView{ID: a.ID.String(), Title: a.Title}
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}
