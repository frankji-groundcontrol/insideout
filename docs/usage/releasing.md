# Releasing

How a release is cut today, read off
[`.github/workflows/release.yml`](../../.github/workflows/release.yml), and what
is not decided yet. A release is how a user without a checkout gets the CLI
([installation](installation.md)); the hosted service is deployed separately
([deployment](deployment.md)).

The gate, the provenance attestation, the notices in the archives and the
no-overwrite publish below are review-4 row 3.4
([record](../changelogs/2026-10-18-rv4-remediation/w3-release-supply.md)). They
are CI configuration that has not run on GitHub yet: **pending first CI run
after push**, and the first `v*` tag cut after that push is their first proof.
Still open, so this page is not a finished runbook: the version policy and the
rollback rule (rv4-PM-05), and the download checks in `scripts/bootstrap.sh`.

## The tag rule

Pushing a tag that matches `v*` runs the `release` workflow. The workflow has
no manual trigger. The only tag so far is `v0.1.0` (2026-10-11), cut before the
gate below existed.

## What the workflow does, in order

1. **Gate.** The tagged commit must be reachable from the default branch and the
   `ci` workflow must have a successful run for exactly that SHA (the job has
   `actions: read` to list it). `ci.yml` is the only place the
   `DATABASE_URL`-gated suites run; this runner has no PostgreSQL service, so a
   green `ci` run is the evidence they passed. An API error, an unreadable
   answer or a zero count stops the run: the gate cannot fail open. Tag the
   commit `ci` ran on.
2. **Tests.** `go test ./... -race -count=1` in `server/`. A red test stops the
   job, so nothing is built or uploaded. The `DATABASE_URL`-gated suites skip
   here.
3. **Build.** `insideout` and `insideout-mcp` for linux and darwin on amd64 and
   arm64, CGO off, packed as `insideout-<os>-<arch>.tar.gz` with both binaries
   at the top level beside `LICENSE` and `THIRD_PARTY_NOTICES.md` (the MPL-2.0
   section 3.2 source offer and the license texts), plus `checksums.txt`
   (SHA-256) and `bootstrap.sh`. `insideout` is stamped with the tag
   (`-X main.version`), so `insideout version` prints it. The other two build
   recipes stamp too: `scripts/install.sh` passes `git describe --tags --always
   --dirty` of the checkout it builds (`dev` when there is no git), and
   `server/Dockerfile` takes a `VERSION` build argument (`docker build
   --build-arg VERSION="$(git describe --tags --always --dirty)" server`; its
   context has no `.git`, and an image built without the argument prints `dev`).
4. **Attest.** A signed build-provenance attestation for every file in `dist/`,
   made with the job's OIDC identity (`id-token: write`, `attestations: write`).
   It sits after the build and before the publish, so nothing unattested is
   released.
5. **Publish.** `gh release create <tag> dist/* --verify-tag --generate-notes`.
   `--verify-tag` aborts unless the tag already exists on the remote. There is no
   `--clobber` and no upload over an existing release: a tag that already has a
   release is an error, so a published release is never rewritten. If a run dies
   after the draft was created, delete that draft before re-running the job.

`releases/latest/download/bootstrap.sh` (the one-liner in
[installation](installation.md)) fetches the installer shipped with the newest
release, so it always matches the assets it downloads.

## Verifying a download

- **Checksum.** `bootstrap.sh` checks the archive against `checksums.txt` before
  it installs anything. The file comes from the same origin as the archive, so it
  catches a corrupt or truncated download, not a swapped upload.
- **Provenance.** `gh attestation verify <file> --repo <owner>/<repo>` checks the
  signed attestation against this repository and this workflow. `bootstrap.sh`
  does not run it yet (rv4-SEC-07 and rv4-PUB-03, partial), so run it by hand.
- **Contents.** `tar -tzf insideout-<os>-<arch>.tar.gz` lists `insideout`,
  `insideout-mcp`, `LICENSE` and `THIRD_PARTY_NOTICES.md`.

## Before you tag

Confirm that the `ci` run on the commit you are tagging is green. Cite it by
dated SHA, not as "the head run" (see [HANDOFF](../HANDOFF.md)). The gate checks
it too and stops the run when there is none, after the tag exists: delete a tag
that was refused (`git push --delete origin <tag>`) and tag the right commit, or
wait for `ci` to finish and re-run the failed job.

## Not decided yet

These are the owner's calls, written down so nobody assumes an answer. The
questions and the repository-settings click paths are in the
[release-supply record](../changelogs/2026-10-18-rv4-remediation/w3-release-supply.md#owner-decisions-deferred_owner).

- **Version policy.** What raises the minor or the patch number, whether a tag
  with a suffix publishes as a pre-release, and when `v1.0.0` happens.
- **Rollback.** A published release is never overwritten, so a bad one is
  superseded by a new tag; whether it is also deleted or marked a pre-release is
  open. A rebuilt asset under the same tag would change bytes that users may
  already have verified against `checksums.txt`.
- **Settings the workflow cannot make:** immutable releases, who may create `v*`
  tags, which checks are required to merge, and GitHub's secret scanning and
  Dependabot alerts.

Records: rv4-PM-05 and rv4-TEST-01 in the
[review-4 open findings](../issues/2026-10-18-review-4-open-findings.md);
[wave 1, CLI front door](../changelogs/2026-10-18-rv4-remediation/w1-cli-front-door.md).
