package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// maxUpdatesPage bounds the limit a caller may ask the updates feed for.
const maxUpdatesPage = 100

// updatesPage reads one page of a project's updates, newest first, after the
// update before (nil: from the newest). nextCursor is the id of the page's
// oldest update when older ones remain, else nil. The caller has checked
// membership.
func (s *Server) updatesPage(ctx context.Context, userID, pid uuid.UUID, limit int, before *uuid.UUID, f store.UpdateFilter) ([]projectUpdateView, *string, error) {
	// One row more than the page tells whether another page follows.
	rows, err := s.store.ListProjectUpdatesWhere(ctx, userID, pid, limit+1, before, f)
	if err != nil {
		return nil, nil, err
	}
	var next *string
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[limit-1].ID.String()
		next = &last
	}
	views := make([]projectUpdateView, len(rows))
	for i, u := range rows {
		views[i] = projectUpdateResponse(u)
	}
	return views, next, nil
}

// handleListProjectUpdates is GET /projects/{pid}/updates?before=&limit=: the
// page after the nextCursor the project page or an earlier page returned
// (rv5-PARB-07). ?node= and ?pending=true narrow it (store.UpdateFilter).
func (s *Server) handleListProjectUpdates(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	pid, ok := pathUUID(w, r, "pid")
	if !ok {
		return
	}
	limit := store.ProjectUpdatesPageSize
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > maxUpdatesPage {
			httpx.WriteError(w, http.StatusBadRequest, "limit must be 1 to 100", "", nil)
			return
		}
		limit = n
	}
	var before *uuid.UUID
	if raw := r.URL.Query().Get("before"); raw != "" {
		id, ok := parseUUIDParam(w, "before", raw)
		if !ok {
			return
		}
		before = &id
	}
	if _, err := s.store.GetProjectForMember(r.Context(), pid, userID); err != nil {
		s.writeStoreError(w, "get project", err, errMsgs{store.ErrNotFound: "project not found"})
		return
	}
	f, ok := s.updateFilter(w, r, userID, pid)
	if !ok {
		return
	}
	updates, next, err := s.updatesPage(r.Context(), userID, pid, limit, before, f)
	if err != nil {
		// A cursor that is no update of this project (deleted since) is not
		// read as the end of history.
		s.writeStoreError(w, "list project updates", err, errMsgs{store.ErrNotFound: "cursor not found in this project's updates"})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"updates": updates, "nextCursor": next})
}

// updateFilter reads ?node= (a node of this project) and ?pending=true.
func (s *Server) updateFilter(w http.ResponseWriter, r *http.Request, userID, pid uuid.UUID) (store.UpdateFilter, bool) {
	var f store.UpdateFilter
	q := r.URL.Query()
	if raw := q.Get("pending"); raw != "" {
		pending, err := strconv.ParseBool(raw)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "pending must be true or false", "", nil)
			return f, false
		}
		f.Pending = pending
	}
	if raw := q.Get("node"); raw != "" {
		id, ok := parseUUIDParam(w, "node", raw)
		if !ok {
			return f, false
		}
		n, err := s.store.GetRoadmapNode(r.Context(), userID, id)
		if err == nil && n.ProjectID != pid {
			err = store.ErrNotFound
		}
		if err != nil {
			s.writeStoreError(w, "updates node", err, errMsgs{store.ErrNotFound: "node not found in this project"})
			return f, false
		}
		f.Node = &id
	}
	return f, true
}
