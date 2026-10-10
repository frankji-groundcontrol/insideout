package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// REL-15: http.ResponseController reaches the connection (SetWriteDeadline,
// Flush) only through a wrapper's Unwrap method. statusRecorder wraps every
// response in withLogging and in the auth throttle, so without Unwrap no
// handler can bound a write to a client that stopped reading.
func TestStatusRecorder_LetsResponseControllerReachTheConnection(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		if err := http.NewResponseController(rec).SetWriteDeadline(time.Now().Add(time.Second)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("SetWriteDeadline through a statusRecorder: status %d (%s), want 200", resp.StatusCode, body)
	}
}
