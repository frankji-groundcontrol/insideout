package main

import (
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/clientconfig"
)

// Review-4 FR-F69. `setup --check` took every *url.Error for "unreachable
// (offline?) — not failed", so a certificate this machine does not trust, a
// scheme that is not http(s), or a connection the peer breaks passed with exit
// 0 and a line saying "not failed": the very cases a check of the API address
// exists to catch. Only a target that cannot be reached at all (refused, no
// route, no such host, timed out) is the offline skip. The anonymous probe has
// the same flaw, because probe.APIProbe reports every transport failure as
// unreachable.

// rvSetupChkBadTarget is an API address whose peer answers, or tries to, and
// must not be trusted or understood. api returns the base and a count of the
// requests the peer served: a credential must never be among them.
type rvSetupChkBadTarget struct {
	name string
	api  func(t *testing.T) (base string, served func() int)
}

func rvSetupChkBadTargets() []rvSetupChkBadTarget {
	return []rvSetupChkBadTarget{
		{"certificate this machine does not trust", func(t *testing.T) (string, func() int) {
			srv, served := rvSetupChkServer(t, true)
			return srv.URL + "/api/v1", served
		}},
		{"https against a plain http port", func(t *testing.T) (string, func() int) {
			srv, served := rvSetupChkServer(t, false)
			return strings.Replace(srv.URL, "http://", "https://", 1) + "/api/v1", served
		}},
		{"scheme that is not http", func(t *testing.T) (string, func() int) {
			return "htp://127.0.0.1:1/api/v1", func() int { return 0 }
		}},
		{"peer that hangs up", func(t *testing.T) (string, func() int) {
			return rvSetupChkHangUp(t) + "/api/v1", func() int { return 0 }
		}},
	}
}

// rvSetupChkServer is an API that answers 401 and counts what reaches it; with
// secure it speaks TLS under a certificate no trust store knows.
func rvSetupChkServer(t *testing.T, secure bool) (*httptest.Server, func() int) {
	t.Helper()
	var n atomic.Int32
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	srv.Config.ErrorLog = log.New(io.Discard, "", 0) // the failed handshakes are the point
	if secure {
		srv.StartTLS()
	} else {
		srv.Start()
	}
	t.Cleanup(srv.Close)
	return srv, func() int { return int(n.Load()) }
}

// rvSetupChkHangUp is a listener that takes each connection and closes it
// without a word, as a middlebox that breaks connections does.
func rvSetupChkHangUp(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			c.Close()
		}
	}()
	return "http://" + ln.Addr().String()
}

func TestRvSetupChkATargetThatCannotBeTrustedFailsInsteadOfSkipping(t *testing.T) {
	lanes := []struct{ name, token string }{{"signed in", "tok-a"}, {"anonymous", ""}}
	for _, target := range rvSetupChkBadTargets() {
		for _, lane := range lanes {
			t.Run(target.name+"/"+lane.name, func(t *testing.T) {
				base, served := target.api(t)

				code, out := checkWith(t, clientconfig.Config{APIBase: base, Token: lane.token})

				line := rvSetupChkLine(out, "api")
				if code != 1 || !strings.Contains(line, "fail") || strings.Contains(line, "not failed") {
					t.Errorf("exit %d, api line %q; want a failed check, not an offline skip", code, line)
				}
				if n := served(); n != 0 {
					t.Errorf("the target served %d request(s); nothing, a credential least of all, may reach a peer that cannot be verified", n)
				}
			})
		}
	}
}
