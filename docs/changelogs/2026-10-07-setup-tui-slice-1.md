# 2026-10-07 — Setup TUI slice 1: the setup subcommand, built by parallel workflow and joined

**Scope:** slice 1 of the setup TUI plan — `insideout setup` remote
mode + client config persistence — implemented by dynamic-workflow
fan-out (slots A–E) and joined in-session. Plan:
[`docs/plans/2026-10-07-setup-tui.md`](../plans/2026-10-07-setup-tui.md).

## What changed

- **Slot packages** (built in parallel, per-slot independent review
  with bounded fix rounds): `internal/clientconfig` (0600 config,
  env>file precedence), `internal/setup/flow` (pure state machine:
  URL validation, probe, password/code/magic sign-in, persistence),
  `internal/setup/ui` (huh form, ErrNoTTY refusal),
  `internal/setup/catalog` + `internal/setup/envwrite`
  (`.env.example` form rules + comment-preserving 0600 writer, for
  slice 2), `internal/setup/probe` (API-401, pgx ping with DSN
  scrubbing, GitHub, LLM shape), and hidden terminal reads for every
  secret prompt in `cli_auth.go` (`x/term`; the reviewer caught the
  ReadPassword newline-consume trap —
  [learning](../learning/2026-10-07-xterm-readpassword-consumes-newline.md)).
- **Join** (`cmd/insideout/cli_setup.go` + wiring): `setup`
  registered before `config.Load`; flags mirror `setup.sh`
  (`--hosted` works now; `--local --postgres bundled|own [--no-start]`
  keeps the contract and points at the script until slice 2);
  prober/persister adapters; the shared login client rebinds to the
  probed URL. All 19 CLI env reads (API base + token) and the MCP
  server's `sessionToken` now resolve through `clientconfig`.
- **Two defects the join surfaced, both fixed red→green:**
  1. `clientconfig.Load` applied env overrides only when a config
     file existed — the early `IsNotExist` return skipped them, so
     every env-pointed, file-less caller (the common case) got empty
     credentials. New test pins env-wins-without-file.
  2. A workflow slot reverted `cli.go`'s uncommitted
     `defaultAPIBase = apiclient.DefaultBase` to HEAD's localhost
     literal (collateral on a dirty worktree —
     [learning](../learning/2026-10-07-parallel-slots-dirty-worktree.md)).
     Restored; `setup --hosted` prints the hosted API again.

## Verification

- Authoritative whole-tree gate (absolute paths; the workflow's own
  gate had a cwd defect —
  [learning](../learning/2026-10-07-workflow-gates-need-absolute-paths.md)):
  `go vet ./...` clean; `go test ./...` **25 packages ok, 0
  failures**; Python env batteries **47 + 37 passed** direct-run;
  `gofmt -l` clean over all touched packages.
- Live: hosted `GET /api/v1/me` → **401** (the pass signal);
  `setup --hosted` prints the hosted base; piped stdin → ErrNoTTY
  exit 2; invalid flag combinations exit 2; `--local --postgres
  bundled --no-start` prints the working script command.
- Dual-mode binary: **19,760,802 bytes** (baseline 17,827,666;
  +1.93 MB from bubbletea/huh — under the escape-hatch threshold;
  the plan's size record updated).

## Operator notes

Nothing deployed; worktree only. The CLI now also reads
`<OS config dir>/insideout/config.json` (0600) as a credential
source below the environment — machines that set `INSIDEOUT_TOKEN`
are unaffected. The interactive huh form still needs a real human
terminal walk (notCovered). Next: slice 2 (local-mode assembly),
slice 3 (`--check`).
