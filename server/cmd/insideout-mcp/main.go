// Command insideout-mcp is the MCP projection of the InsideOut product:
// one tool per CLI verb over the same /api/v1 contract. The server and
// its tools are built (and tested) in internal/mcpserver; this main only
// wires stdio. Configure with INSIDEOUT_API (default: hosted API) and
// INSIDEOUT_TOKEN, or sign the process in at runtime with the auth_*
// tools — a login tool's token is process state, never conversation
// output.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/server"

	"github.com/frankji-groundcontrol/insideout/server/internal/mcpserver"
)

const defaultAPIBase = "https://server-production-9c338.up.railway.app/api/v1"

func main() {
	api := flag.String("api", envOr("INSIDEOUT_API", defaultAPIBase), "API base URL")
	flag.Parse()

	if err := server.ServeStdio(mcpserver.Build(*api)); err != nil {
		fmt.Fprintln(os.Stderr, "insideout-mcp:", err)
		os.Exit(1)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
