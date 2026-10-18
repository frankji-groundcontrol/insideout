package agent

import (
	"context"
	"errors"
	"log/slog"
)

// logErr reports a failed best-effort write of a run's bookkeeping: its terminal
// status, the breaker result, the text that streamed, the heartbeat. These writes
// happen after the client has its answer (or while the turn is being torn down),
// so there is no caller to return the error to; but a run whose terminal write is
// lost stays "running" until the reaper, and a breaker that misses failures never
// opens, so the operator has to be able to see which write was lost and why. op is
// the store method that failed and attrs say what it was writing (the run id).
//
// A nil err logs nothing, so a call site wraps the store call directly. A
// cancelled context is not a lost write: only the heartbeat runs on the request's
// own context, and a client that left is routine (the loop's own handling of it
// is already quiet); every terminal write runs detached, so it can not be
// cancelled, only time out, and that is logged.
func logErr(log *slog.Logger, op string, err error, attrs ...any) {
	if err == nil || errors.Is(err, context.Canceled) {
		return
	}
	log.Error("ai run bookkeeping write failed", append([]any{"op", op, "error", err}, attrs...)...)
}
