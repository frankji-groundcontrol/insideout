#!/usr/bin/env python3
"""Fail when the release job can publish untested or unstamped binaries.

Usage: python3 -I scripts/tests/check_release_gate.py

.github/workflows/release.yml builds and publishes four binary archives on
every `v*` tag push. Review 4 found it did that with no test step and no CI
gate (TEST-01, PM-05): a tag on a red commit shipped. It also built an
unstamped `insideout`, so `insideout version` could not say which release
it was (USER-02, PM-04). This guard reads the workflow text (stdlib only,
no YAML parser) and requires:

  1. a step that runs `go test` with `-race` before the first step that
     runs `go build` or `gh release`, so a failing test stops the publish;
  2. that step cannot fail open (no continue-on-error, no `|| true`);
  3. the `./cmd/insideout` build carries `-ldflags "-X main.version=..."`
     and server/cmd/insideout/version.go declares `var version` (the
     linker silently ignores -X for a symbol that does not exist).

The publish path is also gated and cannot overwrite (rv4-PUB-03, SEC-07):

  4. a gate step before any test or build proves the tagged commit is an
     ancestor of the default branch and that a successful `ci` run exists
     for that exact SHA (the full DATABASE_URL-gated suites run there, not
     on this runner); it exits non-zero and cannot fail open;
  5. no `--clobber` and no `gh release upload`: a published release is
     never rewritten, and `gh release create` uses `--verify-tag`;
  6. an `actions/attest-build-provenance` step runs after the build and
     before the publish, so no unattested archive is ever published;
  7. the job grants what 4 to 6 need (`actions: read`,
     `contents: write`, `id-token: write`, `attestations: write`) and its
     checkout fetches full history, which the ancestor check reads.

The other recipes that build `insideout` for someone to run stamp it too
(rv4-PM-04 residual, R1-04), so `insideout version` does not print `dev` from
an installed or containerised build:

  8. scripts/install.sh builds ./cmd/insideout with `-X main.version=` and
     takes the value from `git describe` (it runs in a checkout);
     server/Dockerfile does the same from a `VERSION` build argument declared
     after the stage's FROM (its context, server/, has no .git, and an ARG
     before FROM is empty inside the stage).

Exits 1 naming every failed property, 2 when the workflow shape cannot be
read. CI-config property: the first real proof is the first tag run.
"""
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parents[2]
WORKFLOW = ROOT / ".github/workflows/release.yml"
VERSION_GO = ROOT / "server/cmd/insideout/version.go"


def steps(text):
    """The job's steps in order, each a list of lines, full-line comments dropped."""
    lines = [l for l in text.splitlines() if not l.lstrip().startswith("#")]
    start = next((i for i, l in enumerate(lines) if re.match(r"^\s*steps:\s*$", l)), None)
    if start is None:
        print("release.yml: no `steps:` list found", file=sys.stderr)
        sys.exit(2)
    out, dash = [], None
    for line in lines[start + 1 :]:
        if not line.strip():
            continue
        indent = len(line) - len(line.lstrip())
        if dash is not None and indent < dash:
            break
        if re.match(r"^\s*- ", line) and (dash is None or indent == dash):
            dash = indent
            out.append([line])
        elif out:
            out[-1].append(line)
    if not out:
        print("release.yml: `steps:` has no entries", file=sys.stderr)
        sys.exit(2)
    return out


def commands(step):
    """The step's shell text: everything from its `run:` key on."""
    for i, line in enumerate(step):
        if re.match(r"^\s*(- )?run:", line):
            return "\n".join(step[i:])
    return ""


def first(all_steps, pattern):
    return next((i for i, s in enumerate(all_steps) if re.search(pattern, commands(s))), None)


def uncommented(text):
    return "\n".join(l for l in text.splitlines() if not l.lstrip().startswith("#"))


def job_permissions(text):
    """The release job's `permissions:` mapping ({} when it has none)."""
    found = re.search(r"^    permissions:\n((?:      \S.*\n)+)", uncommented(text) + "\n", re.M)
    return dict(re.findall(r"(\S+):\s*(\S+)", found.group(1))) if found else {}


