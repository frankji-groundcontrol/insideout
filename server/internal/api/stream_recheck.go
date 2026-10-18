package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// streamRecheckTimeout bounds the membership read a heartbeat makes, so a slow
// database delays one ping instead of pinning the stream.
const streamRecheckTimeout = 5 * time.Second

// streamStillAllowed is what a passive SSE stream asks on every heartbeat: is
// the session that opened it still good, and may this viewer still see this
// project? A stream is authorised when it opens, so without the question it
// outlives its access token (rv4-SEC-08: a stream lives as long as its token,
// then the client reconnects with a refreshed one) and a member removed from
// the workspace keeps receiving the project's live data until their tab closes
// (rv4-REL-03). Only a definite no ends a stream; a database error answers
// yes, because a blip must not tear down every open canvas and the next
// heartbeat asks again. The cost is one membership read per open stream per
// heartbeat.
func (s *Server) streamStillAllowed(r *http.Request, userID, pid uuid.UUID) bool {
	if _, _, err := s.tokens.VerifyToken(bearerOrCookie(r)); err != nil {
		return false // the access token the stream was opened with has expired
	}
	ctx, cancel := context.WithTimeout(r.Context(), streamRecheckTimeout)
	defer cancel()
	_, err := s.store.GetProjectForMember(ctx, pid, userID)
	return !errors.Is(err, store.ErrNotFound) && !errors.Is(err, store.ErrForbidden)
}
