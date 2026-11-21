package api

import (
	"net/http"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

type prdListView struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Status     string  `json:"status"`
	AuthorID   string  `json:"authorId"`
	AuthorName string  `json:"authorName"`
	ProjectID  *string `json:"projectId"`
	UpdatedAt  string  `json:"updatedAt"`
}

// handleListWorkspacePrds lists every PRD of a workspace to its members,
// newest first (rv5-FE-02, PARB-03).
func (s *Server) handleListWorkspacePrds(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserID(r.Context())
	wsID, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	rows, err := s.store.ListWorkspacePrds(r.Context(), userID, wsID)
	if err != nil {
		s.writeStoreError(w, "list workspace prds", err, errMsgs{store.ErrNotFound: "workspace not found"})
		return
	}
	views := make([]prdListView, len(rows))
	for i, p := range rows {
		views[i] = prdListView{
			ID: p.ID.String(), Title: p.Title, Status: p.Status, AuthorID: p.AuthorID.String(),
			AuthorName: p.AuthorName, ProjectID: uuidString(p.ProjectID), UpdatedAt: p.UpdatedAt.Format(timeLayout),
		}
	}
	httpx.WriteJSON(w, http.StatusOK, views)
}
