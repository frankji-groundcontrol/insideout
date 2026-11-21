# Install front door

Date: 2026-10-20.

The person-facing path is [the user guide](../usage/user-guide.md):
prebuilt binaries, numbered `insideout setup`, sign-in, MCP. It does
not mention server variables or the developer key files.

`insideout setup` with no flags now asks numbered questions on a
terminal. The first question is the managed service, a service you
host, or a server on this machine. The host is checked before the next
question, and a failure stays on that step. Every question has Back
and Exit. Flags are unchanged.
The arrow-key form library is no longer a dependency.

The maintained map of those questions is
[setup paths](../usage/setup-paths.md).

Agents use [insideout-setup](../../skills/insideout-setup/SKILL.md) for
the hosted service and
[insideout-server](../../skills/insideout-server/SKILL.md) only when
someone asked to run the server.

## Checks

`go test ./internal/setup/ui/ -count=1` and `go test ./internal/modtidy/ -count=1`
passed. `go mod tidy -diff` is clean after `charmbracelet/huh` left the module.
