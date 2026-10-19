package api

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
)

// forwardedIgnoredWarned marks the Servers that already logged an ignored
// X-Forwarded-For. Server cannot grow a field without touching server.go, so
// the once-per-Server flag lives here, like singletonGuards.
var forwardedIgnoredWarned sync.Map // map[*Server]struct{}

// clientIP is the address the auth throttle and the resend cooldown key on.
//
// With no trusted proxy (INSIDEOUT_TRUSTED_PROXY_HOPS=0, the default) it is
// the TCP peer: nothing a client sends can be believed, and keying on the tail
// of a header the client writes itself gave an attacker a fresh bucket per
// request (rv4-SEC-02).
//
// Behind N trusted proxies each one appends the address it saw to the right of
// whatever the client sent, so the Nth entry from the right is the one the
// outermost trusted proxy appended — the client as that proxy saw it — and
// every entry to its left is client-controlled and ignored. Which proxy appends
// is NGINX_XFF in client/nginx.conf. Hosted, the chain is browser → Railway
// edge → nginx → server and NGINX_XFF stays $http_x_forwarded_for: the edge
// appends, nginx passes the chain through (REL-13), one hop. Compose sets
// NGINX_XFF to $proxy_add_x_forwarded_for, so nginx itself is that one hop.
// Fewer entries than hops, or an entry that is not an IP address, falls back
// to the peer: a shared bucket is the safe-closed answer when the count and
// the chain disagree.
//
// ponytail: IPv6 clients are keyed by full address; a /64 prefix would stop a
// host rotating inside its own block. Add it if v6 abuse shows up.
func (s *Server) clientIP(r *http.Request) string {
	hops := 0
	if s.cfg != nil {
		hops = s.cfg.TrustedProxyHops
	}
	switch {
	case hops > 0:
		if ip, ok := forwardedClient(r.Header.Values("X-Forwarded-For"), hops); ok {
			return ip
		}
	case r.Header.Get("X-Forwarded-For") != "":
		s.warnForwardedIgnored()
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// forwardedClient returns the hops-th entry from the right of the
// X-Forwarded-For header lines. A repeated header is one list (RFC 9110
// §5.3), so every line counts, not just the first: a proxy may append a line
// of its own. Empty entries carry no address and are skipped.
func forwardedClient(lines []string, hops int) (string, bool) {
	var entries []string
	for _, part := range strings.Split(strings.Join(lines, ","), ",") {
		if e := strings.TrimSpace(part); e != "" {
			entries = append(entries, e)
		}
	}
	if hops > len(entries) {
		return "", false
	}
	addr, err := netip.ParseAddr(entries[len(entries)-hops])
	if err != nil {
		return "", false
	}
	return addr.Unmap().String(), true
}

// warnForwardedIgnored says once per Server that a request carried
// X-Forwarded-For while no proxy is trusted. Behind a proxy that is a
// misconfiguration with a loud symptom (every client shares the proxy's
// throttle bucket), so it gets a log line instead of a quiet fallback.
func (s *Server) warnForwardedIgnored() {
	if s.log == nil {
		return
	}
	if _, dup := forwardedIgnoredWarned.LoadOrStore(s, struct{}{}); dup {
		return
	}
	s.log.Warn("X-Forwarded-For ignored: INSIDEOUT_TRUSTED_PROXY_HOPS is 0, so the auth throttle and resend cooldown key on the TCP peer — " +
		"behind a reverse proxy every client then shares the proxy's bucket; set it to the number of trusted proxies (docs/usage/deployment.md)")
}
