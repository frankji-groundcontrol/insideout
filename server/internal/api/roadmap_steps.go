package api

import (
	"errors"
	"net/http"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/roadmaptime"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// The ordered steps of one level (docs/architecture/roadmap-tree.md, "Steps"):
// what the Canvas draws, and what the CLI and MCP read, from one function.

type nodeTitleView struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type roadmapStepView struct {
	Node         roadmapNodeView `json:"node"`
	Stage        int             `json:"stage"`
	Loop         *int            `json:"loop"`
	After        []string        `json:"after"`
	Links        map[string]int  `json:"links"`
	ItemsInside  int             `json:"itemsInside"`
	WaitsOutside []nodeTitleView `json:"waitsOutside"`
}

type roadmapStepsView struct {
	Root      *roadmapNodeView  `json:"root"`
	Ancestors []nodeRefView     `json:"ancestors"`
	Steps     []roadmapStepView `json:"steps"`
}

// handleRoadmapSteps answers the steps under ?root={nid} in order. Without a
// root it opens a root-scoped agent token's root, else the project's one
// root, else lists the project's roots. A root-scoped agent token may not
// open a level outside its root.
func (s *Server) handleRoadmapSteps(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	pid, ok := pathUUID(w, r, "pid")
	if !ok {
		return
	}
	var root *uuid.UUID
	if raw := r.URL.Query().Get("root"); raw != "" {
		id, ok := parseUUIDParam(w, "root", raw)
		if !ok {
			return
		}
		root = &id
	}
	nodes, err := s.store.ListRoadmap(r.Context(), userID, pid)
	if err == nil && root != nil && !slices.ContainsFunc(nodes, func(n store.RoadmapNode) bool { return n.ID == *root }) {
		err = store.ErrNotFound
	}
	if err != nil {
		s.writeStoreError(w, "roadmap steps", err, errMsgs{
			store.ErrNotFound:  "project or root node not found",
			store.ErrForbidden: "you must be a member of this workspace",
		})
		return
	}
	if root != nil && s.agentRootRefuses(w, r, root) {
		return
	}
	if tok := RootOf(r.Context()); root == nil && tok != nil &&
		slices.ContainsFunc(nodes, func(n store.RoadmapNode) bool { return n.ID == *tok }) {
		root = tok
	}
	httpx.WriteJSON(w, http.StatusOK, roadmapStepsResponse(nodes, root, s.now()))
}

// roadmapStepsResponse orders the level under root (nil: chosen as
// roadmaptime.StepIDs chooses) over the project's nodes.
func roadmapStepsResponse(nodes []store.RoadmapNode, root *uuid.UUID, now time.Time) roadmapStepsView {
	byID := make(map[string]store.RoadmapNode, len(nodes))
	for _, n := range nodes {
		byID[n.ID.String()] = n
	}
	in := stepNodes(nodes)
	rootID := ""
	if root != nil {
		rootID = root.String()
	}
	rootID, stepIDs := roadmaptime.StepIDs(in, rootID)
	out := roadmapStepsView{Ancestors: []nodeRefView{}, Steps: []roadmapStepView{}}
	if n, ok := byID[rootID]; ok {
		v := roadmapNodeResponse(n, now)
		out.Root = &v
		for p, ok := byID[parentString(n)]; ok; p, ok = byID[parentString(p)] {
			out.Ancestors = append([]nodeRefView{nodeRefResponse(store.RoadmapNodeRef{ID: p.ID, Title: p.Title, OwnerID: p.OwnerID})}, out.Ancestors...)
		}
	}
	for _, st := range roadmaptime.Steps(in, stepIDs) {
		v := roadmapStepView{Node: roadmapNodeResponse(byID[st.ID], now), Stage: st.Stage, Loop: st.Loop,
			After: st.After, Links: st.Links, ItemsInside: st.ItemsInside, WaitsOutside: []nodeTitleView{}}
		for _, id := range st.WaitsOutside {
			v.WaitsOutside = append(v.WaitsOutside, nodeTitleView{ID: id, Title: byID[id].Title})
		}
		out.Steps = append(out.Steps, v)
	}
	return out
}

// stepNodes reduces store nodes to what roadmaptime.Steps reads, in list order.
func stepNodes(nodes []store.RoadmapNode) []roadmaptime.StepNode {
	out := make([]roadmaptime.StepNode, len(nodes))
	for i, n := range nodes {
		out[i] = roadmaptime.StepNode{ID: n.ID.String(), ParentID: parentString(n), Position: n.Position, DependsOn: uuidStrings(n.DependsOn)}
	}
	return out
}

func parentString(n store.RoadmapNode) string {
	if n.ParentID == nil {
		return ""
	}
	return n.ParentID.String()
}

// requestRoot is the subtree a list or Progress read asks for: ?root={nid},
// else a root-scoped agent token's root when it is a node of the project,
// else nil (the whole project). It answers 400 itself for a malformed root.
func (s *Server) requestRoot(w http.ResponseWriter, r *http.Request, userID, pid uuid.UUID) (*uuid.UUID, bool) {
	if raw := r.URL.Query().Get("root"); raw != "" {
		id, ok := parseUUIDParam(w, "root", raw)
		return &id, ok
	}
	tok := RootOf(r.Context())
	if tok == nil {
		return nil, true
	}
	n, err := s.store.GetRoadmapNode(r.Context(), userID, *tok)
	if errors.Is(err, store.ErrNotFound) || errors.Is(err, store.ErrForbidden) || (err == nil && n.ProjectID != pid) {
		return nil, true // the root lies in another project: this one reads whole
	}
	if err != nil {
		s.writeStoreError(w, "roadmap root", err, nil)
		return nil, false
	}
	return tok, true
}
