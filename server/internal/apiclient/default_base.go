package apiclient

// DefaultBase is the hosted API. The CLI and the MCP server call it when no
// --api flag, INSIDEOUT_API or saved login names another server, so a finished
// install can reach the service without any environment variables.
//
// It is a var so a release can retarget the shipped binaries at link time
// (a fork or a second deployment needs no source edit):
//
//	go build -ldflags "-X github.com/frankji-groundcontrol/insideout/server/internal/apiclient.DefaultBase=https://api.example.org/api/v1"
//
// -X replaces only a package-level string var initialised with a literal, and
// scripts/default-api.sh reads this line, so keep the declaration on one line.
var DefaultBase = "https://insideout.yalotein.net/api/v1"
