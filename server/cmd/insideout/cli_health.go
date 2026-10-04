package main

// `insideout health` — container HEALTHCHECK probe: GET the server's own
// /healthz over loopback, exit 0 on 200 and 1 otherwise. The distroless
// image (server/Dockerfile) ships no shell or curl, so the binary
// self-probes. Dispatched before config.Load on purpose: a probe must not
// depend on DATABASE_URL and friends — a misconfigured server fails the
// probe through the HTTP layer instead.

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

func runHealthCommand() int {
	url := "http://127.0.0.1:" + healthPort() + "/healthz"
	client := &http.Client{Timeout: 3 * time.Second}
	res, err := client.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "health: %v\n", err)
		return 1
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "health: %s returned %s\n", url, res.Status)
		return 1
	}
	return 0
}

// healthPort mirrors config.listenAddr's precedence — INSIDEOUT_ADDR, then
// PORT, then :8080 — reduced to just the port, without importing the full
// config loader.
func healthPort() string {
	addr := os.Getenv("INSIDEOUT_ADDR")
	if addr == "" {
		addr = os.Getenv("PORT")
	}
	if addr == "" {
		return "8080"
	}
	if i := strings.LastIndex(addr, ":"); i >= 0 && i+1 < len(addr) {
		return addr[i+1:]
	}
	return addr
}
