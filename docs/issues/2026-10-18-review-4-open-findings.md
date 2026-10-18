# Review 4 open findings — the ones that block a stranger or a release

Open, recorded 2026-10-18. [Multirole review 4](../review/2026-10-18-multirole-4/README.md)
read `main` @ `9f0b1c7` and filed 145 findings (0 Critical, 10 High, 88 Medium,
47 Low; nine Highs skeptic-confirmed, one added by the critic). The
[register](../review/2026-10-18-multirole-4/01-findings-register.md) holds every
row and the
[remediation plan](../review/2026-10-18-multirole-4/40-remediation-plan.md)
is the ordered fix list; this record tracks the nine Highs, the red test on
`main`, and the owner decisions, each linked to its register ID and plan row.

Review IDs restart per review, so cite these as `rv4-USER-01` outside the
review folder (review 3 also has a `USER-01`).

Status 2026-10-18: wave 0 of the plan (rows 0.1 to 0.4) is integrated on the
local branch `rv4/integration` (base `d53035a`, head `97d7529`, then the
records commit) and recorded in the
[remediation changelog](../changelogs/2026-10-18-rv4-remediation/index.md).
Waves 1 to 3 have not started. The Disposition column is filled by each wave's
integrator with `FIXED` (red and fix commits), `NOT_REAL`, `DEFERRED_OWNER`,
`PARTIAL` or `BLOCKED`, plus the reason.

## Highs and the red test (priority order)

| Register ID | Sev | Priority | Owner | Plan row | Done when | Disposition |
|---|---|---|---|---|---|---|
| rv4-USER-01 | High | P0 | engineering | 1.1 | On a fresh invitation-only database, `promote-admin` creates or promotes the first admin and admin settings answer 200; the red test is in the suite. | Open, wave 1. |
| rv4-TEST-01 | High | P0 | engineering | 1.5 | `release.yml` runs `go test ./... -race` (or requires a green `ci` run for the tag SHA) before it publishes. | Open, wave 1. |
| rv4-L1 (ARCH-04, QUAL-05, PM-02) | Medium | P0 | engineering | 1.3, 1.4 | `TestProjectQuietFlagged` passes on any date; `ci.yml` has a weekly schedule; HANDOFF cites a dated SHA. | Open, wave 1. The test is red on the wave-0 base too (gate note below). |
| rv4-FULL-02, rv4-USER-04 | High | P1 | engineering | 2.2 | A CLI login older than the access TTL still runs `whoami`; plain `login` persists its token. | Open, wave 2. |
| rv4-ARCH-01 | High | P1 | engineering | 2.1 | Each verb honours `--api` over config; the MCP binary reads the config `api_base`; one resolver remains. | Open, wave 2. |
| rv4-FEAT-01, rv4-AIDEV-04 | High | P1 | engineering | 2.5 | An agent-scoped token is refused at `commit` and `proposal_decide` server-side; the MCP profile omits `signup_mode`, `invite_*` and deletes. | Open, wave 2. |
| rv4-FEAT-02 | High | P1 | engineering | 2.3 | Accepting a proposal with apply twice leaves one set of roadmap nodes; apply errors surface as 500. | Open, wave 2. |
| rv4-FULL-01 | High | P1 | engineering | 2.4 | Web Expand with AI creates children; a real-shape contract test pins the response. | Open, wave 2. |
| rv4-COMP-02, rv4-COMP-01 | High, Medium | P1 | engineering | 1.16, 1.17 | A deadline picked as 2026-10-18 still reads 2026-10-18 under `TZ=Asia/Shanghai`; a 67-character Chinese title is accepted on every field that has a character limit. | Open, wave 1. |

## Gate note (wave 0, coordinator, 2026-10-18)

The Go test gate was red on exactly one test, `server/internal/api`
`TestProjectQuietFlagged` (2 subtests: "active with fresh update is not
flagged" and "active recently created with no updates is not flagged"). It
fails identically at the wave base `d53035a` and at the integration head
`97d7529` (the coordinator re-ran it under `env -i`, 2026-10-18).
`w0-privacy` changes no Go file, so wave 0 is recorded as green apart from
that proven baseline failure. Plan row 1.3 (wave 1, cluster
`w1-clock-and-runes`) fixes it; from wave 1 on the full suite must be fully
green. Source: `coordinator-w0.json` in the remediation reports directory
(outside the repository); the wave record is in the
[changelog index](../changelogs/2026-10-18-rv4-remediation/index.md).

## Owner decisions (no engineering action until decided)

Dispositions are wave 0's, SHAs are on `rv4/integration`, and the owner halves
are written out in
[the owner record](2026-10-18-rv4-owner-privacy-decisions.md).

| Register ID | Decision | Default until decided | Disposition |
|---|---|---|---|
| rv4-PUB-01, rv4-PM-11 | Rotate the hosted admin address after scrubbing it from five docs. | Scrub the docs (plan row 0.1); keep the address. | Scrub `FIXED`: PUB-01 red `c9216d7`, fix `7df9dfe`; PM-11 red `950f0ee`, fix `98f7c15`; the stems, the owner mail domain and the quoted finding left the handoff, the plans board and the review report in `3d65d50` (grep-verified: the guard cannot see truncated stems). Rotation `DEFERRED_OWNER`. |
| rv4-PUB-05 | Rotate or disable the legacy Supabase publishable key and confirm the Data API exposes nothing. | Delete `.gitignore:72` (plan row 0.2); key untouched. | `DEFERRED_OWNER`. The code half is `FIXED`: red `8b80c54`, fix `23b2617` (`.env.*` ignores the legacy env file again). The dashboard check and the key are the owner's. |
| rv4-PUB-06 | Accept the institutional author identities in history, or rewrite history (force-push of `main`). | Accept; do not rewrite. | `DEFERRED_OWNER`. Nothing to change by default; the decision is filed in `67da7b2`. The release tag's tagger address falls under this row. |
| rv4-PUB-10 | Ship `hosts/` as an example tree and ignore the real one. | Scrub paths to placeholders only. | Scrub `FIXED`: red `8b421b7`, fix `22f0665`; the OS user left two older docs and the setup skill's host sentence matches in `3d65d50`. The example tree is `DEFERRED_OWNER`. |
| rv4-PUB-12 | Whether the deployed host also becomes a placeholder in dated records (the provider ids already did). | Keep the host: the shipped default, public by design. | Ids `FIXED`: red `848ad0c`, fix `c2b5caa`, pinned to three records by a test (no tree-wide UUID rule, on purpose). The host is the owner's call. |
| rv4-PUB-02, rv4-DATA-06 | Privacy notice, retention and erasure posture for the hosted service. | Write `PRIVACY.md` first (plan row 3.5). | Open, wave 3. |

## What closes this record

Rows 0.1 to 2.6 of the plan are done and their checks pass, or each remaining
row has its own dated record. Rows 0.1 to 0.4 are recorded (wave 0); their
owner halves stay open in [the owner record](2026-10-18-rv4-owner-privacy-decisions.md).
The Medium and Low rows stay in the register and the plan; they do not get
issue files unless a wave opens one.
