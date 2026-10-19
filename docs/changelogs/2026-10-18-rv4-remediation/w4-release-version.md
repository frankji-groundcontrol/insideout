# Wave 4, release version — review-4 remediation

Branch `rv4/w4-release-version`, base `582a4bd`. Findings are cited as
`rv4-<ID>` elsewhere because review IDs restart per review. This cluster takes
over what [wave 3, release supply chain](w3-release-supply.md#moved-to-wave-4)
left for the installer and the runbook: rv4-PM-04, rv4-PM-05, and the consumer
side of rv4-PUB-03 and rv4-SEC-07 in `scripts/bootstrap.sh`.

**Nothing here has run on an Actions runner.** `release.yml`, the attestation
and the `ci` job that runs the `check_*.py` guards are **pending first CI run
after push**. What ran locally is stated per finding below.

## rv4-PM-04: `insideout version` no longer prints `dev` from the other recipes

Red `4ae2352` (cherry-picked from the wave-3 lead, `14d9377`): the release gate
guard demands that every recipe that builds `./cmd/insideout` for someone to run
stamps `-X main.version=` and supplies a value.

```
scripts/install.sh: the ./cmd/insideout build is not stamped with -ldflags "-X main.version=...": `insideout version` prints dev (R1-04, PM-04)
scripts/install.sh: nothing supplies the stamp's value (`git describe` is missing)
server/Dockerfile: the ./cmd/insideout build is not stamped with -ldflags "-X main.version=...": `insideout version` prints dev (R1-04, PM-04)
server/Dockerfile: nothing supplies the stamp's value (`ARG VERSION` is missing)
```

Fix: `scripts/install.sh` passes `git describe --tags --always --dirty` (`dev`
without git); `server/Dockerfile` takes `ARG VERSION=dev` after `FROM` (the
build context is `server/`, which has no `.git`, and an `ARG` before `FROM` is
empty inside the stage). Two statements that this made stale moved with it:
the comment in `server/cmd/insideout/version.go` and the build step plus the
"Still open" paragraph of `docs/usage/releasing.md`.

Run by hand at the fix commit:

| Run | `insideout version` |
|-----|---------------------|
| `scripts/install.sh` from this checkout, scratch `HOME` | `insideout v0.1.0-339-g4ae23524-dirty` (equal to `git describe --tags --always --dirty` in the tree) |
| `scripts/install.sh` from a `git ls-files` export of the tree (no `.git`) | `insideout dev` |
| the Dockerfile's own `RUN go build` line (host `GOOS`), `VERSION=dev` | `insideout dev` |
| the same line, `VERSION=v9.9.9-test` | `insideout v9.9.9-test` |

No Docker is installed on this machine, so the image build itself is exercised
by the `server image` job in `ci.yml` (pending first CI run after push); the
`RUN` line was read out of the Dockerfile and run, not retyped.

Known limit: `git describe` in an unpacked source archive that sits inside some
other git repository describes that repository. It only changes the string.

## rv4-PM-05: `docs/usage/releasing.md` describes the workflow that ships

Red `d04723f` (the wave-3 lead's `7718517`, cherry-picked): the guard requires
the tag rule, version policy and rollback sections, the gate inputs
`release.yml` has (`--verify-tag`, `attest-build-provenance`, `actions: read`),
and no instruction to overwrite a release. On this base that guard printed one
false line next to the three real gaps: the page says "There is no" at a line
end and "`--clobber`" at the next line start, and the `(?<!no `)` look-behind
could not see across the wrap. Red `1bf2908` reads the page with whitespace
collapsed and says how to write the absence; a hand-made mutant (the page plus a
`gh release upload ... --clobber` line) is still caught. The true red list:

```
docs/usage/releasing.md: no `## Version policy` section
docs/usage/releasing.md: no `## Rollback` section
docs/usage/releasing.md does not mention `attest-build-provenance`, which release.yml does
```

The page now has: the tag rule, cutting a release, the five workflow steps with
the attest action named, a Version policy section (what each build prints; the
MINOR versus PATCH rule is **proposed, not decided**), verifying a download,
rollback, the supply-chain controls (each checked against the workflow files)
and the owner decisions. `docs/usage/README.md` describes the page again.

## Owner decisions (DEFERRED_OWNER)

- **rv4-PM-05 policy.** What raises MINOR versus PATCH while the major is 0,
  what makes `v1.0.0`, and whether a suffixed tag (`v0.2.0-rc.1`) publishes as a
  pre-release. The page carries a proposed default and says it is not decided.
- **Rollback of a published release.** Mark a bad one a pre-release, delete it,
  or only supersede it. Only superseding is documented.
- **Hosted version.** Whether the Railway `server` service gets a `VERSION`
  variable so the hosted binary reports a version instead of `dev`.
- The repository-settings items stay with the owner, as in wave 3: immutable
  releases, who may create `v*` tags, required checks, GitHub secret and
  dependency features, and the seven findings in old history.
