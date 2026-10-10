#!/usr/bin/env python3
"""Fail when doc prose claims something the tree contradicts (review-3 wave 3).

Usage: python3 -I scripts/tests/check_docs_truth.py

Every check first derives its premise from the tree (HANDOFF's deploy line,
the CI workflow, git tags, the MCP tool registry, prds.go's section keys)
and then asserts the doc agrees, so the prose cannot silently re-drift when
reality moves: when the premise disappears the demand disappears with it.
Exits 1 naming every offending claim (review-3 PM-02, PM-03, ARCH-04,
ARCH-05/FEAT-08, USER-05, COMP-02, COMP-04, REL-04).
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


def client_readme_matches_handoff(failures):
    """PM-02: while HANDOFF says a deploy is pending, client/README must not
    claim production serves this tree."""
    if not re.search(r"Nothing from \d{4}-\d{2}-\d{2} on is deployed", read("docs/HANDOFF.md")):
        return  # no pending-deploy record; nothing to reconcile
    readme = read("client/README.md")
    if "serves this tree" in readme:
        fail(failures, "client/README.md claims production 'serves this tree' while "
                       "docs/HANDOFF.md records an undeployed span")
    for needle in ("serves a release of this tree", "behind `main`", "HANDOFF"):
        if needle not in readme:
            fail(failures, f"client/README.md lacks '{needle}' — the deploy-skew qualifier")


def board_row_matches_ci(failures):
    """PM-03: a CI android release-APK job voids the board's blocked row."""
    if "flutter build apk --release" not in read(".github/workflows/ci.yml"):
        return  # no android release build in CI; the row may stay blocked
    row = next((line for line in read("docs/plans/README.md").splitlines()
                if "2026-08-17-flutter-client.md" in line), "")
    if not row:
        fail(failures, "docs/plans/README.md lost the 2026-08-17-flutter-client row")
        return
    if "In flight" in row or "blocked" in row:
        fail(failures, "plans/README.md's Nuxt→Flutter row is In flight/Android-blocked "
                       "while ci.yml's android job builds release APKs")
    if "**Completed**" not in row:
        fail(failures, "plans/README.md's Nuxt→Flutter row is not closed as Completed")


def backend_tree_matches_layout(failures):
    """ARCH-04: the backend Layout tree names both binaries, the CLI layer,
    and every internal package that exists."""
    backend = read("docs/architecture/backend.md")
    packages = sorted(p.name for p in (ROOT / "server/internal").iterdir() if p.is_dir())
    for name in packages:
        if not re.search(rf"^\s+{re.escape(name)}/\s", backend, re.M):
            fail(failures, f"backend.md's Layout tree omits internal/{name}/")
    if "insideout-mcp" not in backend:
        fail(failures, "backend.md's Layout tree omits the cmd/insideout-mcp binary")
    if not re.search(r"^\s+cmd/insideout/.*cli", backend, re.M | re.I):
        fail(failures, "backend.md's cmd/insideout line omits the cli*.go CLI/setup layer")


def main_comment_defers_to_cli_go(failures):
    """ARCH-04: main.go's doc comment points at cli.go's dispatch instead of
    enumerating product CLI verbs that go stale."""
    src = read("server/cmd/insideout/main.go")
    comment = src[: src.index("package main")]
    if "cli.go" not in comment:
        fail(failures, "cmd/insideout/main.go's doc comment does not point at cli.go's dispatch")
    for match in re.finditer(r"//\s*insideout (login|whoami|workspaces|projects|prd)\b", comment):
        fail(failures, f"main.go still enumerates CLI verb '{match.group(1)}' in its doc comment")


def mcp_tool_counts(facts, failures):
    """ARCH-05/FEAT-08: any stated MCP tool count equals the registry, and
    the docs point at tools/list as the source of truth."""
    names = set()
    for path in sorted((ROOT / "server/internal/mcpserver").glob("*.go")):
        if path.name.endswith("_test.go"):
            continue
        text = path.read_text(encoding="utf-8")
        names |= set(re.findall(r'addTextTool\("([a-z0-9_]+)"', text))
        names |= set(re.findall(r'addTool\("([a-z0-9_]+)"', text))
    facts["mcp tools"] = len(names)
    for doc in ("README.md", "docs/usage/cli-mcp-and-guidance.md"):
        text = read(doc)
        if "tools/list" not in text:
            fail(failures, f"{doc} does not point at tools/list for the MCP tool set")
        for pattern in (r"\b(\d+)\s+tools\b", r"(\d+)\s*个工具", r"\((\d+)\s+today\)"):
            for match in re.finditer(pattern, text):
                if int(match.group(1)) != len(names):
                    fail(failures, f"{doc} states {match.group(1)} MCP tools; "
                                   f"the registry defines {len(names)}")


