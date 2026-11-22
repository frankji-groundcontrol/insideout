package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/roadmaptime"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// One root of a person's work tree with its first glance (2026-11-22 plan,
// S2): now is the in-progress dated work below it (Progress's Now), waitsOn
// and blocking the unfinished links across its edge. The node view itself
// stays as every other read returns it.
type myRoadmapRootView struct {
	Node        roadmapNodeView            `json:"node"`
	ProjectID   string                     `json:"projectId"`
	WorkspaceID string                     `json:"workspaceId"`
	ProjectName string                     `json:"projectName"`
	Ancestors   []nodeRefView              `json:"ancestors"`
	Now         []roadmaptime.ProgressItem `json:"now"`
	WaitsOn     []nodeLinkView             `json:"waitsOn"`
	Blocking    []nodeLinkView             `json:"blocking"`
}

// nodeLinkView is a node across a dependency from the root.
type nodeLinkView struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	OwnerName string `json:"ownerName,omitempty"`
}

func linkViews(links []store.RoadmapLink) []nodeLinkView {
	out := make([]nodeLinkView, len(links))
	for i, l := range links {
		out[i] = nodeLinkView{ID: l.ID.String(), Title: l.Title, Status: l.Status, OwnerName: l.OwnerName}
	}
	return out
}

func myRoadmapResponse(roots []store.MyRoadmapRoot, now time.Time) []myRoadmapRootView {
	out := make([]myRoadmapRootView, len(roots))
	for i, root := range roots {
		out[i] = myRoadmapRootView{
			Node: roadmapNodeResponse(root.Node, now), ProjectID: root.Node.ProjectID.String(),
			WorkspaceID: root.WorkspaceID.String(), ProjectName: root.ProjectName,
			Ancestors: make([]nodeRefView, len(root.Ancestors)),
			Now:       roadmaptime.Assemble(root.Subtree, now).Now,
			WaitsOn:   linkViews(root.WaitsOn), Blocking: linkViews(root.Blocking),
		}
		for j, a := range root.Ancestors {
			out[i].Ancestors[j] = nodeRefResponse(a)
		}
	}
	return out
}

// handleMyRoadmap lists the caller's own work tree in one workspace: the
// topmost nodes they are in charge of, each a root with its project, path and
// first glance (rv5-TREEB-04). A workspace the caller is not in is 404.
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
	httpx.WriteJSON(w, http.StatusOK, myRoadmapResponse(roots, s.now()))
}

// handleAllMyRoadmap is handleMyRoadmap across every workspace the caller is
// a member of.
func (s *Server) handleAllMyRoadmap(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	roots, err := s.store.ListAllMyRoadmapRoots(r.Context(), userID)
	if err != nil {
		s.writeStoreError(w, "list all my roadmap", err, nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, myRoadmapResponse(roots, s.now()))
}
