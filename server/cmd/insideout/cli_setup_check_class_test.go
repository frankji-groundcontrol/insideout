package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strings"
	"syscall"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/clientconfig"
	"github.com/frankji-groundcontrol/insideout/server/internal/setup/probe"
)

// Review-4 FR-F69: which transport errors are "the target could not be reached
// at all" (the offline skip) and which are a peer that failed the check. The
// live cases are in cli_setup_check_tls_test.go; these are the errors a test
// cannot make a local server produce (a name that does not resolve, a proxy
// that is down, a timeout) next to the ones it can, so the whitelist is pinned
// both ways.
func TestRvSetupChkTargetUnreachableIsAWhitelist(t *testing.T) {
	refused := &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}
	noHost := &net.OpError{Op: "dial", Net: "tcp", Err: &net.DNSError{Err: "no such host", Name: "api.example.invalid", IsNotFound: true}}
	get := func(err error) error {
		return &url.Error{Op: "Get", URL: "https://api.example.invalid/api/v1/me", Err: err}
	}

	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		// Nothing was reached: the offline skip.
		{"connection refused", get(refused), true},
		{"no such host", get(noHost), true},
		{"a bare resolver error", &net.DNSError{Err: "server misbehaving", Name: "x"}, true},
		{"the proxy is down", get(&net.OpError{Op: "proxyconnect", Net: "tcp", Err: refused}), true},
		{"the request deadline passed", get(context.DeadlineExceeded), true},
		{"the connection went quiet", get(&net.OpError{Op: "read", Net: "tcp", Err: os.ErrDeadlineExceeded}), true},
		{"refused, as APIProbe reports it", fmt.Errorf("%w: %w", probe.ErrUnreachable, get(refused)), true},

		// A peer answered, or tried to: the check fails.
		{"certificate from an unknown authority", get(x509.UnknownAuthorityError{}), false},
		{"certificate the verifier rejects", get(&tls.CertificateVerificationError{Err: x509.UnknownAuthorityError{}}), false},
		{"certificate for another host", get(x509.HostnameError{Certificate: &x509.Certificate{}, Host: "api.example.invalid"}), false},
		{"expired certificate", get(x509.CertificateInvalidError{Cert: &x509.Certificate{}, Reason: x509.Expired}), false},
		{"not a TLS handshake", get(tls.RecordHeaderError{Msg: "first record does not look like a TLS handshake"}), false},
		{"scheme that is not http", get(errors.New(`unsupported protocol scheme "htp"`)), false},
		{"http answer to an https client", get(errors.New("http: server gave HTTP response to HTTPS client")), false},
		{"the peer closed the connection", get(io.EOF), false},
		{"the peer reset the connection", get(&net.OpError{Op: "read", Net: "tcp", Err: syscall.ECONNRESET}), false},
		{"cancelled", get(context.Canceled), false},
		{"untrusted, as APIProbe reports it", fmt.Errorf("%w: %w", probe.ErrUnreachable, get(x509.UnknownAuthorityError{})), false},
		{"no error at all", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := targetUnreachable(tc.err); got != tc.want {
				t.Errorf("targetUnreachable(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

// A peer that failed the check is named by what went wrong with it, never as
// "unreachable": that word is the offline skip, and probe.APIProbe prefixes it
// to every transport failure.
func TestRvSetupChkAPeerThatFailedIsNamedByItsCauseNotAsUnreachable(t *testing.T) {
	for _, lane := range []struct{ name, token string }{{"signed in", "tok-a"}, {"anonymous", ""}} {
		t.Run(lane.name, func(t *testing.T) {
			base, _ := rvSetupChkBadTargets()[0].api(t) // the certificate this machine does not trust

			code, out := checkWith(t, clientconfig.Config{APIBase: base, Token: lane.token})

			line := rvSetupChkLine(out, "api")
			if code != 1 || !strings.Contains(line, "certificate") || strings.Contains(line, "unreachable") {
				t.Errorf("exit %d, api line %q; want a failure that names the certificate and does not say unreachable", code, line)
			}
		})
	}
}
