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
          Neither scan may be muted: each scanner is one exact command line
          (govulncheck ./... from server/), and no `||` list, continue-on-error
          or `if:` skip may swallow a finding. pub07_mutations proves it by
          running that check on muted copies of security.yml.
  SEC-07  every `uses:` in every workflow is pinned to a full 40-hex commit
          SHA with the release tag kept in a trailing `# vX.Y.Z` comment, so
          a moved or hijacked tag cannot change what runs (Dependabot reads
          the comment and rewrites both). Local and docker:// refs are exempt.
  PUB-09  every archive the install skill downloads (Go, Flutter) is checked
          against a pinned sha256 before it is unpacked or replaces the
          installed toolchain, and a mismatch stops the block: a `&&` chain
          must be the last statement (nothing after it, such as a write to a
          shell rc file, may still run), or the check is followed by `|| exit`.
          (The Go floor half of PUB-09 lives in check_go_floor.py.)

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


SECURITY = ".github/workflows/security.yml"
# The scanners, each as the one command line it may have: a flag, pipe, redirect or
# `||` list on these lines changes what a scan reports. --redact keeps a finding from
# printing the secret into a public log.
SCANS = {"govulncheck": "govulncheck ./...",
         "gitleaks": 'gitleaks git --redact --verbose --no-banner --exit-code 1 --log-opts="$range" .'}
# What lets a failing step pass; no step of a file whose job is to fail may carry it.
MUTES = [(r"\|\|", "an `||` list"), (r"\bcontinue-on-error\b", "`continue-on-error`"),
         (r"(?m)^\s*(?:set\s+\+\w*e|exit\s+0\b|trap\b)", "`set +e`, `exit 0` or `trap`")]


def security_workflow(raw=None):
    """Findings for the text of security.yml (the file in the tree unless `raw` is given)."""
    raw = read(SECURITY) if raw is None else raw
    if raw is None:
        return [f"{SECURITY} does not exist: no govulncheck, no secret scan (PUB-07)"]
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
    commands = [re.sub(r"^(?:- )?(?:run:\s*)?", "", l.strip()) for l in text.splitlines()]
    for tool, line in SCANS.items():
        found = [c for c in commands if c.split(None, 1)[:1] == [tool]]
        if found != [line]:
            out.append(f"security.yml: the one `{tool}` command must be exactly `{line}`, found {found or 'none'}: "
                       "a flag, pipe, `||` list or echo in its place mutes the scan")
    if not re.search(r"(?m)^\s*working-directory: server\n\s*run: govulncheck \./\.\.\.$", text):
        out.append("security.yml: govulncheck must run from server/, where go.mod is (`working-directory: server` just above its `run:`)")
    out += [f"security.yml: {what} lets a failing scan pass" for pat, what in MUTES if re.search(pat, text)]
    if any(c != "github.event_name != 'schedule'" for c in re.findall(r"(?m)^\s*(?:- )?if:\s*(.*\S)", text)):
        out.append("security.yml: only `github.event_name != 'schedule'` on the secrets job may skip a job or step: a skipped scan is a muted scan")
    if "fetch-depth: 0" not in text:
        out.append("security.yml: the secret scan needs `fetch-depth: 0` to see the commits it scans")
    return out


def gitignore_keys():
    lines = {l.strip() for l in (read(".gitignore") or "").splitlines()}
    return [f".gitignore does not ignore {pat} (PUB-07)" for pat in ("*.pem", "*.key", "*.p12") if pat not in lines]


def pub07():
    """PUB-07: least-privilege tokens, dependency and secret scanning, key files ignored."""
    return workflow_permissions() + dependabot() + security_workflow() + gitignore_keys()


def edit(pattern, repl, count=0):
    """A mutation: re.sub over the file text (`^` and `$` match at every line)."""
    return lambda text: re.sub(pattern, repl, text, count=count, flags=re.M)


