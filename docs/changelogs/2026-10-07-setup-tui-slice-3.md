# 2026-10-07 — Setup TUI slice 3: `--check`, and the docs teach `insideout setup`

**Scope:** slice 3 of the setup TUI plan plus the per-slice docs
surfaces. Plan:
[`docs/plans/2026-10-07-setup-tui.md`](../plans/2026-10-07-setup-tui.md).

## What changed

- **`insideout setup --check [--llm]`**
  (`server/cmd/insideout/cli_setup_check.go`): re-runs the health
  probes against the current configuration — API base + token via
  clientconfig (env > config file > hosted default), database DSN,
  `GITHUB_TOKEN`, and `INSIDEOUT_LLM_API_KEY` from the repo `.env`
  when a checkout is present (`INSIDEOUT_ENV_FILE` honored).
  Offline-safe by design: unreachable targets report `skip` with a
  reason and do not fail the run; configured targets that answer
  wrongly (a rejected GitHub token, a 5xx, a bad LLM key shape)
  fail it; a configured-but-down database fails. `--llm` adds one
  live provider request (`GET /models`, free) — skipped with a
  message for non-OpenAI schemas, where only the shape check runs.
- **Docs surfaces now teach the front door**:
  [installation.md](../usage/installation.md) (interactive remote
  path, `--local` with the own-URL role precondition, the
  re-verify section, agent steps ending in `--check`),
  [cli-mcp-and-guidance.md](../usage/cli-mcp-and-guidance.md)
  (setup block; Go version corrected to 1.25+),
  the [install skill](../../skills/insideout-setup/SKILL.md) (steps
  3/6 prefer `insideout setup`; `--check` is the first gate),
  `README.md` (bilingual wizard block), and root `SKILL.md`.

## Verification

- Tests (`cli_setup_check_test.go`): API ok against an httptest
  server, API skip on unreachable, db skip without a DSN, db fail
  when configured and down, LLM skip/bad-shape/non-openai-live
  lanes, flag exclusivity (`--check` vs `--hosted`/`--local`,
  `--llm` without `--check`).
- Full gate: `go vet ./...` clean, `go test ./...` **25 packages
  ok, 0 failures**, `gofmt` clean; Python env batteries 47+37
  (untouched).
- **Live run** (repo root, real config): `setup --check` →
  `api ok (hosted, authentication required)`, `db ok`, `github skip
  (no token)`, `llm shape ok`, exit 0.

## Operator notes

Nothing deployed. `--check` makes no writes and sends no secrets
beyond the probes themselves (GitHub/LLM keys to their own
providers, as their tools already do). Remaining from the plan: the
human-interactive huh walk, the bundled-Docker live run (needs a
Docker host), and optional slice 4.
