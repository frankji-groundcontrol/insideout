// Package api wires HTTP routes, middleware, and per-domain handlers on
// top of the store and auth packages.
package api

import (
	"context"

	"github.com/google/uuid"
)

type contextKey int

const userHolderKey contextKey = iota

// userHolder is a per-request mutable box installed into the context once
// by withRequestID (the outermost middleware). Because it's a pointer,
// requireAuth — which runs much further down the handler chain, after the
// mux has dispatched to a specific route — can populate it, and
// withLogging can read it back after next.ServeHTTP returns. A plain
// context.WithValue from requireAuth would NOT work here: http.Request
// copies via WithContext don't propagate back up to the caller, only
// down, so an outer middleware can never see a context value added by a
// handler nested deeper in the chain unless they share mutable state like
// this holder.
type userHolder struct {
	id uuid.UUID
	ok bool
}

func newUserHolderContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, userHolderKey, &userHolder{})
}

func setUserID(ctx context.Context, id uuid.UUID) {
	if h, ok := ctx.Value(userHolderKey).(*userHolder); ok {
		h.id = id
		h.ok = true
	}
}

// UserID returns the authenticated user id resolved by requireAuth, and
// whether one was present.
func UserID(ctx context.Context) (uuid.UUID, bool) {
	h, ok := ctx.Value(userHolderKey).(*userHolder)
	if !ok || !h.ok {
		return uuid.UUID{}, false
	}
	return h.id, true
}
