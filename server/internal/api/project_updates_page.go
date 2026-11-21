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
func (s *Server) updatesPage(ctx context.Context, userID, pid uuid.UUID, limit int, before *uuid.UUID) ([]projectUpdateView, *string, error) {
	// One row more than the page tells whether another page follows.
	rows, err := s.store.ListProjectUpdates(ctx, userID, pid, limit+1, before)
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
// (rv5-PARB-07).
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
	updates, next, err := s.updatesPage(r.Context(), userID, pid, limit, before)
	if err != nil {
		// A cursor that is no update of this project (deleted since) is not
		// read as the end of history.
		s.writeStoreError(w, "list project updates", err, errMsgs{store.ErrNotFound: "cursor not found in this project's updates"})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"updates": updates, "nextCursor": next})
}
