package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/roadmaptime"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// handleRoadmapProgress is PRODUCT.md's default Progress view: Now
// dominant (deadlined work only), at most three justified Next, Done
// listed (doneCount is the length of that list); in-progress work
// without a deadline surfaces as needsDeadline — time is a first-class
// constraint.
func (s *Server) handleRoadmapProgress(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	pid, ok := pathUUID(w, r, "pid")
	if !ok {
		return
	}
	view, err := s.roadmapProgress(r, userID, pid)
	if err != nil {
		s.writeRoadmapProgressError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, view)
}

// handleRoadmapProgressStream is the live half of the Progress view.
// Connecting writes the current snapshot; every later node change on
// the project writes it again as an event named "progress".
func (s *Server) handleRoadmapProgressStream(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	pid, ok := pathUUID(w, r, "pid")
	if !ok {
		return
	}
	flusher, canFlush := w.(http.Flusher)
	if !canFlush {
		httpx.WriteError(w, http.StatusInternalServerError, "streaming unsupported", "", nil)
		return
	}
	view, err := s.roadmapProgress(r, userID, pid)
	if err != nil {
		s.writeRoadmapProgressError(w, err)
		return
	}
	release, ok := s.takeStreamSlot(w, userID, pid)
	if !ok {
		return
	}
	defer release()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")

	updates, cancel := s.progress.Subscribe(pid.String())
	defer cancel()
	write := func(v roadmaptime.Progress) {
		raw, err := json.Marshal(v)
		if err != nil {
			return
		}
		httpx.ArmWriteDeadline(w, httpx.SSEWriteTimeout)
		fmt.Fprintf(w, "event: progress\ndata: %s\n\n", raw)
		flusher.Flush()
	}
	write(view)

	heartbeat := time.NewTicker(streamHeartbeat)
	defer heartbeat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-s.streamsCtx.Done():
			return
		case _, okCh := <-updates:
			if !okCh {
				return
			}
			next, err := s.roadmapProgress(r, userID, pid)
			if err != nil {
				s.log.Error("roadmap progress stream", "error", err)
				continue
			}
			write(next)
		case <-heartbeat.C:
			httpx.ArmWriteDeadline(w, httpx.SSEWriteTimeout)
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}

func (s *Server) roadmapProgress(r *http.Request, userID, pid uuid.UUID) (roadmaptime.Progress, error) {
	nodes, err := s.store.ListRoadmap(r.Context(), userID, pid)
	if err != nil {
		return roadmaptime.Progress{}, err
	}
	parents := make(map[string]bool, len(nodes))
	for _, n := range nodes {
		if n.ParentID != nil {
			parents[n.ParentID.String()] = true
		}
	}
	in := make([]roadmaptime.Node, 0, len(nodes))
	for _, n := range nodes {
		in = append(in, roadmaptime.Node{ID: n.ID.String(), Title: n.Title, Status: n.Status, Deadline: n.Deadline, Leaf: !parents[n.ID.String()], Acceptance: n.Acceptance, OwnerName: n.OwnerName})
	}
	return roadmaptime.Assemble(in, s.now()), nil
}

func (s *Server) writeRoadmapProgressError(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "project not found", "", nil)
		return
	}
	if errors.Is(err, store.ErrForbidden) {
		httpx.WriteError(w, http.StatusForbidden, "you must be a member of this workspace", "", nil)
		return
	}
	s.log.Error("roadmap progress", "error", err)
	httpx.WriteError(w, http.StatusInternalServerError, "internal error", "", nil)
}
