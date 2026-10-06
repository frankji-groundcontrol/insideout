# 2026-10-07 — Setup TUI: the interactive form's local branch wired to the assembly

**Scope:** small follow-up completing the interactive experience —
the huh form's "Server on this machine" choice now runs the slice-2
assembly instead of refusing. Plan:
[`docs/plans/2026-10-07-setup-tui.md`](../plans/2026-10-07-setup-tui.md).

## What changed

- `flow.Deps` gains a `LocalSetup func(ctx, postgres) error` join
  hook — `flow.Run` stays pure and still refuses local mode; only
  `ui.Run`'s local branch consumes the hook.
- `ui.Run` local branch: one huh select (bundled Docker / own URL),
  then the hook. The own-URL DSN paste stays inside the local
  assembly's hidden read — one hidden-input path, not two.
- `cli_setup.go` wires the hook to `runLocalSetup(postgres, false)`.

## Verification

`go vet ./...` clean; `go test ./...` **25 packages ok, 0 failures**
(flow's purity tests unchanged and green); binary 19,866,802 bytes;
headless smoke: piped `insideout setup` still refuses with the
ErrNoTTY message. The interactive walk itself needs a human at a
real terminal — the remaining objective gate, recorded on the board.

## Operator notes

Nothing deployed. To complete the live TTY walk: run
`insideout setup` (or the freshly built binary) in a real terminal —
choose either branch; secrets are hidden as typed; remote saves the
0600 config, local drives the `.env`/migrate/boot flow.
