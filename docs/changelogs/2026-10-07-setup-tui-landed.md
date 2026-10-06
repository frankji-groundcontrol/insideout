# 2026-10-07 — Setup TUI: live walks verified and the series landed

**Scope:** the closing slice of the setup TUI
([plan](../plans/2026-10-07-setup-tui.md)) — the two remaining
objective gates exercised, one real bug found and fixed, and the
whole series committed in per-task local commits. Follow-up to
[the interactive-local wiring](2026-10-07-setup-tui-interactive-local.md).

## What was done

- **Live TTY walks of both branches, driven through a PTY**
  ([practice](../practices/2026-10-07-live-tty-walk-insideout-setup.md)):
  remote reached the real hosted API and failed honestly on
  clearly-fake credentials (secret absent from the PTY log, no
  config written); local own-URL against a scratch PostgreSQL 18
  ran to **exit 0** — hidden DSN, **31 migrations** (correcting the
  earlier "23": the on-disk file count is the truth), summary gate,
  boot declined.
- **Bug found by the walk, fixed:** after a *local* success the
  wrapper printed the *remote* success line ("signed in and saved
  the config") — now gated on `res.SignedIn` in `cli_setup.go`;
  cmd package retested green.
- **Bundled-Docker gate closed by acceptance** (runtime direction,
  2026-10-07): no Docker on this host; the own-URL scratch-PostgreSQL
  run stands in. The bundled lane stays unit-covered
  (skip-if-docker test); any future Docker host can confirm with
  `insideout setup --local --postgres bundled`.
- **Landed as six per-task local commits** (no push): deps →
  hosted-API default → install skill → setup slot packages → the
  setup join (`insideout setup` + `--check`) → docs and records.
  Shared index READMEs were staged as filtered copies so the
  app-icon slice's rows stay uncommitted with its files.

## Verification

`go vet` clean; `go build ./...` ok; `go test ./...` all packages
green on the exact committed tree; Python env batteries untouched
(green earlier the same day). Known pre-existing wart left alone:
six files under `internal/agent`, `internal/api`, `internal/store`
fail `gofmt -l` at HEAD itself — unrelated to this series.

## Operator notes

Nothing deployed. Deliberately left: slice 4 (release pipeline,
optional, own gate), the real-credential remote sign-in walk
(user-owned), and the app-icon slice's own commits.
