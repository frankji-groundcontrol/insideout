# Releasing

How a release is cut and checked, read off
[`.github/workflows/release.yml`](../../.github/workflows/release.yml), and the
decisions that stay with the owner. A release is how a user without a checkout
gets the CLI ([installation](installation.md)); the hosted service is deployed
separately ([deployment](deployment.md)).

The gate, the provenance attestation, the notices in the archives and the
no-overwrite publish below are review-4 row 3.4
([record](../changelogs/2026-10-18-rv4-remediation/w3-release-supply.md)). They
are CI configuration that has not run yet: **pending first CI run after push**.
The first `v*` tag cut after that push is their first proof.

## The tag rule

- Pushing a tag that matches `v*` runs the `release` workflow. There is no
  manual trigger. Name the tag `vMAJOR.MINOR.PATCH`; the workflow does not check
  the shape. The only tag so far is `v0.1.0` (2026-10-11).
- Tag the commit that `ci` went green on, and only a commit that is on `main`.
  The first step of the workflow refuses anything else.
- An annotated tag (`git tag -a`) records who tagged and when; a lightweight tag
  works the same for the workflow.

## Cutting a release

1. Merge to `main`. Wait for the `ci` run on the merge commit to finish green;
   cite it by dated SHA, not as "the head run" (see [HANDOFF](../HANDOFF.md)).
2. Tag that exact commit and push the tag:
   `git tag -a v0.2.0 <sha> -m "v0.2.0" && git push origin v0.2.0`.
3. Watch the run (`gh run watch`). It stops at the first red step.
4. Check the result: [Verifying a download](#verifying-a-download).

If the gate fails, nothing was built or published. "not on main": delete the
tag (`git push --delete origin v0.2.0`) and tag the right commit. "no successful
ci run": wait for `ci` to finish green on that commit, then re-run the failed
job.

## What the workflow does, in order

1. **Gate.** The tagged commit must be reachable from the default branch, and
   the `ci` workflow must have a successful run for exactly that SHA (the job
   has `actions: read` to list it). An API error, an unreadable answer or a zero
   count stops the run, so the gate cannot fail open.
2. **Test.** `go test ./... -race -count=1` in `server/`. A red test stops the
   job before anything is built.
3. **Build.** `insideout` and `insideout-mcp` for linux and darwin on amd64 and
   arm64, CGO off, packed as `insideout-<os>-<arch>.tar.gz`: both binaries,
   `LICENSE` and `THIRD_PARTY_NOTICES.md` at the top level. Plus `checksums.txt`
   (SHA-256 of the four archives) and `bootstrap.sh`. `insideout` is stamped with
   the tag.
4. **Attest.** `actions/attest-build-provenance` signs a provenance attestation
   for every file in `dist/` with the job's OIDC identity (`id-token: write`,
   `attestations: write`). It sits after the build and before the publish, so
   nothing unattested is released.
5. **Publish.** `gh release create <tag> dist/* --verify-tag --generate-notes`.
   gh creates the release as a draft, uploads the assets, then publishes it.
   `--verify-tag` aborts unless the tag already exists on the remote. There is no
   `--clobber` and no upload-over: an existing release for the tag is an error,
   so a published release is never rewritten.

`releases/latest/download/bootstrap.sh` (the one-liner in
[installation](installation.md)) fetches the installer shipped with the newest
release that is neither a draft nor a pre-release, so it always matches the
assets it downloads.

**Why a green `ci` run and not a PostgreSQL service in this job.** The release
runner has no PostgreSQL, so the `DATABASE_URL`-gated suites skip in step 2; the
gate is the evidence they passed. A service here would copy the two-role
database setup that `ci.yml` builds, and the copy would drift. The `ci` run also
carries the jobs a release should not skip: the client tests, the install
smoke, the redaction guard and the `check_*.py` guards. The cost is that a tag
pushed minutes after a merge fails the gate until `ci` is green; re-run it then.

## Version policy

The tag is the version. What `insideout version` prints:

| Build | Prints |
|-------|--------|
| release archive | the tag, e.g. `v0.2.0` (`-X main.version=$GITHUB_REF_NAME`) |
| `scripts/install.sh` in a checkout | `git describe --tags --always --dirty`: the tag on a tagged commit, `v0.1.0-176-g14d9377` after it, `-dirty` with local edits; `dev` without git |
| server container image | the `VERSION` build argument; `dev` when none is passed (compose and Railway pass none today) |
| `go run`, `go build`, `go test` | `dev` |

[`check_release_gate.py`](../../scripts/tests/check_release_gate.py) pins the
stamp in all three build paths: the linker silently ignores `-X` for a symbol
that does not exist. To stamp a local image:
`docker build --build-arg VERSION="$(git describe --tags --always --dirty)" server`.
Railway injects service variables into a Dockerfile build and the Dockerfile
declares `ARG VERSION`, so a `VERSION` variable on the `server` service would
stamp the hosted image.

