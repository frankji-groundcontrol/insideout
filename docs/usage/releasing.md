# Releasing

How a release is cut and checked, read off
[`.github/workflows/release.yml`](../../.github/workflows/release.yml), and the
decisions that stay with the owner. A release is how a user without a checkout
gets the CLI ([installation](installation.md)); the hosted service is deployed
separately ([deployment](deployment.md)).

The gate, the provenance attestation, the notices in the archives and the
no-overwrite publish below are review-4 row 3.4
([record](../changelogs/2026-10-18-rv4-remediation/w3-release-supply.md)). They
are CI configuration that has not run on GitHub yet: **pending first CI run
after push**, and the first `v*` tag cut after that push is their first proof.
The download checks in `scripts/bootstrap.sh` (an https-only base, the
attestation check) are the consumer half of that row
([Verifying a download](#verifying-a-download)): a local mirror and a stub `gh`
prove them, but their real attestation path also waits for that first tag.
Still open, so this page is not a finished runbook: the
[owner decisions](#owner-decisions) at the end.

## The tag rule

- Pushing a tag that matches `v*` runs the `release` workflow. There is no
  manual trigger. Name the tag `vMAJOR.MINOR.PATCH`; the workflow does not check
  the shape. The only tag so far is `v0.1.0` (2026-10-11), cut before the gate
  below existed.
- Tag the commit that `ci` went green on, and only a commit that is on `main`:
  the first step of the workflow refuses anything else.
- An annotated tag (`git tag -a`) records who tagged and when; to the workflow a
  lightweight tag is the same.

## Cutting a release

1. Merge to `main`. Wait for the `ci` run on the merge commit to finish green;
   cite it by dated SHA, not as "the head run" (see [HANDOFF](../HANDOFF.md)).
2. Tag that exact commit and push the tag:
   `git tag -a v0.2.0 <sha> -m "v0.2.0" && git push origin v0.2.0`.
3. Watch the run (`gh run list --workflow release`, then `gh run watch <run-id>`).
   It stops at the first red step.
4. Check the result: [Verifying a download](#verifying-a-download).

If the gate fails, nothing was built or published. "not on `main`": delete the
tag (`git push --delete origin v0.2.0`) and tag the right commit. "no successful
ci run": wait for `ci` to finish green on that commit, then re-run the failed
job.

## What the workflow does, in order

1. **Gate.** The tagged commit must be reachable from the default branch and the
   `ci` workflow must have a successful run for exactly that SHA (the job has
   `actions: read` to list it). `ci.yml` is the only place the
   `DATABASE_URL`-gated suites run; this runner has no PostgreSQL service, so a
   green `ci` run is the evidence they passed. An API error, an unreadable
   answer or a zero count stops the run: the gate cannot fail open.
2. **Tests.** `go test ./... -race -count=1` in `server/`. A red test stops the
   job, so nothing is built or uploaded. The `DATABASE_URL`-gated suites skip
   here.
3. **Build.** `insideout` and `insideout-mcp` for linux and darwin on amd64 and
   arm64, CGO off, packed as `insideout-<os>-<arch>.tar.gz` with both binaries
   at the top level beside `LICENSE` and `THIRD_PARTY_NOTICES.md` (the MPL-2.0
   section 3.2 source offer and the license texts), plus `checksums.txt`
   (SHA-256) and `bootstrap.sh`. `insideout` is stamped with the tag
   (`-X main.version`); see [Version policy](#version-policy) for what the other
   build recipes print.
4. **Attest.** `actions/attest-build-provenance` (pinned by commit SHA) signs a
   build-provenance attestation for every file in `dist/`, with the job's OIDC
   identity (`id-token: write`, `attestations: write`). It sits after the build
   and before the publish, so nothing unattested is released.
5. **Publish.** `gh release create <tag> dist/* --verify-tag --generate-notes`.
   `--verify-tag` aborts unless the tag already exists on the remote. There is no
   `--clobber` and no upload over an existing release: a tag that already has a
   release is an error, so a published release is never rewritten.

`releases/latest/download/bootstrap.sh` (the one-liner in
[installation](installation.md)) fetches the installer shipped with the newest
release that is neither a draft nor a pre-release, so it always matches the
assets it downloads.

## Version policy

The tag is the version. What `insideout version` prints depends on how the
binary was built; each recipe supplies its own value, and
[`check_release_gate.py`](../../scripts/tests/check_release_gate.py) pins the
stamp in all three (the linker silently ignores `-X` for a symbol that does not
exist):

| Build | Prints |
|-------|--------|
| release archive | the tag, e.g. `v0.2.0` (`-X main.version=$GITHUB_REF_NAME`) |
| `scripts/install.sh` in a checkout | `git describe --tags --always --dirty`: `v0.1.0` on the tagged commit, `v0.1.0-176-g14d9377` after it, a trailing `-dirty` with local edits; `dev` without git |
| server image (`server/Dockerfile`) | the `VERSION` build argument; `dev` when none is passed |
| `go run`, `go build`, `go test` | `dev` |

To stamp a local image:
`docker build --build-arg VERSION="$(git describe --tags --always --dirty)" server`.
Nothing in this repository passes the argument (`docker-compose.yml` sets none),
so an image built from it prints `dev`. Stamping the hosted image is an owner
decision, below.

**What raises which number is not decided.** Proposed default, for the owner to
accept or change: while the major version is 0, a change that breaks a
documented command, flag, config key or API contract raises MINOR and anything
else raises PATCH. Open with it: what makes `v1.0.0`, and whether a tag with a
suffix (`v0.2.0-rc.1`) should publish as a pre-release. Today every tag
publishes as a normal release and takes `latest`; a pre-release lane would be a
`--prerelease` condition in `release.yml`.

## Verifying a download

- **Transport.** `bootstrap.sh` fetches over https only: `INSIDEOUT_RELEASE_BASE`
  must be an `https://` URL (plain http is accepted for `localhost`, `127.0.0.1`
  and `[::1]` only), and a redirect to plain http is refused.
- **Checksum.** `bootstrap.sh` checks the archive against `checksums.txt` before
  it installs anything. The file ships from the same origin as the archive, so it
  catches a corrupt or truncated download, not a swapped upload.
- **Provenance.** `bootstrap.sh` runs `gh attestation verify` on the archive
  after the checksum, when `gh` is installed, signed in to github.com and new
  enough to have the command, and installs nothing if it fails. Otherwise it
  prints `Provenance not checked` with the reason and installs on the checksum
  alone. By hand: `gh attestation verify insideout-linux-amd64.tar.gz --repo
  frankji-groundcontrol/insideout --signer-workflow
  frankji-groundcontrol/insideout/.github/workflows/release.yml` (the script
  passes exactly those; the second flag requires that this workflow built it).
  The attestation covers every file of the release, `checksums.txt` and
  `bootstrap.sh` included, so the installer itself can be checked the same way
  before it is piped to a shell. Pending first CI run: no release carries an
  attestation yet. `v0.1.0` predates it, so with `gh` signed in `bootstrap.sh`
  stops on `v0.1.0`; the first tag cut after the workflow lands fixes that for
  `latest`.
- **Contents.** `tar -tzf insideout-<os>-<arch>.tar.gz` lists `insideout`,
  `insideout-mcp`, `LICENSE` and `THIRD_PARTY_NOTICES.md`. The notices carry the
  MPL-2.0 section 3.2 source offer and the license text of every module linked
  into the binaries.

## Rollback

A published release is never edited or overwritten: someone may already have
verified an archive against `checksums.txt`, and new bytes under the same tag
break that. A bad release is superseded by a new tag.

- **Published and bad.** Fix on `main`, wait for `ci`, and tag the next version.
  The new release becomes `latest`, so the one-liner installs it. Say what is
  wrong in the bad release's notes. Until the fix ships the bad release is still
  `latest`. Whether to also mark it a pre-release (which takes it out of `latest`)
  or delete it is open; see [Owner decisions](#owner-decisions). Nothing in the
  workflow does either.
- **The run died before the release existed.** Nothing was published: re-run the
  failed job.
- **The run died after the draft was created.** `gh release view <tag>` shows a
  draft. Delete it (`gh release delete <tag>` keeps the tag) and re-run the failed
  job.
- **Never** delete a tag that has a published release and push the same name at
  another commit. Use a new tag.

## Supply-chain controls

| Control | Where | What it does |
|---------|-------|--------------|
| Pinned Actions | every workflow | each `uses:` is a full commit SHA with the tag in a trailing comment; Dependabot reads the comment and rewrites both |
| Least privilege | every workflow | top-level `permissions: contents: read`; the release job adds the four it needs |
| Dependency updates | [`dependabot.yml`](../../.github/dependabot.yml) | weekly pull requests for `gomod`, `pub` and `github-actions`, with a 7-day cooldown |
| Vulnerability scan | [`security.yml`](../../.github/workflows/security.yml) | `govulncheck` (pinned) on the server packages, built by the toolchain `server/go.mod` selects; fixed by bumping a module or the toolchain line, never by muting the job |
| Secret scan | `security.yml` | `gitleaks` (pinned, redacted) over the commits a push or pull request adds |
| Notices | [`THIRD_PARTY_NOTICES.md`](../../THIRD_PARTY_NOTICES.md) | every module `server/go.mod` requires has an entry; a new dependency fails the guard until it has one |
| Toolchain archives | [install skill](../../skills/insideout-setup/references/modules.md) | the Go and Flutter archives are checked against a pinned SHA-256 before they are unpacked; the `ubuntu-install` workflow compares the Flutter pins to the vendor feeds |
| Download checks | [`scripts/bootstrap.sh`](../../scripts/bootstrap.sh) | https-only release base and redirects, the SHA-256 against `checksums.txt`, and `gh attestation verify` when `gh` can ([Verifying a download](#verifying-a-download)) |

[`check_supply_chain.py`](../../scripts/tests/check_supply_chain.py),
`check_release_gate.py` and
[`check_bootstrap_release.py`](../../scripts/tests/check_bootstrap_release.py)
pin these properties and run in the `ci` `env-suites` job. The `security`
workflow is not part of the release gate. Gap still open: the Docker base images
are pinned by tag, not digest, and are not in `dependabot.yml`.

## Owner decisions

Written down so nobody assumes an answer. None of them is a workflow edit an
agent can make for you. The click paths for the repository settings are also in
the
[release-supply record](../changelogs/2026-10-18-rv4-remediation/w3-release-supply.md#owner-decisions-deferred_owner).

- **Version policy.** What raises MINOR versus PATCH while the major is 0, what
  makes `v1.0.0`, and whether a suffixed tag publishes as a pre-release. The
  proposal is in [Version policy](#version-policy).
- **Rollback of a published release.** Mark a bad one a pre-release, delete it,
  or leave it and supersede it only. Today only the last is documented.
- **The release gate.** The release waits for a green `ci` run on the tagged
  commit because its runner has no PostgreSQL service, so the `DATABASE_URL`-gated
  suites cannot run there. Keep that, or add a PostgreSQL service to the release
  job and run them in it?
- **Immutable releases.** Make a published release's tag and assets unchangeable?
  Settings > General > Releases > "Enable release immutability".
- **Who may tag.** Restrict creating, updating and deleting `v*` tags to the
  maintainers? Settings > Rules > Rulesets > New ruleset > New tag ruleset,
  target `v*`.
- **Required checks.** Which `ci` and `security` jobs must be green to merge to
  `main`? `security` cannot be required until `govulncheck` is green. Settings >
  Rules > Rulesets > New ruleset > New branch ruleset, target `main`, "Require
  status checks to pass".
- **GitHub secret and dependency features.** Turn on secret scanning, push
  protection, Dependabot alerts and Dependabot security updates? Settings >
  Advanced Security (older UI: Code security and analysis).
- **Old history (rv4-PUB-05).** A full-history `gitleaks` run reports 7 findings.
  Allow-list the fixtures in a `.gitleaksignore`, and rotate or disable the key
  behind the deleted-file ones? History is never rewritten by an agent.
- **Installs without `gh`.** `bootstrap.sh` installs on the checksum alone when
  `gh` is missing, signed out or too old, and says so; most people who run the
  one-liner have no `gh`. Requiring it would lock them out, and nothing here
  decides that for you.
- **Hosted version.** Should the hosted image report a version rather than
  `dev`? Railway service `server` > Variables, add `VERSION`: the Dockerfile
  declares `ARG VERSION`, and Railway hands a service variable to a declared
  `ARG`. Confirm on the first deploy that sets it.

Records: rv4-PM-04, rv4-PM-05, rv4-TEST-01, rv4-PUB-03, rv4-PUB-04, rv4-PUB-07,
rv4-SEC-07 and rv4-PUB-09 in the
[review-4 open findings](../issues/2026-10-18-review-4-open-findings.md);
[wave 3, release supply chain](../changelogs/2026-10-18-rv4-remediation/w3-release-supply.md);
[wave 4, release version](../changelogs/2026-10-18-rv4-remediation/w4-release-version.md).
