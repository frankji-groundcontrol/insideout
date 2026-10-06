# 2026-10-07 — Live TTY walk of `insideout setup` (PTY-driven)

## Trigger

The setup TUI plan's verification goal: every "works" claim backed
by a live TTY run — the interactive huh form had only its refusal
branch tested. A human-terminal walk needs hands; a PTY walk proves
the same code path (real character devices on both ends, real
keystrokes, real rendering) and is reproducible.

## Method

macOS `/usr/bin/script -q <log> <binary> setup` allocates a genuine
PTY; keystrokes are fed with `sleep`-spaced `printf` (Enter to
accept defaults, `2` + Enter for menu picks, typed values for
fields). Credentials: clearly-fake for the remote login (real ones
are user-owned and must never be invented); a real scratch
PostgreSQL 18 DSN for the local walk (initdb on a scratch port,
roles per `docker/postgres-init/001-create-app-role.sh`).

## Results

- **Remote branch, real PTY, hosted API**: form rendered end-to-end
  (mode → URL with hosted default → method → email → password → MCP
  confirm); the typed password **never appears in the PTY log**
  (hidden capture confirmed on a real terminal); the flow probed the
  hosted API (passed — 401 = expected) and reached the real login
  endpoint, which rejected the fake credentials (`unauthorized`);
  **no config file was written on failure** (correct). Exit 1 with
  an honest error — the walk's intended stopping point without real
  credentials.
- **Local branch (own URL), real PTY, scratch PostgreSQL 18**:
  mode → local → own → hidden DSN paste → pgx probe pass → **all 31
  embedded migrations applied** (31 `.sql` files on disk; an earlier
  exercise printed 23 from a stale count — 31 is the file-count
  truth) → summary gate (`2 set · required satisfied`) → boot offer
  declined (default N) → **exit 0**. No secret ever echoed.
- **Bug found by the walk and fixed**: after a *local* success the
  wrapper printed the *remote* success line ("signed in and saved
  the config") — wrong on both facts for local. Fixed to print only
  when `SignedIn` (`cli_setup.go`); retested (cmd package green).
- A re-run of the walk failed on **my provisioning slip** — the
  scratch database was recreated owned by `setup_test` instead of
  `insideout_owner`, reproducing exactly the precondition in
  [the role-provisioning learning](../learning/2026-10-07-own-url-role-provisioning-precondition.md);
  with correct provisioning (first run) the walk passes.

## Scope

Any interactive CLI whose live-TTY behavior must be verified without
hands — especially hidden-input claims (grep the PTY log for the
secret) and success/failure side effects.

## When to apply

`script -q log binary` + timed keystrokes; assert on the log (field
rendered, secret absent, error text) and on side effects (files
written or not, DB migrated). Distinguish provisioning slips from
code defects before blaming the tool — re-check the environment
against the documented preconditions.
