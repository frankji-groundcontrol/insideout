package httpx

import (
	"net/http"
	"time"
)

// SSEWriteTimeout bounds one SSE write to a client. It is not a stream
// lifetime: the deadline is re-armed before every write, so a stream that
// idles longer than this between events is fine, but a write that cannot
// complete in this long (a client that stopped reading) fails and frees the
// handler.
const SSEWriteTimeout = 30 * time.Second

// ArmWriteDeadline sets the connection's write deadline to d from now. The
// server has no WriteTimeout (an SSE stream must stay open), and net/http
// clears the deadline after each request, so keep-alive connections are
// unaffected. The error is ignored on purpose: ErrNotSupported only comes
// from recorders in tests, and a real failure will surface in the write.
func ArmWriteDeadline(w http.ResponseWriter, d time.Duration) {
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(d))
}
