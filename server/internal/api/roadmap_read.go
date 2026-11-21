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
	if err != nil {
		s.writeStoreError(w, "list roadmap", err, errMsgs{
			store.ErrNotFound:  "project or root node not found",
			store.ErrForbidden: "you must be a member of this workspace",
		})
		return
	}
	now := s.now()
	views := make([]roadmapNodeView, len(nodes))
	for i, n := range nodes {
		views[i] = roadmapNodeResponse(n, now)
	}
	httpx.WriteJSON(w, http.StatusOK, views)
}

// nodeRefView is one ancestor: ownerId (null: none) lets a client see that
// the caller owns the opened tree through an ancestor.
type nodeRefView struct {
	ID      string  `json:"id"`
	Title   string  `json:"title"`
	OwnerID *string `json:"ownerId"`
}

func nodeRefResponse(a store.RoadmapNodeRef) nodeRefView {
	v := nodeRefView{ID: a.ID.String(), Title: a.Title}
	if a.OwnerID != nil {
		id := a.OwnerID.String()
		v.OwnerID = &id
	}
	return v
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
	// Kept on purpose: a refusal answers 404 so a non-member cannot learn the row exists.
	read, err := s.store.ReadRoadmapNode(r.Context(), userID, nid)
	if errors.Is(err, store.ErrNotFound) || errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusNotFound, "node not found", "", nil)
		return
	}
	if err != nil {
		s.writeStoreError(w, "read roadmap node", err, nil)
		return
	}
	out := roadmapNodeReadView{
		Node:        roadmapNodeResponse(read.Node, s.now()),
		Ancestors:   make([]nodeRefView, len(read.Ancestors)),
		ProjectID:   read.Node.ProjectID.String(),
		ProjectName: read.ProjectName,
	}
	for i, a := range read.Ancestors {
		out.Ancestors[i] = nodeRefResponse(a)
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

type myRoadmapRootView struct {
	Node        roadmapNodeView `json:"node"`
	ProjectID   string          `json:"projectId"`
	ProjectName string          `json:"projectName"`
	Ancestors   []nodeRefView   `json:"ancestors"`
}

// handleMyRoadmap lists the caller's own work tree in one workspace: the
// topmost nodes they own, each a root with its project and path
// (rv5-TREEB-04). A workspace the caller is not in is 404.
func (s *Server) handleMyRoadmap(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	wsID, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	// Kept on purpose: a refusal answers 404 so a non-member cannot learn the row exists.
	roots, err := s.store.ListMyRoadmapRoots(r.Context(), userID, wsID)
	if errors.Is(err, store.ErrNotFound) || errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusNotFound, "workspace not found", "", nil)
		return
	}
	if err != nil {
		s.writeStoreError(w, "list my roadmap", err, nil)
		return
	}
	now := s.now()
	out := make([]myRoadmapRootView, len(roots))
	for i, root := range roots {
		out[i] = myRoadmapRootView{
			Node: roadmapNodeResponse(root.Node, now), ProjectID: root.Node.ProjectID.String(),
			ProjectName: root.ProjectName, Ancestors: make([]nodeRefView, len(root.Ancestors)),
		}
		for j, a := range root.Ancestors {
			out[i].Ancestors[j] = nodeRefResponse(a)
		}
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}