What raises MINOR versus PATCH, and what makes `v1.0.0`, is the owner's call
(see [Owner decisions](#owner-decisions)). Proposed default, not decided: while
the major version is 0, a change that breaks a documented command, flag, config
key or API contract raises MINOR and anything else raises PATCH.

## Verifying a download

- **Checksum.** `bootstrap.sh` checks the archive against `checksums.txt` before
  it installs anything. The file ships from the same origin as the archive, so it
  catches a corrupt or truncated download, not a swapped upload.
- **Provenance.** `gh attestation verify insideout-linux-amd64.tar.gz --repo
  frankji-groundcontrol/insideout` checks the signed attestation; add
  `--signer-workflow frankji-groundcontrol/insideout/.github/workflows/release.yml`
  to require that this workflow built it. Pending first CI run: not exercised
  yet. `bootstrap.sh` does not run it, so run it by hand.
- **Contents.** `tar -tzf insideout-<os>-<arch>.tar.gz` lists `insideout`,
  `insideout-mcp`, `LICENSE` and `THIRD_PARTY_NOTICES.md`. The notices carry the
  MPL-2.0 section 3.2 source offer and the license text of every module linked
  into the binaries.

## Rollback

A published release is never edited or overwritten: someone may already have
verified an archive against `checksums.txt`, and new bytes under the same tag
break that.

- **Published and bad.** Supersede it: fix on `main`, wait for `ci`, tag the next
  patch version. The new release becomes `latest`, so the one-liner installs it.
  To pull the bad one out of `latest` before the fix is ready:
  `gh release edit <tag> --prerelease`; `latest` falls back to the previous
  release. Say what is wrong in its notes. Deleting a published release is the
  owner's call.
- **Run died before the release existed.** Nothing was published: re-run the
  failed job.
- **Job killed mid-upload (timeout, lost runner).** gh deletes its own temporary
  draft when an upload fails, but a killed job cannot. If `gh release view <tag>`
  shows a draft, delete it (`gh release delete <tag>` keeps the tag) and re-run
  the failed job.
- **Never** delete a tag that has a published release and push the same name
  again. Use a new tag.

## Supply-chain controls

| Control | Where | What it does |
|---------|-------|--------------|
| Pinned Actions | every workflow | each `uses:` is a full commit SHA with the tag in a trailing comment; Dependabot reads the comment and rewrites both |
| Least privilege | every workflow | top-level `permissions: contents: read`; the release job adds the four it needs |
| Dependency updates | [`dependabot.yml`](../../.github/dependabot.yml) | weekly pull requests for `gomod`, `pub` and `github-actions`, with a 7-day cooldown |
| Vulnerability scan | [`security.yml`](../../.github/workflows/security.yml) | `govulncheck` (pinned) on the packages the server builds, with the toolchain `server/go.mod` selects; fixed by bumping the module or the toolchain line, never by muting the job |
| Secret scan | `security.yml` | `gitleaks` (pinned, redacted) over the commits a push or pull request adds |
| Notices | [`THIRD_PARTY_NOTICES.md`](../../THIRD_PARTY_NOTICES.md) | every module `server/go.mod` requires has an entry; a new dependency fails the guard until it has one |
| Toolchain archives | [install skill](../../skills/insideout-setup/SKILL.md) | the Go and Flutter archives are checked against a pinned SHA-256 before they are unpacked; the `ubuntu-install` workflow compares the Flutter pins to the vendor feeds |

[`check_supply_chain.py`](../../scripts/tests/check_supply_chain.py) and
`check_release_gate.py` pin these properties and run in the `ci` `env-suites`
job. The `security` workflow is not part of the release gate. Known gaps:
`bootstrap.sh` does not enforce https for `INSIDEOUT_RELEASE_BASE` or verify the
attestation; the Docker base images are pinned by tag, not digest, and are not
in `dependabot.yml`; the pre-commit redaction check only warns (the `ci`
redaction guard and `gitleaks` are what block).

## Owner decisions

Written down so nobody assumes an answer. None of them is a workflow edit an
agent can make for you.

- **Version policy.** Question: what raises MINOR versus PATCH while the major
  is 0, what makes `v1.0.0`, and should a tag with a suffix (`v0.2.0-rc.1`)
  publish as a pre-release? Today every tag publishes as a normal release and
  takes `latest`. Where: edit this page; a pre-release lane is a `--prerelease`
  condition in `release.yml`.
- **Immutable releases.** Question: make a published release's tag and assets
  unchangeable? Where: repository Settings > General > Releases > "Enable release
  immutability".
- **Who may tag.** Question: restrict creating, updating and deleting `v*` tags
  to the maintainers? Where: Settings > Rules > Rulesets > New ruleset > New tag
  ruleset, target `v*`, restrict creations, updates and deletions.
- **Required checks.** Question: which `ci` and `security` jobs must be green to
  merge to `main`? `security` cannot be required until `govulncheck` is green.
  Where: Settings > Rules > Rulesets > New ruleset > New branch ruleset, target
  `main`, "Require status checks to pass".
- **GitHub secret and dependency features.** Question: turn on secret scanning,
  push protection, Dependabot alerts and Dependabot security updates? Where:
  Settings > Advanced Security (older UI: Code security and analysis).
- **Old history.** A full-history `gitleaks` run reports 7 findings, all rule
  `generic-api-key`: 5 in three files still in the tree (test fixtures and a
  review ledger; two of them are reported twice) and 2 in files deleted since.
  Question: allow-list the fixtures in a `.gitleaksignore`, and rotate or
  disable the key behind the deleted-file findings (rv4-PUB-05)? History is never
  rewritten by an agent.
- **Hosted version.** Question: should the hosted image report a version rather
  than `dev`? Where: Railway service `server` > Variables, add `VERSION`.

Records: rv4-PM-05, rv4-TEST-01, rv4-PUB-03, rv4-PUB-04, rv4-PUB-07, rv4-SEC-07
and rv4-PUB-09 in the
[review-4 open findings](../issues/2026-10-18-review-4-open-findings.md);
[wave 3, release supply chain](../changelogs/2026-10-18-rv4-remediation/w3-release-supply.md).
