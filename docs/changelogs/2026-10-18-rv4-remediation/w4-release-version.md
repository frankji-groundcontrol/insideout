# Wave 4, release version — review-4 remediation

Branch `rv4/w4-release-version`, base `582a4bd`. Findings are cited as
`rv4-<ID>` elsewhere because review IDs restart per review. This cluster takes
over what [wave 3, release supply chain](w3-release-supply.md#moved-to-wave-4)
left for the installer and the runbook: rv4-PM-04, rv4-PM-05, and the consumer
side of rv4-PUB-03 and rv4-SEC-07 in `scripts/bootstrap.sh`.

**Nothing here has run on an Actions runner, and no release carries an
attestation.** `release.yml`, the attestation and the `ci` job that runs the
`check_*.py` guards are **pending first CI run after push**; the first `v*` tag
cut after that push is the first end-to-end proof of the provenance check.
What ran locally is stated per finding below.

## Commits

Red first, each finding's fix in its own commit; every red was run and its output
kept before the fix.

| Finding | Red | Fix |
|---------|-----|-----|
| rv4-PM-04 | `4ae2352` (the wave-3 lead's `14d9377`, cherry-picked) | `970a1da` |
| rv4-PM-05 | `d04723f` (the lead's `7718517`, cherry-picked), `1bf2908` (guard repair) | `7a5947d` |
| rv4-SEC-07 | `d8b5fff` | `30c9dc0` |
| rv4-PUB-03 | `dc84e7e` | `c565c18` |

## rv4-PM-04: `insideout version` no longer prints `dev` from the other recipes

Red `4ae2352`: the release gate guard demands that every recipe that builds
`./cmd/insideout` for someone to run stamps `-X main.version=` and supplies a
value.

```
scripts/install.sh: the ./cmd/insideout build is not stamped with -ldflags "-X main.version=...": `insideout version` prints dev (R1-04, PM-04)
scripts/install.sh: nothing supplies the stamp's value (`git describe` is missing)
server/Dockerfile: the ./cmd/insideout build is not stamped with -ldflags "-X main.version=...": `insideout version` prints dev (R1-04, PM-04)
server/Dockerfile: nothing supplies the stamp's value (`ARG VERSION` is missing)
```

Fix `970a1da`: `scripts/install.sh` passes `git describe --tags --always --dirty`
(`dev` without git); `server/Dockerfile` takes `ARG VERSION=dev` after `FROM` (the
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

Fix `7a5947d`: the page now has the tag rule, cutting a release, the five
workflow steps with the attest action named, a Version policy section (what each
build prints; the MINOR versus PATCH rule is **proposed, not decided**),
verifying a download, rollback, the supply-chain controls (each checked against
the workflow files, `dependabot.yml`, `security.yml` and the install skill) and
the owner decisions. `docs/usage/README.md` describes the page again.

## rv4-SEC-07: `bootstrap.sh` insists on an https release base

Red `d8b5fff`: a new guard,
[`check_bootstrap_release.py`](../../../scripts/tests/check_bootstrap_release.py).
It pipes `scripts/bootstrap.sh` to `bash -s` on stdin, as `curl | bash` runs it,
on a PATH of symlinked real tools plus a stub, with a scratch `HOME`; a throwaway
`ThreadingHTTPServer` on 127.0.0.1 (OS-assigned port) is the release mirror. The
harness is proven by the case that already passed at that commit: a release
served from a loopback mirror over http installs both binaries. Red, 18 lines:

```
INSIDEOUT_RELEASE_BASE=http://example.invalid/releases: want a refusal that names https before any request; got exit 22, 1 request(s), stderr ''
... the same line for nine more bases (localhost and 127.0.0.1 lookalikes, user-info before an @, ftp, file, no scheme) ...
INSIDEOUT_RELEASE_BASE=https://example.invalid/releases: a request leaves redirects open to plain http (no --proto-redir =https)
... the same for the four loopback bases and the default base ...
a redirect to plain http was followed: exit 0, installed ['insideout', 'insideout-mcp'], requested ['/redirect/<archive>', '/good/<archive>', '/redirect/checksums.txt', '/good/checksums.txt']
docs/usage/installation.md: INSIDEOUT_RELEASE_BASE is described without saying it must be https
```

Fix `30c9dc0`:

- `base_ok()` runs before any external command, with shell builtins only. `https://`
  passes anywhere. Plain http passes only for a mirror on this machine, decided on
  the authority (the text before the first `/`): exactly `localhost`, `127.0.0.1`
  or `[::1]`, with an optional all-digit port. `localhost.example`, a host that
  merely starts with `127.0.0.1`, user-info before an `@` (`http://localhost@host`),
  ftp, file and a base with no scheme are refused with a message that names https
  and says nothing was downloaded.
- Every download goes through `fetch()`, which adds `--proto-redir =https`: a
  mirror, or anything on the path, cannot step a download down to plain http with
  a redirect. The loopback http mirror still works because the first request is
  not a redirect.

Green: `bootstrap release ok: https-only release base and redirects, ...`; the two
neighbours that parse the script (`check_first_run_copy.py`, 6 checks, and
`check_ops_hardening.py`, 5 checks) stay green; `bash -n` passes under the bash 3.2.57
that macOS ships. Five weakened variants were each run through the guard and each
turned it red: any text accepted as the port, `localhost*` and `127.0.0.1*`
matched by prefix, the host read from the user-info before an `@`, `--proto-redir`
dropped, the https arm dropped.

## rv4-PUB-03: `bootstrap.sh` verifies the build attestation when `gh` can

Red `dc84e7e` extends the same guard with a stub `gh` whose answers each case
sets (`GH_AUTH`, `GH_HELP`, `GH_VERIFY`) and which reads stdin the way a prompt
would. Red, 8 lines (the SEC-07 half stays green):

```
gh is not installed: want a 'Provenance not checked' notice that says why ('not installed'); got exit 0, installed ['insideout', 'insideout-mcp'], stderr ''
gh is signed out: want a 'Provenance not checked' notice that says why ('not signed in'); got exit 0, installed [...], stderr ''
gh has no attestation command: want a 'Provenance not checked' notice that says why ('attestation'); got exit 0, installed [...], stderr ''
want one `gh attestation verify <archive> --repo frankji-groundcontrol/insideout --signer-workflow frankji-groundcontrol/insideout/.github/workflows/release.yml` run against github.com whatever GH_HOST says; got []
a failed verification should install nothing, say so and show gh's reason; got exit 0, installed ['insideout', 'insideout-mcp'], stderr ''
docs/usage/installation.md: does not say bootstrap.sh runs gh attestation verify
docs/usage/releasing.md: does not say bootstrap.sh runs gh attestation verify
docs/usage/releasing.md: still lists bootstrap.sh's download checks as open
```

Two properties passed at the red commit and are pinned so they stay true: a
checksum mismatch stops before `gh` is asked (the order is checksum, then
provenance), and `release.yml` has an `attest-build-provenance` step over `dist/*`
(the workflow the script pins is the one that attests).

Fix `c565c18`, between the checksum check and `tar -xzf` (nothing is unpacked or
installed before it):

- **gh can verify** when it is on PATH, `gh auth status --hostname github.com`
  passes and `gh attestation --help` works (a gh older than the command has none).
  Then `GH_HOST=github.com gh attestation verify <archive> --repo <repo>
  --signer-workflow <repo>/.github/workflows/release.yml`. Success prints
  `Provenance verified: built by <workflow>`. Any failure prints gh's own output,
  then `could not verify the build provenance of <archive> — nothing was
  installed`, the two usual causes (the release predates attestations; the archive
  is not what the workflow built) and `scripts/install.sh` as the way around, and
  exits 1.
- **gh cannot verify** (not installed, signed out, too old): a three-line
  `Provenance not checked: <reason>` notice, then the install goes on. The notice
  says what the checksum alone shows.
- **Every gh call has `</dev/null`.** Under `curl | bash` the script is on stdin,
  and a gh that reads it eats the rest. With the redirect removed the guard's stub
  does exactly that: the script exits 0 having installed nothing, and the guard
  reports it.
- `GH_HOST` is pinned for the verify call, so a shell with `GH_HOST` pointing at an
  enterprise host still checks this repository on github.com.

### The policy, and what it does to `v0.1.0`

Fail closed when the check can run; loud and open when it cannot. "Closed" has to
cover every failure of a check that ran: gh cannot tell "no attestation" from
"attestation for other bytes" by exit code, and treating "no attestation" as a
pass would let a swapped archive through, which is the attack. "Open" is the
other half: requiring gh would lock out most people who run the one-liner.

Consequence, stated in `installation.md` and `releasing.md`: `v0.1.0`, the only
release, predates the attestation step. Until the first tag cut by the current
workflow exists, `releases/latest/download` is `v0.1.0`, so a signed-in `gh` stops
the one-liner on it with the explanation above. Ways forward: that next tag;
`scripts/install.sh` from a checkout; running without a signed-in `gh`, which
skips the check with the notice. An archive someone built themselves has no
attestation either, so `INSIDEOUT_RELEASE_BASE` is documented as the base of a
mirror of a release.

Green: `bootstrap release ok: https-only release base and redirects, provenance
checked when gh can, loopback mirror installs`; `check_release_gate.py`,
`check_first_run_copy.py`, `check_ops_hardening.py` and `check_doc_links.py` green;
`bash -n` passes under bash 3.2.57. Eleven weakened variants of the new block were
each run through the guard and each turned it red: no `</dev/null`, no signed-in
probe, no capability probe, a failed verification let through, verification after
the install, verification before the checksum, `GH_HOST` not pinned, no
`--signer-workflow`, the checksums file verified instead of the archive, gh's
reason swallowed, a silent skip.

### What is and is not proven

- **Proven locally:** the script's behaviour on every branch (no gh, signed out,
  old gh, verified, rejected, checksum mismatch) against a stub `gh`, a stub
  `curl`, and a loopback `ThreadingHTTPServer`; the argv it passes; the order of
  checks; bash 3.2 syntax; the docs saying so.
- **Not proven, and not claimed:** the real attestation path. No release carries
  an attestation, so `gh attestation verify` has never run against api.github.com
  for this repository (pending first CI run after push, then the first tag). The
  stub answers by design; a gh that changes its flags would only show up there.
- **A live GitHub download was not possible from the machine this ran on:**
  github.com:443 is unreachable from it (curl exit 28 after 75 s), so the fixed
  script was not run against the real `v0.1.0` assets. The loopback mirror serves
  the same names (`insideout-<os>-<arch>.tar.gz`, `checksums.txt`).
- **Known limits:** the check pins the repository and the workflow path, not the
  tag, because the one-liner does not know the tag it will fetch; anyone who can
  run `release.yml` in this repository gets attested artifacts (an attestation
  proves where bytes were built, not that they are benign); `bootstrap.sh` itself
  cannot be checked before it is run (the page says how to verify it by hand);
  only github.com is supported as the attestation host; with the checksum file on
  the same origin as the archive, installs without `gh` have the integrity check
  they had before and nothing more.

## Owner decisions (DEFERRED_OWNER)

- **rv4-PM-05 policy.** What raises MINOR versus PATCH while the major is 0,
  what makes `v1.0.0`, and whether a suffixed tag (`v0.2.0-rc.1`) publishes as a
  pre-release. The page carries a proposed default and says it is not decided.
- **Rollback of a published release.** Mark a bad one a pre-release, delete it,
  or only supersede it. Only superseding is documented.
- **The release gate (R1-05).** The release waits for a green `ci` run on the
  tagged SHA because its runner has no PostgreSQL service. Keep that, or add a
  PostgreSQL service to the release job. Recorded on the page, not decided.
- **Installs without `gh`.** Install on the checksum alone, as now, or require
  `gh` (which locks out most one-liner users). Recorded on the page.
- **Hosted version.** Whether the Railway `server` service gets a `VERSION`
  variable so the hosted binary reports a version instead of `dev`.
- The repository-settings items stay with the owner, as in wave 3: immutable
  releases, who may create `v*` tags, required checks, GitHub secret and
  dependency features, and the seven findings in old history.

## Not taken

- **R3-15 (Docker base images by digest, a `docker` ecosystem in `dependabot.yml`)**
  stays open: `dependabot.yml` is not this cluster's file, and a digest has to be
  read from a registry. `releasing.md` still lists it as a gap.

## For the integrator

- INTEGRATOR TODO: index this fragment in
  `docs/changelogs/2026-10-18-rv4-remediation/index.md`, `docs/changelogs/README.md`
  and `docs/changelogs/index/2026-10.md`.
- INTEGRATOR TODO: `docs/plans/README.md` and `docs/HANDOFF.md`: row 3.4 can say
  PUB-03 and SEC-07 are closed on the consumer side; the real attestation path is
  **pending first CI run after push**. Add the follow-up: the first `v*` tag after
  the push proves the gate, the attestation and `bootstrap.sh`'s check end to end,
  and until then `latest` is `v0.1.0`, which a signed-in `gh` refuses.
- INTEGRATOR TODO: `docs/issues/2026-10-18-review-4-open-findings.md`: R1-04 (stamp
  in `install.sh` and the image fixed; `docker-compose.yml` still passes no
  `VERSION`), R1-05 (documented, owner decision open), R1-16 (the page is written;
  version policy and rollback stay owner decisions), R3-14 (closed by `30c9dc0` and
  `c565c18`, real path pending), R3-18 (this cluster landed). R3-15 stays open.
- INTEGRATOR TODO: `docker-compose.yml` is not this cluster's file. Pass the build
  argument (`args: VERSION: ${VERSION:-dev}` under the server `build:`) so a
  compose-built image can be stamped; until then it prints `dev`.
- INTEGRATOR TODO: `docs/architecture/cli-mcp-and-client-config.md` (the prebuilt
  bullet, about line 150) describes `bootstrap.sh` without the https rule and the
  provenance check; `README.md` (about line 72) says the one-liner is "verified
  working", which is true of the day it was checked and not of a signed-in `gh`
  against `v0.1.0`.
- INTEGRATOR TODO: if another wave-4 cluster edits the supply-chain table in
  `docs/usage/releasing.md`, keep the `Download checks` row and the shorter "Gap still
  open" sentence; the table was re-checked against the workflow files at
  `c565c18`.
