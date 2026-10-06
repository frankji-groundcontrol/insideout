package apiclient

// DefaultBase is the hosted API. The CLI and the MCP server call it
// when INSIDEOUT_API is unset, so a finished install can reach the
// service without any environment variables. A local or self-hosted
// server sets INSIDEOUT_API to that server's /api/v1 URL.
const DefaultBase = "https://insideout.yalotein.net/api/v1"
