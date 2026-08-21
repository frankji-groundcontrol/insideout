package api

import (
	"net/http"

	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/google/uuid"
)

// pathUUID parses a Go 1.22+ mux path wildcard as a UUID, writing a 400
// and returning ok=false if it isn't one.
func pathUUID(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	raw := r.PathValue(name)
	id, err := uuid.Parse(raw)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid "+name, "", nil)
		return uuid.UUID{}, false
	}
	return id, true
}

// parseUUIDParam parses any string (query/body) as a UUID, answering
// with the same 400 shape as pathUUID.
func parseUUIDParam(w http.ResponseWriter, name, raw string) (uuid.UUID, bool) {
	id, err := uuid.Parse(raw)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid "+name, "", nil)
		return uuid.UUID{}, false
	}
	return id, true
}
