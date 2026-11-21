package api

import (
	"net/http"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

type evidenceView struct {
	ID        string `json:"id"`
	NodeID    string `json:"nodeId"`
	Kind      string `json:"kind"`
	Detail    string `json:"detail"`
	SourceURL string `json:"sourceUrl"`
	CreatedAt string `json:"createdAt"`
	// ObservedAt is the date a person saw a validation observation, as it
	// was entered (YYYY-MM-DD); omitted on the machine-appended kinds.
	ObservedAt *string `json:"observedAt,omitempty"`
}

// handleListEvidence returns a node's delivery evidence, newest first, and
// 404 for a node the caller cannot see (rv5-PARC-10).
func (s *Server) handleListEvidence(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	nid, ok := pathUUID(w, r, "nid")
	if !ok {
		return
	}
	rows, err := s.store.ListRoadmapEvidence(r.Context(), userID, nid)
	if err != nil {
		s.writeStoreError(w, "evidence list", err, errMsgs{store.ErrNotFound: "node not found"})
		return
	}
	views := make([]evidenceView, len(rows))
	for i, e := range rows {
		views[i] = evidenceView{
			ID: e.ID.String(), NodeID: e.NodeID.String(), Kind: e.Kind, Detail: e.Detail,
			SourceURL: e.SourceURL, CreatedAt: e.CreatedAt.Format(timeLayout),
		}
		if e.ObservedAt != nil {
			d := e.ObservedAt.Format("2006-01-02")
			views[i].ObservedAt = &d
		}
	}
	httpx.WriteJSON(w, http.StatusOK, views)
}
