# Wave 0, privacy scrub — review-4 remediation

Branch `rv4/w0-privacy`, `2502f46..b5fc986` (11 commits: five red/fix pairs
and the owner record), then this fragment. Findings are cited as `rv4-<ID>`
because review IDs restart per review. Rows 0.1 to 0.4 of the
[remediation plan](../../review/2026-10-18-multirole-4/40-remediation-plan.md);
the owner halves are in
[the owner-decision record](../../issues/2026-10-18-rv4-owner-privacy-decisions.md).
No value, name or address is spelled in this fragment.

## What changed

- **The redaction guard rejects any e-mail address that is not a placeholder**
  (rv4-PUB-01, red `0f5a02a`, fix `abcfb08`). `scripts/check-redaction.sh`
  knew one literal address; it now has a third rule that fails on any other
  address in any tracked text file and prints `path:line` only, so a public CI
  log can never echo the value. The hosted-admin address left two dated
  records (3 spots) and became `<admin-email>`.
- **The owner-domain account address left the 2026-10-11 record** (rv4-PM-11,
  red `39f4652`, fix `1891cdd`): `<owner-email>`, and the bounced rehearsal
  account is `<rehearsal-account>`. The red test is a whole-tree run of the
  guard, and it only goes red once the rule exists, which is the point: the
  rule found what the old guard could not.
- **The dead `.gitignore` exception is gone** (rv4-PUB-05, red `a9510b1`, fix
  `3d7080d`): `.env.*` ignores the legacy env file again
  (`git check-ignore -v` now reports the `.env.*` rule). Nothing reads that
  file; one historical record mentions it.
- **Host manifests hold placeholders** (rv4-PUB-10, red `c905aa3`, fix
  `e253503`): the `checkout` field is `<repo-checkout>` and the `ssh_alias`
  field is `<ssh-alias>` in both `host.yml`; the two dated records that quoted
  the alias value say it is a placeholder. The `name` field is untouched (it
  must equal the folder), and no `hosts/` directory was renamed, moved or
  deleted.
- **Provider ids left the dated records** (rv4-PUB-12, red `9ecc5ec`, fix
  `530eac6`): two deployment ids, one project row id and one design-tool
  project id became `<deployment-id>`, `<prior-deployment-id>` and
  `<project-id>`.
- **The owner decisions are one issue record** (rv4-PUB-06 and the owner
  halves of 0.1 to 0.3, `b5fc986`). No code, so no red/fix pair: the status is
  `DEFERRED_OWNER`.

## The address rule

