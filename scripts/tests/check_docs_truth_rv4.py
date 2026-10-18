#!/usr/bin/env python3
"""Fail when the record-system docs contradict the tree (review-4 wave 3).

Usage: python3 -I scripts/tests/check_docs_truth_rv4.py

Companion to check_docs_truth.py, split out to keep both under the line
budget. Same method: derive the premise from the tree first, then assert the
doc agrees, so the prose cannot silently re-drift. Exits 1 naming every
offending claim (rv4-FULL-09, rv4-USER-08, rv4-FEAT-08, rv4-PM-08).
"""
import pathlib
import re
import subprocess
import sys

ROOT = pathlib.Path(__file__).resolve().parents[2]


def read(name):
    return (ROOT / name).read_text(encoding="utf-8")


def fail(failures, message):
    failures.append(message)


def mentions(text, name):
    # \b keeps INSIDEOUT_GH_PRIVATE_KEY from matching inside ..._KEY_FILE
    return re.search(rf"\b{re.escape(name)}\b", text) is not None


def env_variables_documented(facts, failures):
    """rv4-FULL-09: every INSIDEOUT_* variable the Go code reads, or a CI
    workflow sets, is in the environment reference; the ones the server
    process reads are also in .env.example, the env catalog's source."""
    read_at = {}
    for path in sorted((ROOT / "server").rglob("*.go")):
        if path.name.endswith("_test.go"):
            continue
        where = path.relative_to(ROOT).as_posix()
        for name in re.findall(r'"(INSIDEOUT_[A-Z0-9_]+)"', path.read_text(encoding="utf-8")):
            read_at.setdefault(name, where)
    for wf in sorted((ROOT / ".github/workflows").glob("*.yml")):
        for name in re.findall(r"\b(INSIDEOUT_[A-Z0-9_]+)\b", wf.read_text(encoding="utf-8")):
            read_at.setdefault(name, wf.relative_to(ROOT).as_posix())
    facts["env variables"] = len(read_at)
    reference, example = read("docs/usage/environment.md"), read(".env.example")
    for name, where in sorted(read_at.items()):
        if not mentions(reference, name):
            fail(failures, f"docs/usage/environment.md does not list {name} (used at {where})")
        if where.startswith(("server/internal/config/", "server/internal/agent/")) \
                and not mentions(example, name):
            fail(failures, f".env.example does not list {name} (the server reads it at {where})")


# Living docs a reader follows; dated records (changelogs, plans, review) are history.
LIVING_FILES = ("README.md", "PRODUCT.md", "docs/SETENV.md", ".env.example")
LIVING_TREES = ("docs/usage", "docs/architecture")
OFFLINE_OVERCLAIM = re.compile(r"(?i)end[- ]to[- ]end|canned|stages, tools|whole PRD flow|recommended default")


def living_docs():
    found = [ROOT / name for name in LIVING_FILES]
    for tree in LIVING_TREES:
        found += sorted((ROOT / tree).rglob("*.md"))
    return found


def offline_coach_not_overclaimed(failures):
    """rv4-USER-08: the offline template coach echoes the member's last message
    and does nothing else (no stages, no sections, no tool calls). While the
    code does that, no living doc may call it end-to-end, canned or the
    recommended default, and the offline-mode section must say "echo" and what
    real coaching needs."""
    if "onDelta(lastUser)" not in read("server/internal/agent/template.go"):
        return  # the streamer no longer echoes; the wording may change
    for path in living_docs():
        for block in re.split(r"\n\s*\n", path.read_text(encoding="utf-8")):
            hit = OFFLINE_OVERCLAIM.search(block)
            if hit and re.search(r"(?i)offline", block):
                fail(failures, f"{path.relative_to(ROOT)} calls the offline coach "
                               f"'{hit.group(0)}', but it only echoes the last message")
    section = re.search(r"### Offline AI mode\n(.*?)(?=\n###? )", read("docs/usage/local-development.md"), re.S)
    text = section.group(1) if section else ""
    if not re.search(r"(?i)\becho", text):
        fail(failures, "local-development.md's Offline AI mode section does not say the coach echoes")
    if not re.search(r"\bset\s+`INSIDEOUT_LLM_API_KEY`", text):
        fail(failures, "local-development.md's Offline AI mode section does not say coaching "
                       "needs a set INSIDEOUT_LLM_API_KEY")


def readme_scopes_github_evidence(failures):
    """rv4-FEAT-08: evidence attaches only to roadmap leaves a committed
    insideout.yaml matches, and only once the GitHub App's webhook delivers.
    While the guide matcher and the webhook route exist, every README
    paragraph that says GitHub evidence attaches must name both conditions."""
    if not (ROOT / "server/internal/guide/guide.go").exists() \
            or "/hooks/github" not in read("server/internal/api/github_webhook.go"):
        return  # no guide matcher or no webhook; nothing to qualify
    for block in re.split(r"\n\s*\n", read("README.md")):
        if "GitHub" in block and re.search(r"(?i)evidence|证据", block) \
                and re.search(r"(?i)attach|挂到", block):
            if "insideout.yaml" not in block:
                fail(failures, "README.md says GitHub evidence attaches without naming the "
                               f"insideout.yaml guide: {block[:48]!r}")
            if not re.search(r"(?i)webhook|GitHub App", block):
                fail(failures, "README.md says GitHub evidence attaches without naming the "
                               f"GitHub App webhook: {block[:48]!r}")


def readme_first_run_path(failures):
    """rv4-PM-08: a stranger's first success must not dead-end. Once a release
    exists the Install section leads with the prebuilt one-liner; the installer
    refusing other OSes is stated; invitation-only signup comes with a way to
    ask for an invite and a no-account try-out."""
    readme = read("README.md")
    start = readme.index("## 安装 / Install")
    end = re.search(r"^## ", readme[start + 1:], re.M)
    section = readme[start:start + 1 + end.start()] if end else readme[start:]
    tags = subprocess.run(["git", "-C", str(ROOT), "tag", "--list", "v*"],
                          capture_output=True, text=True, check=True).stdout.split()
    heads = re.findall(r"^### (.*)$", section, re.M)
    pre = next((i for i, h in enumerate(heads) if "prebuilt" in h), None)
    src = next((i for i, h in enumerate(heads) if "from source" in h), None)
    if tags and (pre is None or src is None or pre > src):
        fail(failures, "README's Install section does not put the prebuilt-binaries "
                       "subsection before building from source")
    if "unsupported OS" in read("scripts/bootstrap.sh"):
        for needle in ("macOS and Linux only", "Windows is not supported"):
            if needle not in section:
                fail(failures, f"README's Install section lacks '{needle}' "
                               "(scripts/bootstrap.sh refuses other OSes)")
    if "invitation_only" in read("server/db/migrations/20260825120000_invitation_signup.sql"):
        if not re.search(r"(?i)invite request", readme):
            fail(failures, "README says signup is invitation-only but names no way to ask for an invite")
        if "Try it without an account" not in section:
            fail(failures, "README's Install section has no 'Try it without an account' line")


if __name__ == "__main__":
    facts, failures = {}, []
    env_variables_documented(facts, failures)
    offline_coach_not_overclaimed(failures)
    readme_scopes_github_evidence(failures)
    readme_first_run_path(failures)
    for message in failures:
        print(message, file=sys.stderr)
    if failures:
        sys.exit(1)
    print(f"docs truth (rv4) ok: {facts['env variables']} INSIDEOUT_* variables documented")
