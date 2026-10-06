# Install skill — agent-guided setup from scratch (2026-10-06)

Goal: a repository-shipped skill, `skills/insideout-setup/`, that any
beginner's coding agent can load and follow to take a bare **macOS** or
**Ubuntu** machine from nothing to a verified InsideOut installation —
every required module with its own spelled-out install steps, the
hosted/own-server decision, environment and key-store setup,
verification gates, and a written installation record under `hosts/`.

Scope: docs + skill files only. No service or client code, nothing to
deploy.

## Why

- The install surface already exists and works
  ([installation.md](../usage/installation.md), `scripts/install.sh`,
  `scripts/setup.sh`, `scripts/env.sh`, `scripts/keys.py`) but it is
  spread across docs; a beginner's agent must assemble the order
  itself. The root [SKILL.md](../../SKILL.md) covers *driving* the
  product, not *building the machine* it runs on.
- Platform support is assumed, not stated: only macOS and Ubuntu are
  expected to work; the skill must say so and stop elsewhere.

## Checklist

- [x] Survey the install surface (scripts + usage docs + hosts records).
- [x] `skills/insideout-setup/SKILL.md` — decision flow (hosted vs own
      server vs full dev), guardrails, flow steps, verification gates,
      recording requirements.
- [x] `skills/insideout-setup/references/modules.md` — per-module
      install steps for macOS (Homebrew) and Ubuntu (apt), each with a
      verify command: base tools, Go 1.25+, Flutter 3.44.x, Docker +
      Compose v2, PostgreSQL 14+, plus InsideOut's own components
      (CLI/MCP binaries, `.env` + key store, migrations).
- [x] Wire pointers: root `SKILL.md` (Setup section), `README.md`
      (install section), `docs/index.md` (For agents),
      `docs/usage/README.md` (table row).
- [x] Changelog `docs/changelogs/2026-10-06-install-skill.md` + index row.
- [x] Verify: skill frontmatter/description limits, every referenced
      path exists, referenced script flags match the scripts
      (`setup.sh -h`, `install.sh` usage, `env.sh check`, `keys.py`
      subcommands), plan/board/changelog/index cross-linked.

## Verification results (2026-10-06)

- Frontmatter: name matches the directory; description 608 chars
  (limit 1024) with "Use when" triggers. SKILL.md 183 lines,
  modules.md 208 lines — both under the ~350-line bar.
- Relative-link check across all ten touched files: zero broken (two
  depth errors caught and fixed during verification).
- Flags cross-checked against the scripts: `setup.sh --hosted |
  --local --postgres bundled|own [--no-start]`, `env.sh
  init|edit|check|list|redact|propagate`, `keys.py init|fill|audit`,
  `mcp-config.sh -q` all match.
- Live checks: Flutter 3.44.0 archive URLs fetched (HTTP 200); Go
  stable list pulled from `go.dev/dl` (skill pins "newest stable",
  not a stale patch); `./scripts/env.sh check` run on this host —
  4 ok, 0 warnings, 0 failures, no values printed.
- The Ubuntu apt/Go claim is written as a check-then-choose
  (`apt policy golang-go`, tarball fallback) rather than a hardcoded
  version assertion, so it survives new Ubuntu releases.

## Verification goals

- A fresh-eyes agent reading only the skill can name, in order: the
  platform check, the one question to ask, the module installs, the
  path-specific commands, the verification gates, and the record to
  write.
- No secret ever appears in a skill instruction; `.env` is never read
  by the agent.
- Links resolve; skill files stay under the ~350-line modularity bar.

## Decisions

- Skill location: `skills/insideout-setup/` (canonical, versioned with
  the repo) rather than a harness-specific dir — users bring different
  agents; discovery is wired through the root `SKILL.md` and routers,
  which every agent reads.
- Engineering review of this plan: skipped — docs-only deliverable with
  no service behavior change; the review checklist applied instead
  (privacy, modularity, link integrity, honest verification).
- Recording: reuse the existing host-record scheme (`hosts/<host>/`,
  never-absent key store) and the key-setup practice record; a dated
  changelog entry records the skill itself.
