package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// Delegation on the record (rv5-TREEB-01, TREEB-09, TREEX-08): hand a node to
// a member, hand it back, read who held it. Delegate and hand-back are
// human-only (owner decision 4): ownership is write authority.

func (s *Server) registerDelegationRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/roadmap/{nid}/delegate", s.requireAuth(s.handleDelegateNode))
	mux.HandleFunc("POST /api/v1/roadmap/{nid}/handback", s.requireAuth(s.handleHandBackNode))
	mux.HandleFunc("GET /api/v1/roadmap/{nid}/assignments", s.requireAuth(s.handleListAssignments))
}

// uuidString is id as a string, nil for nil.
func uuidString(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}
	s := id.String()
	return &s
}

const maxAssignmentNote = 500

// assignmentNote trims a note and refuses one over the column's bound.
func assignmentNote(w http.ResponseWriter, note string) (string, bool) {
	note = strings.TrimSpace(note)
	if tooLong(note, maxAssignmentNote) {
		httpx.WriteError(w, http.StatusBadRequest, "note is too long (max 500 characters)", "", nil)
		return "", false
	}
	return note, true
}

func (s *Server) handleDelegateNode(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	nid, ok := pathUUID(w, r, "nid")
	if !ok {
		return
	}
	var req struct {
		ToUserID string `json:"toUserId"`
		Note     string `json:"note"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	to, err := uuid.Parse(req.ToUserID)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "toUserId must be a user id", "", nil)
		return
	}
	note, ok := assignmentNote(w, req.Note)
	if !ok {
		return
	}
	n, err := s.store.DelegateRoadmapNode(r.Context(), userID, nid, to, note)
	s.writeOwnerChange(w, n, err, "delegate roadmap node",
		"only the project owner, a workspace admin, or the owner of the branch above can change this node's owner")
}

func (s *Server) handleHandBackNode(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	nid, ok := pathUUID(w, r, "nid")
	if !ok {
		return
	}
	var req struct {
		Note string `json:"note"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	note, ok := assignmentNote(w, req.Note)
	if !ok {
		return
	}
	n, err := s.store.HandBackRoadmapNode(r.Context(), userID, nid, note)
	s.writeOwnerChange(w, n, err, "hand back roadmap node", "only the node's current owner can hand it back")
}

// writeOwnerChange answers a delegate or hand-back; forbidden says who may.
func (s *Server) writeOwnerChange(w http.ResponseWriter, n *store.RoadmapNode, err error, what, forbidden string) {
	if err != nil {
		s.writeStoreError(w, what, err, errMsgs{store.ErrNotFound: "node not found", store.ErrForbidden: forbidden})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, roadmapNodeResponse(*n, s.now()))
}

type nodeAssignmentView struct {
	ID             string  `json:"id"`
	Kind           string  `json:"kind"`
	FromUserID     *string `json:"fromUserId"`
	FromName       *string `json:"fromName"`
	ToUserID       *string `json:"toUserId"`
	ToName         *string `json:"toName"`
	AssignedBy     *string `json:"assignedBy"`
	AssignedByName *string `json:"assignedByName"`
	Note           string  `json:"note"`
	CreatedAt      string  `json:"createdAt"`
}

func (s *Server) handleListAssignments(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	nid, ok := pathUUID(w, r, "nid")
	if !ok {
		return
	}
	// Kept on purpose: a refusal answers 404 so a non-member cannot learn the row exists.
	rows, err := s.store.ListNodeAssignments(r.Context(), userID, nid)
	if errors.Is(err, store.ErrNotFound) || errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusNotFound, "node not found", "", nil)
		return
	}
	if err != nil {
		s.writeStoreError(w, "list node assignments", err, nil)
		return
	}
	out := make([]nodeAssignmentView, len(rows))
	for i, a := range rows {
		out[i] = nodeAssignmentView{
			ID: a.ID.String(), Kind: a.Kind, Note: a.Note, CreatedAt: a.CreatedAt.Format(timeLayout),
			FromUserID: uuidString(a.FromUserID), FromName: a.FromName,
			ToUserID: uuidString(a.ToUserID), ToName: a.ToName,
			AssignedBy: uuidString(a.AssignedBy), AssignedByName: a.AssignedByName,
		}
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}
