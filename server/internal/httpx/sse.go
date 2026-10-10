package httpx

import (
	"net/http"
	"time"
)

// SSEWriteTimeout bounds one SSE write to a client.
const SSEWriteTimeout = 30 * time.Second

// ArmWriteDeadline is scaffolding so the SSE write-deadline test compiles:
// it does nothing yet.
func ArmWriteDeadline(w http.ResponseWriter, d time.Duration) {}
