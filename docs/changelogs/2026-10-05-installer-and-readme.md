# 2026-10-05 — Install scripts + agent skill + README

**Scope:** first-class installation and onboarding for the CLI/MCP
surface. Scripts + docs only; no service or client code changed, nothing
to deploy. Plan:
[`docs/plans/2026-10-05-installer-and-readme.md`](../plans/2026-10-05-installer-and-readme.md).

## What changed

- **`scripts/install.sh`** — builds `insideout` + `insideout-mcp` from
  `server/` with the local Go toolchain and installs both into
  `~/.local/bin` by default (first argument or `INSIDEOUT_INSTALL_DIR`
  overrides; the dir is normalised to an absolute path). When the install
  dir is not on PATH it appends one marker-commented `export PATH=…` line
  to the login shell's rc (`~/.zshrc` for zsh, `~/.bashrc` for bash,
  `~/.profile` otherwise); idempotent by construction — the PATH check
  and a marker grep both short-circuit the append.
- **`scripts/mcp-config.sh`** — pure stdout emitter of the standard
  `mcpServers` JSON for `insideout-mcp` (API from `--api`/`INSIDEOUT_API`
  with the CLI's local default; token from `--token`/`INSIDEOUT_TOKEN`,
  else a paste-me placeholder; binary resolved from PATH or the
  installer's `~/.local/bin` path). Prints per-agent placement hints
  (Claude Code `.mcp.json`, Cursor `mcp.json`, generic `mcpServers`
  blocks) by default, `-q` for JSON only. Minimal JSON string escaping
  for backslash and quote.
- **`SKILL.md`** — Setup now leads with `install.sh` + `mcp-config.sh`
  so an external agent can self-install and self-register; manual `go
  build` kept as the fallback.
- **`README.md`** — rewritten around the four onboarding questions: what
  the project does (the current build — inbox → PRD → roadmap → human
  Commit, GitHub evidence, three surfaces — replacing the superseded
  four-stage-coach description), how to install (installer / web /
  local dev), basic usage (web loop + a CLI taste + the agent
  vocabulary), and a further-reading table into `docs/`. The bilingual
  style and the font/attribution record are preserved.
- **`docs/usage/cli-mcp-and-guidance.md`** — Setup leads with the
  installer and the config emitter, manual build demoted to fallback;
  the MCP JSON example notes its equivalence to the emitter's output.
  Usage README row updated.

## Verification

- `bash -n` on both scripts; shellcheck unavailable on this machine.
- Real `install.sh` run twice inside a sandboxed `$HOME` (temp dir, zsh
  as `$SHELL`): first run built both binaries (Go 1.26.3), installed
  them mode 0755, appended exactly one marker line to the sandbox
  `~/.zshrc`; the rerun printed "already on PATH" and left the rc file
  untouched (marker count still 1). `insideout-mcp </dev/null` exits 0.
- `mcp-config.sh -q --api … --token 'tok"weird\test'` output validated
  with `python3 json.load` (escaping correct); default mode prints the
  hints + placeholder-token note.

## Operators / users

Nothing to do for existing deployments. Anyone with a checkout gets the
new path: `./scripts/install.sh` then `./scripts/mcp-config.sh`.
