package api

import (
	"errors"
	"net/http"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/roadmaptime"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// One node opened as its person in charge sees it (2026-11-22 view-as plan):
// the node, who is in charge, and the first glance a `mine` root carries,
// computed for the node's subtree.
type roadmapGlanceView struct {
	Node     roadmapNodeView            `json:"node"`
	InCharge *inChargeView              `json:"inCharge"`
	Now      []roadmaptime.ProgressItem `json:"now"`
	WaitsOn  []nodeLinkView             `json:"waitsOn"`
	Blocking []nodeLinkView             `json:"blocking"`
}

// inChargeView: via names the owned ancestor the person is in charge
// through (null when they own the node or own the project).
type inChargeView struct {
	UserID       string        `json:"userId"`
	Name         string        `json:"name"`
	Via          *nodeTitleRef `json:"via"`
	ProjectOwner bool          `json:"projectOwner"`
}

type nodeTitleRef struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// handleRoadmapGlance answers one node as its person in charge sees it. A
// node the caller cannot see is 404.
func (s *Server) handleRoadmapGlance(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	nid, ok := pathUUID(w, r, "nid")
	if !ok {
		return
	}
	// Kept on purpose: a refusal answers 404 so a non-member cannot learn the row exists.
	g, err := s.store.ReadRoadmapGlance(r.Context(), userID, nid)
	if errors.Is(err, store.ErrNotFound) || errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusNotFound, "node not found", "", nil)
		return
	}
	if err != nil {
		s.writeStoreError(w, "read roadmap glance", err, nil)
		return
	}
	now := s.now()
	out := roadmapGlanceView{
		Node: roadmapNodeResponse(g.Node, now), Now: roadmaptime.Assemble(g.Subtree, now).Now,
		WaitsOn: linkViews(g.WaitsOn), Blocking: linkViews(g.Blocking),
	}
	if c := g.InCharge; c != nil {
		out.InCharge = &inChargeView{UserID: c.UserID.String(), Name: c.Name, ProjectOwner: c.ProjectOwner}
		if c.Via != nil {
			out.InCharge.Via = &nodeTitleRef{ID: c.Via.ID.String(), Title: c.Via.Title}
		}
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}
