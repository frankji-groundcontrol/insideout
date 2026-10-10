#!/usr/bin/env python3
"""Fail while the contribution and security doors are missing (review-3 PM-05).

Usage: python3 -I scripts/tests/check_contribution_surface.py

An MPL-2.0 repo with a hosted multi-user deployment needs a contributor door
(CONTRIBUTING.md) and a private security-disclosure door (SECURITY.md), both
linked from README.md's learn-more table; without them the working contract
lives only in agent-facing files and a hosted-instance finding has only the
public issue tracker. Every premise is derived from the tree — the repo slug
from the bootstrap URL README.md already publishes, the hosted instance from
the CLI's shipped default base, the CI lane from ci.yml — so when a premise
disappears the demand disappears with it. Exits 1 naming every missing door
or dangling pointer.
"""
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parents[2]


def read(name):
    return (ROOT / name).read_text(encoding="utf-8")


def repo_slug():
    """Derive owner/repo from the install URL the README already exposes."""
    match = re.search(r"https://github\.com/([\w.-]+)/([\w.-]+)/",
                      read("README.md"))
    return f"{match.group(1)}/{match.group(2)}" if match else None


def doors_exist(failures):
    """PM-05: the two root doors must exist at all."""
    for name, door in (("CONTRIBUTING.md", "contributor"),
                       ("SECURITY.md", "security-disclosure")):
        if not (ROOT / name).exists():
            failures.append(f"{name} is missing — the repo has no {door} door")


def readme_links_both(failures):
    """PM-05: the learn-more table must link both doors in one row."""
    row = re.compile(r"\[`CONTRIBUTING\.md`\]\(CONTRIBUTING\.md\)"
                     r".*\[`SECURITY\.md`\]\(SECURITY\.md\)")
    if not any(row.search(line) for line in read("README.md").splitlines()):
        failures.append("README.md's learn-more table has no row linking "
                        "CONTRIBUTING.md and SECURITY.md")


def contributing_points_at_the_contract(failures):
    """PM-05: CONTRIBUTING is thin and points at the real contract."""
    if not (ROOT / "CONTRIBUTING.md").exists():
        return  # doors_exist already reported it
    text = read("CONTRIBUTING.md")
    for anchor, premise in (
            ("AGENTS.md", ROOT / "AGENTS.md"),
            ("docs/changelogs", ROOT / "docs" / "changelogs"),
            (".github/workflows/ci.yml", ROOT / ".github" / "workflows" / "ci.yml"),
            ("English-only", ROOT / "AGENTS.md")):
        if premise.exists() and anchor not in text:
            failures.append(f"CONTRIBUTING.md does not point at '{anchor}'")


def security_discloses_privately(failures):
    """PM-05: SECURITY.md names a private channel, the hosting scope and the
    review records, without publishing anything the repo keeps redacted."""
    if not (ROOT / "SECURITY.md").exists():
        return  # doors_exist already reported it
    text = read("SECURITY.md")
    slug = repo_slug()
    if slug and f"github.com/{slug}/security/advisories" not in text:
        failures.append("SECURITY.md lacks the private GitHub security-"
                        "advisory path for this repo")
    for anchor, premise in (
            ("default_base.go", ROOT / "server" / "internal" / "apiclient"
                                / "default_base.go"),
            ("docs/review", ROOT / "docs" / "review")):
        if premise.exists() and anchor not in text:
            failures.append(f"SECURITY.md does not point at '{anchor}'")


def main():
    failures = []
    doors_exist(failures)
    readme_links_both(failures)
    contributing_points_at_the_contract(failures)
    security_discloses_privately(failures)
    if failures:
        print("contribution surface is missing its doors:", file=sys.stderr)
        for failure in failures:
            print(f"  - {failure}", file=sys.stderr)
        sys.exit(1)
    print("contribution surface: CONTRIBUTING.md and SECURITY.md present and linked")


if __name__ == "__main__":
    main()