# Each mutation rewrites a protected file the way a hurried edit would; its guard
# must call the result red. A mutation that stops applying (the file changed
# shape) fails too: the proof would pass vacuously. (what, file, rewrite, guard)
JOB = r"^(    runs-on:.*)$"  # the first job's runner line, a place to add a job-level key
MUTATIONS = [
    ("`|| true` after govulncheck", SECURITY, edit(r"(govulncheck \./\.\.\.)$", r"\1 || true"), security_workflow),
    ("`|| true` after the govulncheck install", SECURITY, edit(r"(govulncheck@v[\d.]+)$", r"\1 || true"), security_workflow),
    ("gitleaks replaced by echo, flags kept", SECURITY, edit(r"^( *)gitleaks git ", r"\1echo gitleaks git "), security_workflow),
    ("gitleaks --exit-code 0", SECURITY, edit(r"--exit-code 1", "--exit-code 0"), security_workflow),
    ("continue-on-error on the first job", SECURITY, edit(JOB, r"\1\n    continue-on-error: true", 1), security_workflow),
    ("`if: false` on the first job", SECURITY, edit(JOB, r"\1\n    if: false", 1), security_workflow),
    ("govulncheck run from the repository root", SECURITY, edit(r"^ *working-directory: server\n", ""), security_workflow),
    ("`exit 0` after gitleaks", SECURITY, edit(r"(gitleaks git .*)$", r"\1\n          exit 0"), security_workflow),
    ("`set +e` instead of `set -euo pipefail`", SECURITY, edit(r"^( *)set -euo pipefail$", r"\1set +e"), security_workflow),
]


def pub07_mutations():
    """PUB-07: the guard goes red on a muted copy of the files it protects."""
    out = []
    for what, path, rewrite, guard in MUTATIONS:
        raw = read(path)
        if raw is None:
            continue  # the guard itself reports the missing file
        mutated = rewrite(raw)
        if mutated == raw:
            out.append(f"{path}: the mutation `{what}` no longer applies: update MUTATIONS so the proof is not vacuous")
        elif not guard(mutated):
            out.append(f"{path}: the guard stays green on a copy with {what} (PUB-07)")
    return out


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


# PUB-09: the install skill (references/modules.md) downloads toolchain archives
# that are unpacked as root (Go replaces /usr/local/go). VERIFY is the checksum
# command, DESTRUCTIVE what must not run before it succeeds.
ARCHIVE_URL = re.compile(r"https?://[^\s\"']+?\.(?:zip|tar\.xz|tar\.gz|tgz)\b")
VERIFY = re.compile(r"(?:sha256sum|shasum -a 256) -c\b")
DESTRUCTIVE = re.compile(r"\btar\b[^&|;]*\s-[A-Za-z]*x|\bunzip\b|\brm -rf /usr/local/go\b")


def pub09():
    """PUB-09: downloaded toolchain archives are verified against a pinned sha256 before use."""
    doc = read("skills/insideout-setup/references/modules.md")
    if doc is None:
        return ["skills/insideout-setup/references/modules.md does not exist (PUB-09)"]
    out = []
    for block in re.findall(r"```bash\n(.*?)```", doc, re.S):
        code = joined("\n".join(re.sub(r"\s#.*$", "", l) for l in block.splitlines()))
        urls = ARCHIVE_URL.findall(code)
        if not urls:
            continue
        name = urls[0].rsplit("/", 1)[-1]
        verify, unpack = VERIFY.search(code), DESTRUCTIVE.search(code)
        if not verify:
            out.append(f"modules.md: {name} is downloaded and unpacked with no `sha256sum -c` / `shasum -a 256 -c` (PUB-09)")
            continue
        if unpack and unpack.start() < verify.start():
            out.append(f"modules.md: {name} is unpacked (or the old toolchain removed) before its checksum is verified (PUB-09)")
        lines = code.splitlines()
        at = next(i for i, l in enumerate(lines) if VERIFY.search(l))
        stop = re.match(r"\s*\S*\s*(&&|\|\|.*\bexit\b)", lines[at][VERIFY.search(lines[at]).end():])
        if not stop:
            out.append(f"modules.md: the checksum check for {name} does not stop the block on a mismatch (chain it with && or follow it with `|| exit`) (PUB-09)")
        elif stop.group(1) == "&&" and any(l.strip() for l in lines[at + 1 :]):
            out.append(f"modules.md: a command follows the `&&` chain that checks {name}: a mismatch stops the chain but that command still runs; "
                       "put it in the chain or follow the check with `|| exit` (PUB-09)")
        if not re.search(r"\b[0-9a-f]{64}\b", block):
            out.append(f"modules.md: no pinned sha256 for {name}: the block must carry the 64-hex value, not fetch it beside the archive (PUB-09)")
    return out


CHECKS = [pub04, pub07, pub07_mutations, sec07, pub09]


def main():
    failures = [f for check in CHECKS for f in check()]
    for failure in failures:
        print(f"supply chain: {failure}", file=sys.stderr)
    if failures:
        sys.exit(1)
    print("supply chain ok: " + ", ".join(c.__name__ for c in CHECKS))


if __name__ == "__main__":
    main()
