// Command insideout-mcp is the MCP projection of the InsideOut product:
// one tool per CLI verb over the same /api/v1 contract. The server and
// its tools are built (and tested) in internal/mcpserver; this main only
// wires stdio. The API base and token resolve like the CLI's
// (clientconfig.Resolve): --api, INSIDEOUT_API, the saved login, then the
// hosted API; INSIDEOUT_TOKEN or the saved token. Or sign the process in at
// runtime with the auth_* tools — a login tool's token is process state,
// never conversation output.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/server"

	"github.com/frankji-groundcontrol/insideout/server/internal/mcpserver"
)

func main() {
	api := flag.String("api", "", "API base URL (default: INSIDEOUT_API, the saved login, then the hosted API)")
	flag.Parse()

	if err := server.ServeStdio(mcpserver.Build(*api)); err != nil {
		fmt.Fprintln(os.Stderr, "insideout-mcp:", err)
		os.Exit(1)
	}
}
