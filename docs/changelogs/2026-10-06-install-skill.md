# 2026-10-06 — Install skill for agent-guided from-scratch setup

**Scope:** a new repository-shipped skill, `skills/insideout-setup/`,
that any coding agent can load and follow to take a bare **macOS** or
**Ubuntu** machine to a verified InsideOut installation — plus pointer
wiring. Docs + skill files only; no service or client code, nothing to
deploy. Plan:
[`docs/plans/2026-10-06-install-skill.md`](../plans/2026-10-06-install-skill.md).

## What changed

- **`skills/insideout-setup/SKILL.md`** — the installer agent's
  contract: platform gate (macOS/Ubuntu only, explicit stop elsewhere),
  the non-negotiables (never read `.env`, one question before
  installing, announced system changes, non-interactive flags,
  idempotent re-runs, human-only sign-in), destination table
  (A hosted · B own server · C full development), the seven flow steps
  (platform → modules → clone → path → MCP registration → key store →
  verification gates → record), and the recording requirements (host
  record under `hosts/`, dated changelog when repo files change, final
  report to the user).
- **`skills/insideout-setup/references/modules.md`** — per-module
  install steps with macOS and Ubuntu variants and a verify command
  each: base tools, Homebrew, Go (minimum 1.25 per `server/go.mod`;
  Ubuntu apt explicitly flagged as often too old with the tarball
  fallback), Docker + Compose v2, PostgreSQL 14+ (own-instance path
  only), Flutter 3.44.x + Chrome (pinned 3.44.0 per CI/`client/Dockerfile`),
  and InsideOut's own components (installer, `.env`, key store,
  migrations, MCP emitter).
- **Pointers**: root `SKILL.md` (Setup section links the install skill;
  stale "Go 1.22+" corrected to 1.25+), `README.md` (bilingual pointer
  in the install section; same Go-version fix),
  [docs/index.md](../index.md) (For agents),
  [docs/usage/README.md](../usage/README.md) (table row).

## Why

The install surface already worked
([installation.md](../usage/installation.md) and the scripts), but the
order, the platform differences, and the recording duty lived only in
the reader's head. The root `SKILL.md` covers *driving* the product;
nothing covered *building the machine*. External users bring their own
agents, so the skill ships in the repo and is discovered through the
routers rather than any harness-specific directory.

## Verification

- Skill frontmatter checked: name matches the directory, description
  under the 1024-char limit with "Use when" triggers; skill files
  165/130 lines — under the ~350-line modularity bar.
- Every referenced path checked to exist (scripts, docs, hosts
  exemplars, the practice record); script flags quoted in the skill
  match the scripts themselves (`setup.sh --hosted|--local --postgres
  bundled|own [--no-start]`, `install.sh [dir]`, `env.sh
  init|edit|check`, `keys.py init|fill|audit`, `mcp-config.sh -q`).
- Flutter 3.44.0 archive URLs and the Go download scheme fetched live
  (HTTP 200); current Go stable checked on `go.dev/dl` so the skill
  pins "newest stable" instead of a stale patch.
- Engineering review of the plan: skipped and recorded in the plan —
  docs-only deliverable; the review checklist (privacy, modularity,
  link integrity, honest verification) applied instead.

## Operator notes

Nothing to deploy; machines that already work are unaffected. When
onboarding a new machine with this skill, the record lands under
`hosts/<hostname>/` — stage those files individually (never
`git add .`), and keep the `.secrets/` folders untracked as before.

## Related records

- Open follow-up: the Ubuntu path has not run on real Ubuntu hardware —
  [issue](../issues/2026-10-06-install-skill-ubuntu-unverified.md).
- Lesson from this change's verification: relative links are
  depth-sensitive; resolve them mechanically —
  [learning](../learning/2026-10-06-relative-links-are-depth-sensitive.md).
