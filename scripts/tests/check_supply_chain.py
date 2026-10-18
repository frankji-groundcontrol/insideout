#!/usr/bin/env python3
"""Fail when the release supply chain regresses (review 4, w3-release-supply).

Usage: python3 -I scripts/tests/check_supply_chain.py

Stdlib only, hermetic (reads repository text, no network, no Go toolchain),
so the CI `env-suites` job runs it with the other check_*.py guards. One
function per finding; each returns its failure messages and main() prints
them all. The release job itself (gate, attestation, no clobber) is pinned by
check_release_gate.py.

  PUB-04  every release archive carries LICENSE, THIRD_PARTY_NOTICES.md and
          the MPL-2.0 section 3.2 source offer, and the notices name every
          module server/go.mod requires (so a new dependency forces an entry).

CI-config properties: the first real proof is the first tag run after push.
"""
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parents[2]

# go.mod modules that no shipped binary links, checked with `go list -deps`
# for linux and darwin on amd64 and arm64 (the four release targets): a
# dependency's tests import the first two, bubbletea links the others on
# Windows only. Anything else in go.mod must appear in the notices.
NOT_LINKED = {
    "github.com/davecgh/go-spew",
    "github.com/pmezard/go-difflib",
    "github.com/erikgeiser/coninput",
    "github.com/mattn/go-localereader",
}


def read(rel):
    path = ROOT / rel
    return path.read_text(encoding="utf-8") if path.is_file() else None


def joined(text):
    """The text without full-line comments and with backslash continuations joined."""
    code = "\n".join(l for l in text.splitlines() if not l.lstrip().startswith("#"))
    return re.sub(r"\\\n\s*", " ", code)


def go_mod_modules():
    """Every module path in the `require` directives of server/go.mod."""
    found, in_block = set(), False
    for line in (read("server/go.mod") or "").splitlines():
        line = line.split("//")[0].strip()
        if line == "require (":
            in_block = True
        elif in_block and line == ")":
            in_block = False
        elif in_block or line.startswith("require "):
            parts = line.removeprefix("require ").split()
            if len(parts) == 2 and parts[1].startswith("v"):
                found.add(parts[0])
    return found


def pub04():
    """PUB-04: LICENSE, third-party notices and the MPL source offer ship in every tarball."""
    out = []
    license_text = read("LICENSE") or ""
    if "Mozilla Public License Version 2.0" not in license_text:
        out.append("LICENSE is missing or is not the MPL-2.0 text")
    release = joined(read(".github/workflows/release.yml") or "")
    tars = [l for l in release.splitlines() if re.search(r"\btar\s+-c", l)]
    for name in ("LICENSE", "THIRD_PARTY_NOTICES.md"):
        if not tars or not all(name in l for l in tars):
            out.append(f"release.yml: the tar command does not pack {name} into every archive (PUB-04)")
    notices = read("THIRD_PARTY_NOTICES.md")
    if notices is None:
        return out + ["THIRD_PARTY_NOTICES.md does not exist at the repository root (PUB-04)"]
    repo = re.search(r"^repo=(\S+)", read("scripts/bootstrap.sh") or "", re.M)
    url = f"https://github.com/{repo.group(1)}" if repo else None
    if "Source Code Form" not in notices or not url or url not in notices:
        out.append(f"THIRD_PARTY_NOTICES.md lacks the MPL-2.0 section 3.2 source offer "
                   f"(the words `Source Code Form` and the repository URL {url})")
    if "Go standard library" not in notices or "The Go Authors" not in notices:
        out.append("THIRD_PARTY_NOTICES.md does not credit the Go standard library (The Go Authors)")
    modules = go_mod_modules()
    if not modules:
        out.append("server/go.mod: no require directives found")
    out += [f"THIRD_PARTY_NOTICES.md does not name {m}, which server/go.mod requires" for m in sorted(modules - NOT_LINKED) if m not in notices]
    out += [f"NOT_LINKED lists {m}, which server/go.mod no longer requires: drop it" for m in sorted(NOT_LINKED - modules)]
    return out


CHECKS = [pub04]


def main():
    failures = [f for check in CHECKS for f in check()]
    for failure in failures:
        print(f"supply chain: {failure}", file=sys.stderr)
    if failures:
        sys.exit(1)
    print("supply chain ok: " + ", ".join(c.__name__ for c in CHECKS))


if __name__ == "__main__":
    main()
