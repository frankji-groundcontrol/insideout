# Review 4 privacy scrub — what only the owner can finish

Open, recorded 2026-10-18. Wave 0 of the
[review-4 remediation](../review/2026-10-18-multirole-4/40-remediation-plan.md)
scrubbed what a change can scrub: e-mail addresses, machine paths, provider
ids and one dead `.gitignore` exception, with a new address rule in
[`scripts/check-redaction.sh`](../../scripts/check-redaction.sh) behind it.
Four things are not engineering work: a mailbox, a dashboard with a key, a
directory layout, and published history. Each is stated below with what is
already done, what holds until the owner decides, and what closes it.

This record spells no name, address, project reference or key; it gives
counts and file locations only. Cite the findings as `rv4-PUB-01` and so on
(review IDs restart per review). Wave-0 changes are in the
[wave-0 fragment](../changelogs/2026-10-18-rv4-remediation/w0-privacy.md).

## Decisions

| Row | Register IDs | The owner decides | Already done | Holds until decided | Closes when |
|---|---|---|---|---|---|
| 0.1 | rv4-PUB-01, rv4-PM-11 | Rotate the hosted admin address and the owner-domain account address, both of which were public in dated records. | Scrubbed to `<admin-email>` / `<owner-email>`: 3 spots in 2 records, and 2 spots in 1 record; the truncated stems and the owner mail domain left the handoff, the plans board and one review quote (follow-up). The guard now rejects any other address. | Keep both addresses. | The owner writes "keep" or "rotated" in this row. |
| 0.2 | rv4-PUB-05 | In the hosting dashboard of the legacy database project, confirm the Data API exposes no schema, then rotate or disable its publishable key. | The `.gitignore` exception for the legacy env file is deleted; `.env.*` ignores that name again. | Project and key untouched. | The owner records "exposed schemas empty" and "key rotated, disabled or accepted". |
| 0.3 | rv4-PUB-10 | Ship `hosts/` as an example tree (`hosts/example/`) and keep the real one untracked, or keep today's layout. | Both `host.yml` files carry `<repo-checkout>` and `<ssh-alias>`; two records stopped quoting the alias; no `hosts/` directory was renamed, moved or deleted; the OS user left two older docs and the setup skill's host-record sentence now matches (follow-up). | Keep the layout. | The owner records "example tree" (then run the prompt below) or "keep". |
| 0.4 | rv4-PUB-06 | Accept the author identities in history, or rewrite history. | Nothing; this row is a decision only. | Accept; do not rewrite. | The owner records "accept" or "rewrite" here. |

## 0.1 Addresses

Rotating changes a live account on the hosted instance, so it is the owner's.
The leftovers outside this wave's file list were closed by the wave-0
follow-up (see Residue below). The release tag embeds a tagger address
(rv4-PM-11); tags are history, so that falls under 0.4.

## 0.2 The legacy project

The ignore rule now sends the legacy env file through `.env.*` like every
other dotenv file; nothing in the tree reads it. Whether the legacy project
still serves anything through its Data API, and whether its publishable key
still works, can only be seen in its dashboard.

## 0.3 The example-tree option

Today `hosts/` holds 31 tracked files: 2 `host.yml`, 5 READMEs and 24 key
manifests (names and fill state, no value) in two machine directories plus an
index README, and nine files outside `hosts/` and the review folder name the
two directories. The directory names are machine names and stay as they are,
because `scripts/keys.py` discovers hosts by folder and every manifest must
still name its own folder. Choosing the example tree is a layout change with
an owner-visible cost: the real tree leaves git, so a second clone no longer
sees which keys exist.

Bounded fix prompt, to run only after the owner records "example tree":

> Add `hosts/example/` with one `host.yml`, a README and one sample manifest
> in today's shape. Untrack the two machine directories (`git rm --cached`),
> add them to `.gitignore`, and re-point the nine records that link to them at
> `hosts/example/`. Keep `scripts/tests/keys` green (`name:` equals the
> folder). Do not delete any untracked key file. Record it as a changelog.

## 0.4 History

The review counted, over 1,155 commits: 1,005 with an institutional author
address, 29 with a machine-local hostname inside the address, and one commit
authored under a third party's real name; there is no `.mailmap`
([PUB-06](../review/2026-10-18-multirole-4/26-public-release.md)). A rewrite
is a lock-step mailmap rewrite plus a force-push of `main`: every clone is
re-cloned, the release tag is re-pointed, and the third party must agree
first. Without a recorded decision nobody rewrites; no wave touches history.

## Residue the guard cannot see (closed by the wave-0 follow-up)

The address rule sees whole addresses. These spots held truncated stems, the
owner's mail domain or a real OS user, in files outside the cluster's file
list (the handoff and the plans board are integrator-owned). They were
engineering work, not owner decisions, and the integrator's follow-up commit
closed them with the same placeholder swap, then re-ran
`sh scripts/check-redaction.sh` (clean):

- `docs/HANDOFF.md` (2 lines) — address stems with ellipses and the owner's
  mail domain, now `<admin-email>`, `<rehearsal-account>` and `<owner-domain>`.
- `docs/plans/README.md` (1 line) — an address stem with an ellipsis, now
  `<admin-email>`.
- `docs/review/2026-10-18-multirole-4/26-public-release.md` (1 line) — the
  same stems inside a quoted finding, now the same placeholders, marked as
  redacted after the review.
- `docs/history/nuxt-migration.md` (6 paths on 5 lines) and
  `docs/plans/2026-07-21-prd-agent-harness/research.md` (1 line) — the real OS
  user in a path, now `/Users/<os-user>/`.
- `skills/insideout-setup/SKILL.md` and the comment in
  `config/git-hooks/pre-commit` — prose that still said `host.yml` holds a
  checkout path and that the guard covers one address; both now match the code
  (a comment-only change to the hook).

The ceiling stays: a truncated stem or a bare domain added later is invisible
to the rule, and a person has to catch it in review. The OS-user part has a
generic check, `git grep -n -I -E '/Users/[A-Za-z0-9._-]+/'`, which prints
nothing on the tracked tree today.
