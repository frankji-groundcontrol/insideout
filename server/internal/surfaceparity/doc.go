// Package surfaceparity holds the request-field parity guard (rv5-PARC-03):
// every JSON field an API write handler decodes must be reachable from the Go
// client, the CLI and the MCP tools, or be listed as a deliberate omission.
// There is no production code here; the tests are the package
// (docs/practices/2026-11-21-surface-parity-guards.md).
package surfaceparity
