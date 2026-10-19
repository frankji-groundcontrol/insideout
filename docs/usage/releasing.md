# Releasing

How a release is cut today, read off
[`.github/workflows/release.yml`](../../.github/workflows/release.yml), and what
is not decided yet. A release is how a user without a checkout gets the CLI
([installation](installation.md)); the hosted service is deployed separately
([deployment](deployment.md)).

## The tag rule

Pushing a tag that matches `v*` runs the `release` workflow. The workflow has
no manual trigger. The only tag so far is `v0.1.0` (2026-10-11).

## What the workflow does, in order

1. **Tests.** `go test ./... -race -count=1` in `server/`. A red test stops the
   job, so nothing is built or uploaded. This runner has no PostgreSQL service,
   so the `DATABASE_URL`-gated suites skip here; the full suite runs in
   `ci.yml` on the same commit. A tag pushed on a commit without a green `ci`
   run can therefore still publish: check the `ci` run for that commit before
   you tag.
2. **Build.** `insideout` and `insideout-mcp` for linux and darwin on amd64 and
   arm64, CGO off, packed as `insideout-<os>-<arch>.tar.gz` with both binaries
   at the top level, plus `checksums.txt` (SHA-256) and `bootstrap.sh`.
   `insideout` is stamped with the tag (`-X main.version`), so
   `insideout version` prints it. A build from the container image or from
   `scripts/install.sh` is not stamped and prints `dev`.
3. **Publish.** `gh release create <tag> ... --generate-notes`. If the release
   already exists, the assets are uploaded over it with `--clobber`: a re-run
   of a tag repairs its assets, and also overwrites them. Row 3.4 of the
   [remediation plan](../review/2026-10-18-multirole-4/40-remediation-plan.md)
   stops that overwrite; revise this step when it lands.

`releases/latest/download/bootstrap.sh` (the one-liner in
[installation](installation.md)) fetches the installer shipped with the newest
release, so it always matches the assets it downloads.

## Before you tag

Confirm that the `ci` run on the commit you are tagging is green. Cite it by
dated SHA, not as "the head run" (see [HANDOFF](../HANDOFF.md)); the
release workflow does not check it for you.

## Not decided yet

These are the owner's calls, written down so nobody assumes an answer:

- **Version policy.** What raises the minor or the patch number, and when
  `v1.0.0` happens.
- **Rollback.** Whether a bad release is deleted, superseded by a patch tag or
  marked a pre-release. Because the workflow overwrites the assets of an
  existing tag, a corrected rebuild of the same tag is possible, but it changes
  bytes that users may already have verified against `checksums.txt`.
- **A stronger gate.** The test step is the floor. Two stronger forms need a
  decision: a PostgreSQL service in the release job, or a check that requires a
  green `ci` run for the tagged SHA (it needs `actions: read`).

Records: rv4-PM-05 and rv4-TEST-01 in the
[review-4 open findings](../issues/2026-10-18-review-4-open-findings.md);
[wave 1, CLI front door](../changelogs/2026-10-18-rv4-remediation/w1-cli-front-door.md).
