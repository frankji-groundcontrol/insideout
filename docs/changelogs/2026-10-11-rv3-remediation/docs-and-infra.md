# Docs and infrastructure — review-3 remediation (waves 1, 3, 4)

Commits: wave 1 `5c8f54f..c89faea`, wave 3 `ce8e173..b056e30`,
wave 4 `b056e30..4464330`. Register rows each change closes are named.

## CI

- **The server image is built in CI** (ARCH-01, `1db90d8`): the Dockerfile's
  floor is certified on every push, and it builds on `golang:1.26-alpine` —
  the review's "lower go.mod to 1.25" was not viable because
  `golang.org/x/sys` and `x/term` already declare `go 1.26.0`
  (`04a0dfd`, floor check `c5459dd`).
- **The upgrade path is rehearsed** (TEST-01, `941fa88`): a CI lane replays
  the deployed schema with pinned production-shaped old data (users,
  workspaces, ideas, PRDs, updates, invite codes including a legacy short
  code), migrates to HEAD and runs smoke against it — so the hosted apply is
  not the migrations' first meeting with populated rows.
- **Smoke has a hosted dispatch mode** (REL-07, `f63f178`) that probes the
  public deployment, and **provider legs run for real when keys are
  configured** (FEAT-03 half, `5800fb1`); the probes are split into
  fragments so the script stays modular.

## Truth checks (all red-test-first)

- `scripts/tests/check_docs_truth.py` (wave 3, `600842b`) fails on nine
  stale doc claims it pinned at birth: backup-restore states the unscheduled
  reality, the README quick start leads with paths that work today, the MCP
  tool count points at `tools/list`, the backend layout tree matches the
  tree, production serves a release behind `main` (ARCH-04, ARCH-05,
  COMP-02, COMP-04).
- The Go-floor check extends to installer prose (`add6eee`) — all five texts
  say 1.26 with go.mod.
- The **promise-gap check** (`06dc239`) fails while PRODUCT.md promises
  parked slices without records; the **contribution-surface check**
  (`20c8ac5`) fails while CONTRIBUTING/SECURITY doors are missing.
- A **line-budget scan** guards the server tree (`095427c`), and the
  client's API paths are pinned to the server's registered routes
  (`0cfc03a`) so the two surfaces cannot drift apart.

## Records

- **The promise register** (COMP-01, `575d969` +
  [50-promise-register.md](../../review/2026-10-10-multirole-3/50-promise-register.md)):
  every PRODUCT.md promise is either implemented, parked with a dated issue
  record, or named as owner-blocked — and PRODUCT.md's status section links
  the register and the parked set.
- **Nine gap-issues records** (wave 4, `7c1b524`..`0ecd938`): scoped external
  feedback invitations, Product Branch/Merge + Decision Log, persona
  grounding labels, the facts trust layer, roadmap lenses, dependency edges
  and rebaseline, the Focus surface, tokenized audience-share link,
  working-version autosave — each a dated slice record under
  `docs/issues/`, indexed in its README.
- **CONTRIBUTING.md and SECURITY.md doors** (USER-05 code half, `6c211d0`),
  linked from the learn-more table; the install one-liner is demoted under
  the release conditional (PM-02 code half, wave 3).
- **Env skeleton completeness** (TEST-07, `7b852e4`): every
  compose-interpolated `INSIDEOUT_*` variable must appear in the skeleton —
  `INSIDEOUT_GH_PRIVATE_KEY_FILE` was the first catch.

## Verification

See [verification.md](verification.md) for the full gate run and the
adversarial contract that accepted every cluster.