def publish_path(text, all_steps):
    """rv4-PUB-03, SEC-07: gated on the default branch and a green ci run,
    never overwriting, attested before it is published."""
    code = uncommented(text)
    failures = []
    gate = first(all_steps, r"merge-base\s+--is-ancestor")
    firsts = [i for i in (first(all_steps, r"\bgo test\b"), first(all_steps, r"\bgo build\b"),
                          first(all_steps, r"\bgh release\b")) if i is not None]
    if gate is None:
        failures.append("no step proves the tagged commit is an ancestor of the default branch (`git merge-base --is-ancestor`): "
                        "a tag on any pushed commit publishes (PUB-03)")
    else:
        body = commands(all_steps[gate])
        for needle in ("actions/workflows/ci.yml/runs", "head_sha", "status=success", "exit 1"):
            if needle not in body:
                failures.append(f"the gate step lacks `{needle}`: it must require a successful ci run for the tagged SHA (PUB-03, PM-05)")
        if re.search(r"continue-on-error|\|\|\s*(true|:)", "\n".join(all_steps[gate])):
            failures.append("the gate step can fail open (continue-on-error or `|| true`)")
        if firsts and gate > min(firsts):
            failures.append("the gate step runs after a test/build/publish step; it must come first")
    if re.search(r"--clobber|\bgh release upload\b", code):
        failures.append("release.yml overwrites assets (`--clobber` or `gh release upload`): a published tag must stay immutable (PUB-03, SEC-07)")
    create = next((l for l in re.sub(r"\\\n\s*", " ", code).splitlines() if re.match(r"\s*(?:if\s+(?:!\s+)?)?gh release create\b", l)), None)
    if create is None:
        failures.append("no `gh release create` publishes the release")
    elif "--verify-tag" not in create:
        failures.append("`gh release create` lacks --verify-tag: a missing tag would be created from the default branch tip")
    attest = next((i for i, s in enumerate(all_steps) if re.search(r"uses:\s*actions/attest-build-provenance@", "\n".join(s))), None)
    build = first(all_steps, r"\bgo build\b")
    publish = first(all_steps, r"\bgh release create\b")
    if attest is None:
        failures.append("no actions/attest-build-provenance step: the archives carry no provenance (PUB-03, SEC-07)")
    elif (build is not None and attest < build) or (publish is not None and attest > publish):
        failures.append("the attestation step must run after the build and before the publish")
    perms = job_permissions(text)
    for scope, level in (("actions", "read"), ("contents", "write"), ("id-token", "write"), ("attestations", "write")):
        if perms.get(scope) != level:
            failures.append(f"the release job lacks `permissions: {scope}: {level}` (needed by the gate, publish or attestation)")
    checkout = next((s for s in all_steps if "actions/checkout" in "\n".join(s)), [])
    if "fetch-depth: 0" not in "\n".join(checkout):
        failures.append("the checkout does not fetch full history: `git merge-base --is-ancestor` cannot see the default branch")
    return failures


# Build recipes other than release.yml, and the pattern that supplies the stamp's value in each.
OTHER_BUILDS = {
    "scripts/install.sh": r"\bgit\b[^\n|;]*\bdescribe\b",
    "server/Dockerfile": r"^ARG VERSION\b",
}


def other_builds():
    """rv4-PM-04 residual (R1-04): install.sh and the Dockerfile stamp `insideout` too."""
    failures = []
    for rel, source in OTHER_BUILDS.items():
        path = ROOT / rel
        text = re.sub(r"\\\n\s*", " ", uncommented(path.read_text(encoding="utf-8") if path.is_file() else ""))
        build = next((l for l in text.splitlines() if "go build" in l and re.search(r"\./cmd/insideout(?![-\w])", l)), None)
        if build is None:
            failures.append(f"{rel}: no `go build ... ./cmd/insideout` line found")
            continue
        if "-X main.version=" not in build:
            failures.append(f"{rel}: the ./cmd/insideout build is not stamped with -ldflags \"-X main.version=...\": "
                            "`insideout version` prints dev (R1-04, PM-04)")
        supply = re.search(source, text, re.M)
        if not supply:
            failures.append(f"{rel}: nothing supplies the stamp's value (expected `{source}`)")
        elif rel.endswith("Dockerfile") and supply.start() < text.index("FROM "):
            failures.append(f"{rel}: `ARG VERSION` sits before FROM, so it is empty inside the build stage")
    return failures


def main():
    text = WORKFLOW.read_text(encoding="utf-8")
    all_steps = steps(text)
    failures = publish_path(text, all_steps)

    test = next(
        (i for i, s in enumerate(all_steps) if "go test" in commands(s) and "-race" in commands(s)),
        None,
    )
    ships = [i for i in (first(all_steps, r"\bgo build\b"), first(all_steps, r"\bgh release\b")) if i is not None]
    if test is None:
        failures.append("no step runs `go test ... -race`: a tag on a red commit publishes (TEST-01, PM-05)")
    elif ships and test > min(ships):
        failures.append("the `go test -race` step runs after a build/publish step; it must come first")
    elif re.search(r"continue-on-error|\|\|\s*(true|:)", "\n".join(all_steps[test])):
        failures.append("the `go test -race` step can fail open (continue-on-error or `|| true`)")

    logical = re.sub(r"\\\n\s*", " ", text)
    build = next((l for l in logical.splitlines() if "go build" in l and re.search(r"\./cmd/insideout(?![-\w])", l)), None)
    if build is None:
        print("release.yml: no `go build ... ./cmd/insideout` line found", file=sys.stderr)
        sys.exit(2)
    if "-X main.version=" not in build:
        failures.append("the ./cmd/insideout build is not stamped with -ldflags \"-X main.version=...\" (USER-02, PM-04)")
    if not VERSION_GO.is_file() or not re.search(r"^var version\s*=", VERSION_GO.read_text(encoding="utf-8"), re.M):
        failures.append("server/cmd/insideout/version.go does not declare `var version`: the -X stamp would be ignored")

    others = other_builds()
    for failure in failures:
        print(f"release.yml: {failure}", file=sys.stderr)
    for failure in others:
        print(failure, file=sys.stderr)
    if failures or others:
        sys.exit(1)
    print("release gate ok: gated on a green ci run, go test -race, no clobber, attested, "
          "version-stamped (release.yml, install.sh, Dockerfile)")


if __name__ == "__main__":
    main()
