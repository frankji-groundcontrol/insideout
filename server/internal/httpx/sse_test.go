package httpx

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// REL-15: a client that stops reading pins its handler goroutine in Write
// until the kernel gives up (minutes, or never), because nothing bounds the
// write. ArmWriteDeadline has to turn that into an error once the deadline
// passes. A raw TCP client sends the request and then never reads, so the
// server's send buffers fill and the next write really blocks.
func TestArmWriteDeadline_BoundsAWriteToAClientThatNeverReads(t *testing.T) {
	handlerErr := make(chan error, 1)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chunk := bytes.Repeat([]byte("x"), 64<<10)
		rc := http.NewResponseController(w)
		for {
			ArmWriteDeadline(w, 200*time.Millisecond)
			if _, err := w.Write(chunk); err != nil {
				handlerErr <- err
				return
			}
			if err := rc.Flush(); err != nil {
				handlerErr <- err
				return
			}
		}
	}))
	t.Cleanup(ts.Close)

	conn, err := net.Dial("tcp", ts.Listener.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	// Registered after ts.Close, so it runs first: closing the client frees a
	// handler that a failing run left blocked, instead of hanging ts.Close.
	t.Cleanup(func() { conn.Close() })
	conn.(*net.TCPConn).SetReadBuffer(4096) // a small window, so the buffers fill fast
	fmt.Fprint(conn, "GET / HTTP/1.1\r\nHost: sse.test\r\n\r\n")

	select {
	case err := <-handlerErr:
		if !errors.Is(err, os.ErrDeadlineExceeded) {
			t.Fatalf("the blocked write failed with %v, want a deadline error", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("handler still blocked writing to a client that never reads, 3s after a 200ms write deadline")
	}
}