def readme_quickstart_works_today(failures):
    """USER-05: with no v* tag, the Install section's first command is not
    the dead-end curl|bash; the hosted web path comes first."""
    tags = subprocess.run(["git", "-C", str(ROOT), "tag", "--list", "v*"],
                          capture_output=True, text=True, check=True).stdout.split()
    if tags:
        return  # a release exists; the curl one-liner is no longer a dead end
    readme = read("README.md")
    start = readme.index("## 安装 / Install")
    section = readme[start:][:]
    end = re.search(r"^## ", readme[start + 1:], re.M)
    if end:
        section = section[: end.start() + 1]
    blocks = re.findall(r"```(?:bash|sh)?\n(.*?)```", section, re.S)
    if not blocks:
        fail(failures, "README's Install section has no command block to start from")
        return
    if "releases/latest/download" in blocks[0]:
        fail(failures, "README's Install section leads with the curl|bash block, "
                       "which 404s until the first v* tag")
    if "./scripts/install.sh" not in blocks[0]:
        fail(failures, "README's Install section does not lead with a working "
                       "command (./scripts/install.sh)")
    if "### 网页端" not in section.split("```")[0]:
        fail(failures, "README's Install section does not lead with the hosted web path")


def product_baseline_scopes_sections(failures):
    """COMP-02: while prds.go validates fixed section keys, PRODUCT.md's
    superseded-assumptions paragraph must scope that as the storage scaffold."""
    prds = read("server/internal/store/prds.go")
    block = re.search(r"var PrdSectionKeys = \[\]string\{(.*?)\}", prds, re.S)
    if not block or not re.findall(r'"[a-zA-Z]+"', block.group(1)):
        return  # no fixed key list in code; the assumption may be fully retired
    paragraph = next((p for p in read("PRODUCT.md").split("\n\n")
                      if "supersedes the earlier product assumptions" in p), "")
    if not paragraph:
        fail(failures, "PRODUCT.md lost the superseded-assumptions paragraph")
        return
    for needle in ("scaffold", "prds.go"):
        if needle not in paragraph:
            fail(failures, f"PRODUCT.md's superseded-assumptions paragraph lacks the "
                           f"'{needle}' scoping of the eight section keys")


def handoff_records_mail_state(failures):
    """COMP-04: while the board still records 'all five flows exercised
    live', HANDOFF must carry the mail/lead-15 reconciliation line."""
    if "all five flows exercised live" not in read("docs/plans/README.md"):
        return  # board record gone; the contradiction is settled
    handoff = read("docs/HANDOFF.md")
    for needle in ("all five flows", "owner-blocked", "pre-wave"):
        if needle not in handoff:
            fail(failures, f"docs/HANDOFF.md lacks the mail-state record ('{needle}') "
                           f"against plans/README.md's five-flows row")


def backup_doc_states_reality(failures):
    """REL-04: while nothing schedules backup-db.sh, the runbook must say so
    and state the operator's duties."""
    candidates = [p for p in list((ROOT / ".github/workflows").glob("*.yml"))
                  + list((ROOT / "scripts").glob("*")) if p.is_file()]
    schedulers = [str(p.relative_to(ROOT)) for p in candidates
                  if p.name != "backup-db.sh" and "backup-db.sh" in p.read_text(encoding="utf-8", errors="replace")]
    if schedulers:
        return  # something schedules the dump now; the honesty line may go
    doc = read("docs/usage/backup-restore.md")
    if not re.search(r"(?i)nothing schedules|no cron|unscheduled", doc):
        fail(failures, "backup-restore.md does not state that nothing schedules the dump")
    if not re.search(r"(?i)after every migration|cadence", doc):
        fail(failures, "backup-restore.md does not state the operator's dump cadence duty")


if __name__ == "__main__":
    facts, failures = {}, []
    client_readme_matches_handoff(failures)
    board_row_matches_ci(failures)
    backend_tree_matches_layout(failures)
    main_comment_defers_to_cli_go(failures)
    mcp_tool_counts(facts, failures)
    readme_quickstart_works_today(failures)
    product_baseline_scopes_sections(failures)
    handoff_records_mail_state(failures)
    backup_doc_states_reality(failures)
    for message in failures:
        print(message, file=sys.stderr)
    if failures:
        sys.exit(1)
    print(f"docs truth ok: {facts['mcp tools']} MCP tools registered, "
          f"{len(list((ROOT / 'server/internal').iterdir()))} internal packages in backend.md")
