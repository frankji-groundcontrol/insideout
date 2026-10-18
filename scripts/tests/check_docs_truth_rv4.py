#!/usr/bin/env python3
"""Fail when the record-system docs contradict the tree (review-4 wave 3).

Usage: python3 -I scripts/tests/check_docs_truth_rv4.py

Companion to check_docs_truth.py, split out to keep both under the line
budget. Same method: derive the premise from the tree first, then assert the
doc agrees, so the prose cannot silently re-drift. Exits 1 naming every
offending claim (rv4-FULL-09).
"""
import pathlib
import re
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


if __name__ == "__main__":
    facts, failures = {}, []
    env_variables_documented(facts, failures)
    for message in failures:
        print(message, file=sys.stderr)
    if failures:
        sys.exit(1)
    print(f"docs truth (rv4) ok: {facts['env variables']} INSIDEOUT_* variables documented")
