# This machine's sandbox can block exec of freshly built binaries

**Learned** 2026-08-24 during the auth-parity live exercise
([changelog](../changelogs/2026-08-24-auth-parity-cli-mcp.md)); same
family as the
[proxy/localhost lesson](2026-08-24-gated-integration-tests-rerun-collisions.md)'s
"environment first, conclusion second" posture.

## The lesson

For part of the evening, any **newly built** binary (a plain
`fmt.Println` hello-world included) hung on exec — no output, no flag
error, `timeout` killed it — while binaries that had already run once
kept working. Not the code, not the libraries: even `--bogusflag`
( instant flag-package exit path) hung, proving the process never
reached `main`.

## When to apply

A freshly `go build -o …` binary hangs before producing any output:
don't debug the program. Confirm with a trivial hello-world build, then
switch to one of the working exec paths observed that night:

1. `go run ./pkg` — cache-path children execed fine all evening.
2. In-test verification — `go test` binaries always ran; e.g. the MCP
   server gained a stdio-level integration test that exercises the same
   protocol without execing a separate binary.

Also: kill by the **child's** PID. `pkill -f 'cmd/insideout'` matches
the `go run` parent's command line, not the cache-path child, which
survives and keeps the port (and its log fd may point at a deleted
file — restart it to get readable logs back).

## Evidence

`/tmp/mcptest/t2` (hello-world) hung; `/tmp/insideout-cli` (built and
first run earlier) ran; the same MCP code verified end-to-end via
`internal/mcpserver` tests and via `go run` over real stdio.
