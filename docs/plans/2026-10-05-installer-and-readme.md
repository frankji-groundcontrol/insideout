# 2026-10-05 — Install scripts + agent skill + README for the CLI/MCP surface

**Goal:** a user (or their coding agent) can install the `insideout` CLI and
`insideout-mcp` server in one step into `~/.local/bin` with PATH handled,
register the MCP server in any MCP-capable agent from one emitted snippet,
and read a root README that answers: what this is, how to install, basic
usage, where to go deeper.

**Scope guard:** no Go service/client code changes, no API/CLI/MCP verb
changes, nothing to deploy. Scripts + docs only.

## Checklist

- [x] `scripts/install.sh` — build both binaries from `server/`, install
      into `INSIDEOUT_INSTALL_DIR` (default `~/.local/bin`), idempotently
      add it to PATH via the login shell's rc file.
- [x] `scripts/mcp-config.sh` — emit the standard `mcpServers` JSON for
      `insideout-mcp` (API base + optional inline token; placeholder when
      absent) plus per-agent placement hints; pure stdout emitter.
- [x] `SKILL.md` — setup section uses the installer and the config emitter
      (an agent can self-install and self-register), manual build kept as
      the fallback.
- [x] `README.md` — rewritten around: what the project does (current
      build, not the superseded coach model), install (web / CLI+MCP via
      installer / local dev), basic usage, further-reading map. Bilingual
      style and the font attribution record preserved.
- [x] `docs/usage/cli-mcp-and-guidance.md` — Setup leads with the
      installer; manual build as fallback; MCP registration via the
      emitter. Usage README row updated.
- [x] Verification: `bash -n` both scripts; real `install.sh` run with
      sandboxed `HOME` + temp install dir (Go build exercised, rc-file
      PATH append idempotent); `mcp-config.sh` output validates as JSON.
- [x] Changelog `docs/changelogs/2026-10-05-installer-and-readme.md`;
      board row closed; `docs/HANDOFF.md` refreshed; commit + push + MBP
      fast-forward.

## Decisions

- Default install end `~/.local/bin` per user call (2026-10-05); overridable
  by first argument or `INSIDEOUT_INSTALL_DIR`. PATH is appended to the rc
  file of `$SHELL`'s family (zsh → `~/.zshrc`, bash → `~/.bashrc`) with a
  marker comment, only when the dir is not already on PATH.
- MCP registration stays a script emitter, not a CLI verb: the parity
  doctrine is API↔CLI↔MCP 1:1, and a local config emitter is not an API
  concern. Per-agent file locations are printed as hints only — the
  canonical JSON is the contract.
- Docs prose English-only; the root README keeps its bilingual zh/en
  product-facing style as established.
