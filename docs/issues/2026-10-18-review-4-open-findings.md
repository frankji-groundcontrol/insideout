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

## Highs and the red test (priority order)

| Register ID | Sev | Priority | Owner | Plan row | Done when |
|---|---|---|---|---|---|
| rv4-USER-01 | High | P0 | engineering | 1.1 | On a fresh invitation-only database, `promote-admin` creates or promotes the first admin and admin settings answer 200; the red test is in the suite. |
| rv4-TEST-01 | High | P0 | engineering | 1.5 | `release.yml` runs `go test ./... -race` (or requires a green `ci` run for the tag SHA) before it publishes. |
| rv4-L1 (ARCH-04, QUAL-05, PM-02) | Medium | P0 | engineering | 1.3, 1.4 | `TestProjectQuietFlagged` passes on any date; `ci.yml` has a weekly schedule; HANDOFF cites a dated SHA. |
| rv4-FULL-02, rv4-USER-04 | High | P1 | engineering | 2.2 | A CLI login older than the access TTL still runs `whoami`; plain `login` persists its token. |
| rv4-ARCH-01 | High | P1 | engineering | 2.1 | Each verb honours `--api` over config; the MCP binary reads the config `api_base`; one resolver remains. |
| rv4-FEAT-01, rv4-AIDEV-04 | High | P1 | engineering | 2.5 | An agent-scoped token is refused at `commit` and `proposal_decide` server-side; the MCP profile omits `signup_mode`, `invite_*` and deletes. |
| rv4-FEAT-02 | High | P1 | engineering | 2.3 | Accepting a proposal with apply twice leaves one set of roadmap nodes; apply errors surface as 500. |
| rv4-FULL-01 | High | P1 | engineering | 2.4 | Web Expand with AI creates children; a real-shape contract test pins the response. |
| rv4-COMP-02, rv4-COMP-01 | High, Medium | P1 | engineering | 1.16, 1.17 | A deadline picked as 2026-10-18 still reads 2026-10-18 under `TZ=Asia/Shanghai`; a 67-character Chinese title is accepted on every field that has a character limit. |

## Owner decisions (no engineering action until decided)

| Register ID | Decision | Default until decided |
|---|---|---|
| rv4-PUB-01, rv4-PM-11 | Rotate the hosted admin address after scrubbing it from five docs. | Scrub the docs (plan row 0.1); keep the address. |
| rv4-PUB-05 | Rotate or disable the legacy Supabase publishable key and confirm the Data API exposes nothing. | Delete `.gitignore:72` (plan row 0.2); key untouched. |
| rv4-PUB-06 | Accept the institutional author identities in history, or rewrite history (force-push of `main`). | Accept; do not rewrite. |
| rv4-PUB-10 | Ship `hosts/` as an example tree and ignore the real one. | Scrub paths to placeholders only. |
| rv4-PUB-02, rv4-DATA-06 | Privacy notice, retention and erasure posture for the hosted service. | Write `PRIVACY.md` first (plan row 3.5). |

## What closes this record

Rows 0.1 to 2.6 of the plan are done and their checks pass, or each remaining
row has its own dated record. The Medium and Low rows stay in the register and
the plan; they do not get issue files unless a wave opens one.
