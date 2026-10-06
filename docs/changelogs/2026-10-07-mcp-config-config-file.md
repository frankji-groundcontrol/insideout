# 2026-10-07 — mcp-config.sh reads the setup config file

**Scope:** one script — `scripts/mcp-config.sh` — landed early from
the setup-TUI plan's join list because it is disjoint from every
workflow slot. Plan:
[`docs/plans/2026-10-07-setup-tui.md`](../plans/2026-10-07-setup-tui.md)
(slice 1 checklist item).

## What changed

- `mcp-config.sh` now reads the 0600 config file that `insideout
  setup` will write (`<OS config dir>/insideout/config.json`) as a
  source for both the API base and the token. Resolution precedence:
  **flags > environment > config file > hosted default**, matching
  the `internal/clientconfig` contract the setup TUI plan specifies
  (process env beats the file).
- The default path mirrors `os.UserConfigDir` per platform
  (`~/Library/Application Support` on Darwin, `XDG_CONFIG_HOME`/
  `~/.config` otherwise); `INSIDEOUT_CONFIG_FILE` overrides it as a
  test affordance.
- Malformed or missing config files degrade silently to the previous
  behavior; `python3` absence does too. The no-token hint text now
  mentions `insideout setup`.

## Verification

`bash -n` plus five live cases against a temp config file holding
fake values only: config supplies both fields; `--api`/`--token`
flags beat config; `INSIDEOUT_TOKEN` env beats config; a malformed
file degrades to defaults with no token line; no config and no env
emits no `INSIDEOUT_TOKEN` key (so the MCP process still signs in
via the `auth_*` tools).

## Operator notes

Nothing to deploy. Behavior is unchanged for anyone without a config
file. When the Go config package lands, keep the JSON field names
(`api_base`, `token`) and the precedence in sync with this script.
