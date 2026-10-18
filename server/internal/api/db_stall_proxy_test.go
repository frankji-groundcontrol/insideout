package api

import (
	"net"
	"sync"
	"testing"
)

// stallProxy is a TCP relay in front of the test database that can go dead
// without closing. After stall() the bytes either side sends are swallowed and
// every connection stays open, which is how a frozen database host or a
// black-holed network looks to the pool: the statement never reaches the
// server, so its statement_timeout cannot fire and only a client-side deadline
// can end the wait.
type stallProxy struct {
	ln     net.Listener
	target string

	mu      sync.Mutex
	stalled bool
	conns   []net.Conn
}

func newStallProxy(t *testing.T, target string) *stallProxy {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for the stall proxy: %v", err)
	}
	p := &stallProxy{ln: ln, target: target}
	t.Cleanup(p.close)
	go func() {
		for {
			client, err := ln.Accept()
			if err != nil {
				return
			}
			upstream, err := net.Dial("tcp", target)
			if err != nil {
				client.Close()
				continue
			}
			p.track(client, upstream)
			go p.relay(upstream, client)
			go p.relay(client, upstream)
		}
	}()
	return p
}

// addr is where to point the pool instead of the database.
func (p *stallProxy) addr() string { return p.ln.Addr().String() }

// stall black-holes everything from now on.
func (p *stallProxy) stall() {
	p.mu.Lock()
	p.stalled = true
	p.mu.Unlock()
}

func (p *stallProxy) isStalled() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stalled
}

func (p *stallProxy) track(conns ...net.Conn) {
	p.mu.Lock()
	p.conns = append(p.conns, conns...)
	p.mu.Unlock()
}

func (p *stallProxy) relay(dst, src net.Conn) {
	buf := make([]byte, 32*1024)
	for {
		n, err := src.Read(buf)
		if n > 0 && !p.isStalled() {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				break
			}
		}
		if err != nil {
			break
		}
	}
	dst.Close()
	src.Close()
}

func (p *stallProxy) close() {
	p.ln.Close()
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, c := range p.conns {
		c.Close()
	}
}
