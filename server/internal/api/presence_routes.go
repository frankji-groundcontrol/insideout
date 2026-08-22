package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/presence"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// Real-time canvas presence: connecting to the stream registers the
// viewer (auth identity + a per-tab client id); every change pushes a
// full pruned snapshot; disconnect leaves. The snapshot endpoint feeds
// CLI/MCP the same truth.
func (s *Server) registerPresenceRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/projects/{pid}/presence", s.requireAuth(s.handlePresenceList))
	mux.HandleFunc("GET /api/v1/projects/{pid}/presence/stream", s.requireAuth(s.handlePresenceStream))
	mux.HandleFunc("POST /api/v1/projects/{pid}/cursor", s.requireAuth(s.handleCursor))
}

func (s *Server) handlePresenceList(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	pid, ok := pathUUID(w, r, "pid")
	if !ok {
		return
	}
	if _, err := s.store.GetProjectForMember(r.Context(), pid, userID); err != nil {
		presenceProjectError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.presence.List(pid.String()))
}

// handleCursor moves this session's canvas cursor (ephemeral broadcast
// to presence stream subscribers).
func (s *Server) handleCursor(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	pid, ok := pathUUID(w, r, "pid")
	if !ok {
		return
	}
	if _, err := s.store.GetProjectForMember(r.Context(), pid, userID); err != nil {
		presenceProjectError(w, err)
		return
	}
	var req struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body", "", nil)
		return
	}
	client := r.URL.Query().Get("client")
	if client == "" {
		client = userID.String()
	}
	name := userID.String()[:8]
	if u, err := s.store.GetUserByID(r.Context(), userID); err == nil && u.Username != "" {
		name = u.Username
	}
	s.presence.Cursor(pid.String(), client, name, req.X, req.Y)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handlePresenceStream(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	pid, ok := pathUUID(w, r, "pid")
	if !ok {
		return
	}
	if _, err := s.store.GetProjectForMember(r.Context(), pid, userID); err != nil {
		presenceProjectError(w, err)
		return
	}
	client := r.URL.Query().Get("client")
	if client == "" {
		client = userID.String()
	}
	name := userID.String()[:8]
	if u, err := s.store.GetUserByID(r.Context(), userID); err == nil && u.Username != "" {
		name = u.Username
	}

	flusher, canFlush := w.(http.Flusher)
	if !canFlush {
		httpx.WriteError(w, http.StatusInternalServerError, "streaming unsupported", "", nil)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")

	updates, cancel := s.presence.Subscribe(pid.String())
	defer cancel()
	cursorCh, cancelCursors := s.presence.SubscribeCursors(pid.String())
	defer cancelCursors()
	writeSnapshot := func(entries []presence.Entry) {
		raw, _ := json.Marshal(entries)
		fmt.Fprintf(w, "event: presence\ndata: %s\n\n", raw)
		flusher.Flush()
	}

	// Registering is itself the first change: subscribers (including
	// this one) receive the snapshot with us in it.
	s.presence.Touch(pid.String(), client, userID.String(), name)
	writeSnapshot(s.presence.List(pid.String()))

	heartbeat := time.NewTicker(10 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-r.Context().Done():
			s.presence.Leave(pid.String(), client)
			return
		case entries, okCh := <-updates:
			if !okCh {
				return
			}
			writeSnapshot(entries)
		case ev, okEv := <-cursorCh:
			if !okEv {
				return
			}
			if raw, err := json.Marshal(ev); err == nil {
				fmt.Fprintf(w, "event: cursor\ndata: %s\n\n", raw)
				flusher.Flush()
			}
		case <-heartbeat.C:
			// The ping doubles as this session's heartbeat refresh.
			s.presence.Touch(pid.String(), client, userID.String(), name)
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}

func presenceProjectError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "project not found", "", nil)
	case errors.Is(err, store.ErrForbidden):
		httpx.WriteError(w, http.StatusForbidden, "you must be a member of this workspace", "", nil)
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
	}
}
