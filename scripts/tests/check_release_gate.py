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


def main():
    text = WORKFLOW.read_text(encoding="utf-8")
    all_steps = steps(text)
    failures = []

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

    for failure in failures:
        print(f"release.yml: {failure}", file=sys.stderr)
    if failures:
        sys.exit(1)
    print("release gate ok: go test -race runs first, fails closed; insideout is version-stamped")


if __name__ == "__main__":
    main()