`git grep -n -I -o -i -E` extracts whole addresses; `grep -v` drops the forms
that are allowed; `sed` reduces what is left to `path:line`. An address passes
when its local part is exactly `noreply`; or its domain has an `example`
label followed by a TLD (`example.com`, `mail.example.org`); or it ends in a
reserved TLD (`.example`, `.test`, `.invalid`, `.localhost`, `.local`); or it
is on `resend.dev` or a subdomain (the mail provider's sandbox); or it is
`x.com`, `x.co` or `b.co` exactly (fixture domains of the Go auth tests); or
the "domain" is an image extension, which keeps `icon@2x.png`-style asset
names out of the report. The look-alikes that must fail are pinned too:
`xnoreply`, `notexample.com`, `example.com.<real domain>`, `xx.com`,
`mail.x.com`, `notresend.dev`.

Known ceilings, recorded on purpose:

- The fixture domains are allowed tree-wide, not only in the four Go test
  files that use them. A later change that adds a fixture on another domain
  (a git SSH remote, `test.com`) trips the guard; use `example.*` names.
- It sees whole addresses. Truncated stems and bare domains are invisible, and
  so is `-I` binary content. The residue list in the owner record names the
  spots a person still has to fix.
- The scan uses extended regular expressions only, so it behaves the same
  under BSD and GNU `grep` and under `dash`. Checked locally under `/bin/dash`
  and bash 3.2 as `sh` with BSD `grep`; **pending first CI run after push**
  (`ci.yml` already runs `sh check-redaction.sh`, owned by `w1-ci-gates`).
- The rv4-PUB-10 test flags any `ssh_alias` followed by a colon and then
  anything but a `<placeholder>` in `docs/` or `hosts/`, including a code
  span; in prose, write the field name without the colon.
- No UUID rule was added. Dozens of legitimate ids live in test fixtures, seed
  SQL and the project's own config, so the rule would fire on later clusters'
  work for no gain; rv4-PUB-12 is pinned to the three named records instead.

## Design spike (rows 0.2 to 0.4, marked `dir`)

- **0.2** Two halves: delete the dead exception (code, done) and the
  dashboard check plus key rotation (owner, filed). The first removes the
  committable slot; the second is not reachable from a worktree.
- **0.3** Three options: scrub contents only (done), ship `hosts/example/`
  with the real tree untracked, or accept as is. Contents-only is reversible
  and keeps `scripts/keys.py` discovery and `scripts/tests/keys` intact. The
  example tree takes the real manifests out of git, which changes how a
  second clone sees key state, so it waits for the owner; the record carries a
  bounded fix prompt. Provider ids became placeholders but the deployed host
  stays in dated records: it is the shipped default, public by design, and the
  guard's host rule already exempts dated records.
- **0.4** Accept or rewrite. A rewrite is a force-push of `main`, a re-pointed
  release tag and a third party's consent, so the default is "accept, do not
  rewrite" and nothing here touches history.

## Left for the integrator

**Closed.** The integrator's wave-0 follow-up carried out every documentation
and comment item below, and the wave-0 section of this folder's `index.md`
lists the records; the paragraphs here are what the cluster handed over, kept
as the record. The one judgment call (the deployed host) stays as decided.

The guard passes on the whole tracked tree, but truncated stems and one real
OS user path remain in files outside this cluster:
`docs/HANDOFF.md` lines 52 and 53, `docs/plans/README.md` line 45, the review
report `26-public-release.md` line 44, `docs/history/nuxt-migration.md` (5
lines) and `docs/plans/2026-07-21-prd-agent-harness/research.md` (1 line).
`skills/insideout-setup/SKILL.md` lines 188 to 190 still describe `host.yml` as
holding a "checkout path"; it now holds a placeholder.

One judgment call: plan row 0.3 says "provider ids and deployed host in dated
records". This cluster replaced the ids only. The finding itself calls the
deployed host public by design (it is the shipped default), the guard exempts
dated records from its host rule on purpose, and a Railway-generated app host
is named in six more files (four dated changelogs, the changelog index and one
architecture page) outside this cluster's list. If the host should become a
placeholder too, that is a follow-up with the owner's say.

The shared indexes (`docs/issues/README.md`, the open-findings record, the
changelog index) are the integrator's.

## Verification

All tests live in `scripts/tests/check_redaction_addresses.py`. Each red
commit was replayed in a scratch checkout and fails (12, 1, 1, 3 and 3 failing
checks for PUB-01, PM-11, PUB-05, PUB-10 and PUB-12); each fix commit turns it
green. The outputs are kept as `w0-privacy-red-<ID>.txt` and
`w0-privacy-green-<ID>.txt` under `/private/tmp/insideout-rv4/reports/`.
Mutation checks a verifier can run: restore the `!.env.supabase` line, put a
real-looking address back, or restore a path in either `host.yml`, and the
matching test fails again.

Gates: `sh scripts/check-redaction.sh`, `sh scripts/check-docs-recorded.sh`,
every `scripts/tests/check_*.py`, `scripts/tests/keys`, the plan's row 0.1
check (no address in `docs/` outside `example.*`, `noreply` or `<`), and
`git ls-files hosts | sort` identical to the list taken before the wave.
