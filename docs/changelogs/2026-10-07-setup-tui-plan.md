# 2026-10-07 — Setup TUI plan opened

**Scope:** planning only — no source, script, or skill changes. Plan:
[`docs/plans/2026-10-07-setup-tui.md`](../plans/2026-10-07-setup-tui.md);
board row added (**Pending**); a HANDOFF "what to do next" line added.

## What changed

- A dated plan for the guided two-mode setup experience — **remote**
  (hosted or self-hosted URL, sign-in via the existing password /
  emailed-code / magic-link flows) and **local** (guided `.env`,
  bundled or own PostgreSQL, health probes per step) — designed as a
  composition of pieces that already exist: `setup.sh` stays the front
  door with its flags unchanged as the agent contract; flow logic in a
  pure, unit-testable `setup_flow.py`; a thin curses renderer; the
  proven `env_tui.py` helpers extracted into a shared `tui_lib.py`.
- Four slices, smallest end-to-end first: remote front door; health
  probes in `env.sh check` (pg_isready skip-with-message pattern);
  config persistence in Go (`login --save`, 0600 file, env-var
  precedence, TDD); optional release pipeline. Slice 0 is a gate:
  engineering review of the plan before slice 1 code.
- Non-goals recorded: API-key auth (remote mode is URL + token — no
  such credential type exists), local mode from outside a repo
  checkout, new TUI frameworks, consent-less secret writes.

## Verification

- Design grounded in code read this turn: `cli_auth.go` login flows
  (stdin-driven), the `env_tui.py`/`env_catalog.py` logic-render split
  and name-based masking, `env.sh check --db`'s probe pattern,
  `setup.sh` flags, the absence of config persistence (token handling
  is `os.Getenv` across the `cli_*.go` call sites), and no release job
  in `ci.yml`.
- Relative links in the plan, board row, and HANDOFF resolve
  (mechanical check; the depth-sensitivity lesson of 2026-10-06
  applied — see
  [learning](../learning/2026-10-06-relative-links-are-depth-sensitive.md)).

## Operator notes

Nothing to deploy; nothing behaves differently. Next action when
picked up: the engineering review gate, then slice 1.
