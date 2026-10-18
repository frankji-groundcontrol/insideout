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
  PUB-07  every workflow narrows the token with a top-level `permissions:`
          (no write at the top level); dependabot covers gomod, pub and
          github-actions; security.yml runs govulncheck and gitleaks, each
          pinned to an exact release and installed by `go install` (no
          third-party Action), gitleaks redacting and seeing full history;
          .gitignore keeps *.pem, *.key and *.p12 out of commits.
  SEC-07  every `uses:` in every workflow is pinned to a full 40-hex commit
          SHA with the release tag kept in a trailing `# vX.Y.Z` comment, so
          a moved or hijacked tag cannot change what runs (Dependabot reads
          the comment and rewrites both). Local and docker:// refs are exempt.

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


def top_permissions(text):
    """(inline value, indented block) of the workflow-level `permissions:`, or None."""
    lines = text.splitlines()
    for i, line in enumerate(lines):
        if line.startswith("permissions:"):
            block = []
            for nxt in lines[i + 1 :]:
                if not (nxt.startswith((" ", "\t")) or not nxt.strip()):
                    break
                block.append(nxt)
            return line.split(":", 1)[1].split("#")[0].strip(), "\n".join(block)
    return None


def workflow_permissions():
    out = []
    for path in sorted((ROOT / ".github/workflows").glob("*.yml")):
        found = top_permissions(path.read_text(encoding="utf-8"))
        if found is None:
            out.append(f"{path.name}: no top-level `permissions:` (the token keeps the repository default) (PUB-07, SEC-07)")
            continue
        inline, block = found
        wide = [f"{k}: {v}" for k, v in re.findall(r"^\s+([\w-]+):\s*([\w-]+)", block, re.M) if v not in ("read", "none")]
        if inline in ("write-all", "read-all") or wide:
            out.append(f"{path.name}: the top-level permissions grant {inline or ', '.join(wide)}; grant write per job instead")
    return out


def dependabot():
    text = read(".github/dependabot.yml")
    if text is None:
        return [".github/dependabot.yml does not exist (PUB-07)"]
    out = [] if re.search(r"^version:\s*2\b", text, re.M) else ["dependabot.yml: no `version: 2`"]
    seen = {}
    for block in re.split(r"^\s*- package-ecosystem:", text, flags=re.M)[1:]:
        eco = block.split("\n", 1)[0].strip().strip("\"'")
        directory = re.search(r"directory:\s*[\"']?([^\"'\s#]+)", block)
        interval = re.search(r"interval:\s*[\"']?(\w+)", block)
        seen[eco] = (directory and directory.group(1), interval and interval.group(1))
    for eco, directory in (("gomod", "/server"), ("pub", "/client"), ("github-actions", "/")):
        if seen.get(eco, (None, None))[0] != directory:
            out.append(f"dependabot.yml: no `{eco}` update for directory {directory}")
        elif seen[eco][1] != "weekly":
            out.append(f"dependabot.yml: the `{eco}` update is not scheduled weekly")
    return out


def security_workflow():
    raw = read(".github/workflows/security.yml")
    if raw is None:
        return [".github/workflows/security.yml does not exist: no govulncheck, no secret scan (PUB-07)"]
    text, out = joined(raw), []
    for event in ("push", "pull_request", "schedule"):
        if not re.search(rf"^  {event}:", text, re.M):
            out.append(f"security.yml: not triggered by {event} (a new advisory needs the weekly schedule, a leak needs push and pull_request)")
    for tool, install in (("govulncheck", r"go install golang\.org/x/vuln/cmd/govulncheck@v\d+\.\d+\.\d+\b"),
                          ("gitleaks", r"go install github\.com/zricethezav/gitleaks/v8@v\d+\.\d+\.\d+\b")):
        if not re.search(install, text):
            out.append(f"security.yml: {tool} is not installed with `go install ...@vX.Y.Z` (pin an exact release, no @latest, no third-party Action)")
    if re.search(r"@latest\b", text):
        out.append("security.yml: `@latest` is not a pin")
    if not re.search(r"govulncheck\s+\./\.\.\.", text):
        out.append("security.yml: govulncheck does not scan ./...")
    scans = [l for l in text.splitlines() if "--log-opts" in l]  # the gitleaks range scan
    if not scans or not all("--redact" in l for l in scans):
        out.append("security.yml: the gitleaks scan (`gitleaks git --log-opts=...`) must pass --redact: an unredacted finding prints the secret into a public log")
    if "fetch-depth: 0" not in text:
        out.append("security.yml: the secret scan needs `fetch-depth: 0` to see the commits it scans")
    return out


def gitignore_keys():
    lines = {l.strip() for l in (read(".gitignore") or "").splitlines()}
    return [f".gitignore does not ignore {pat} (PUB-07)" for pat in ("*.pem", "*.key", "*.p12") if pat not in lines]


def pub07():
    """PUB-07: least-privilege tokens, dependency and secret scanning, key files ignored."""
    return workflow_permissions() + dependabot() + security_workflow() + gitignore_keys()


def sec07():
    """SEC-07: every Action is pinned to a full commit SHA, the tag kept in a trailing comment."""
    out = []
    for path in sorted((ROOT / ".github/workflows").glob("*.yml")):
        for n, line in enumerate(path.read_text(encoding="utf-8").splitlines(), 1):
            m = re.match(r"\s*(?:- )?uses:\s*(\S+)(.*)$", line)
            if not m or m.group(1).startswith(("./", "docker://")):
                continue
            ref = m.group(1).partition("@")[2]
            if not re.fullmatch(r"[0-9a-f]{40}", ref):
                out.append(f"{path.name}:{n}: {m.group(1)} is not pinned to a full commit SHA (SEC-07)")
            elif not re.match(r"\s*#\s*v\d+(\.\d+)+", m.group(2)):
                out.append(f"{path.name}:{n}: {m.group(1)} has no `# vX.Y.Z` comment naming the tag it pins (SEC-07)")
    return out


CHECKS = [pub04, pub07, sec07]


def main():
    failures = [f for check in CHECKS for f in check()]
    for failure in failures:
        print(f"supply chain: {failure}", file=sys.stderr)
    if failures:
        sys.exit(1)
    print("supply chain ok: " + ", ".join(c.__name__ for c in CHECKS))


if __name__ == "__main__":
    main()
