#!/usr/bin/env python3
"""Fail when the record-system docs contradict the tree (review-4 wave 3).

Usage: python3 -I scripts/tests/check_docs_truth_rv4.py

Companion to check_docs_truth.py, split out to keep both under the line
budget. Same method: derive the premise from the tree first, then assert the
doc agrees, so the prose cannot silently re-drift. Exits 1 naming every
offending claim (rv4-FULL-09, rv4-USER-08, rv4-FEAT-08, rv4-PM-08,
rv4-USER-09 README half, rv4-ARCH-06, rv4-PM-03, rv4-PM-09).
"""
import fnmatch
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


def readme_does_not_mislead_prebuilt_users(failures):
    """rv4-USER-09, README half (residue R1-06): the prebuilt install has no
    repository checkout, so a README line that offers `setup --local` says it
    needs one; and `promote-admin` creates an absent account, which is what a
    first run without an invite relies on, so the README says that too."""
    readme = read("README.md")
    if "no repository checkout found" in read("server/cmd/insideout/cli_setup_local_env.go"):
        for line in readme.splitlines():
            if "setup --local" in line and not re.search(r"(?i)checkout|检出", line):
                fail(failures, "README offers 'setup --local' without saying it needs a repository "
                               f"checkout, which the prebuilt binaries lack: {line.strip()[:60]!r}")
    if "createAdminAccount(" in read("server/cmd/insideout/promote_admin.go"):
        for block in re.split(r"\n\s*\n", readme):
            if "promote-admin" in block and not re.search(r"(?i)creates? the account", block):
                fail(failures, "README names promote-admin but not that it creates the account "
                               "when the address has none")


CLIENT_DOC = "docs/architecture/cli-mcp-and-client-config.md"


def client_surface_documented(failures):
    """rv4-ARCH-06: the CLI, MCP and client-config surface is half the product,
    so it has its own architecture doc, reachable from both indexes and from
    backend.md, that names what the code decides: every variable
    clientconfig.Resolve reads, the hosted-default fallback and the
    single-replica advisory-lock key."""
    resolve = ROOT / "server/internal/clientconfig/resolve.go"
    if not resolve.exists() or not (ROOT / "server/cmd/insideout-mcp/main.go").exists():
        return  # no shared client resolution or no MCP binary; nothing to describe
    for index in ("docs/architecture/index.md", "docs/index.md", "docs/architecture/backend.md"):
        if "cli-mcp-and-client-config.md" not in read(index):
            fail(failures, f"{index} does not link cli-mcp-and-client-config.md")
    if not (ROOT / CLIENT_DOC).exists():
        fail(failures, f"{CLIENT_DOC} is missing: no architecture doc describes the "
                       "CLI, MCP and client-config surface")
        return
    doc, code = read(CLIENT_DOC), resolve.read_text(encoding="utf-8")
    wanted = set(re.findall(r'os\.Getenv\("(INSIDEOUT_[A-Z_]+)"\)', code))
    if "apiclient.DefaultBase" in code:
        wanted.add("DefaultBase")
    lock = re.search(r'singletonLockKey = "([^"]+)"', read("server/internal/api/single_replica.go"))
    if lock:
        wanted.add(lock.group(1))
    for term in sorted(wanted):
        if term not in doc:
            fail(failures, f"{CLIENT_DOC} does not name {term}, which the code decides")
    for target in ("agent-token-scope.md", "../usage/cli-mcp-and-guidance.md", "../usage/deployment.md"):
        if (ROOT / "docs/architecture" / target).exists() and target not in doc:
            fail(failures, f"{CLIENT_DOC} does not link {target}")


def section(text, title):
    found = re.search(rf"^## {re.escape(title)}\n(.*?)(?=^## |\Z)", text, re.M | re.S)
    return found.group(1) if found else ""


def backend_states_current_facts(failures):
    """rv4-ARCH-06: backend.md's current-state claims match the code: the
    line-budget guard's real threshold, a Bearer-first authentication paragraph,
    and store and api file lists (globs allowed) that cover every file."""
    backend = read("docs/architecture/backend.md")
    tests = "".join(p.read_text(encoding="utf-8")
                    for p in (ROOT / "server/internal/linebudget").glob("*_test.go"))
    budget = re.search(r"const budget = (\d+)", tests)
    layout = next((l for l in backend.splitlines() if l.lstrip().startswith("linebudget/")), "")
    if budget and (budget.group(1) not in layout or "~350" in layout):
        fail(failures, f"backend.md's linebudget line does not state the enforced budget of {budget.group(1)}")
    if "Bearer" in read("server/internal/api/cookies.go") \
            and ("httpOnly-cookie based" in backend or "Bearer" not in backend):
        fail(failures, "backend.md calls authentication httpOnly-cookie based, but "
                       "requireAuth prefers a Bearer header and only then the cookie")
    for title, directory in (("Store layer", "server/internal/store"), ("API surface", "server/internal/api")):
        tokens = re.findall(r"[a-z][a-z0-9_*]*\.go", section(backend, title))
        for path in sorted((ROOT / directory).glob("*.go")):
            if not path.name.endswith("_test.go") \
                    and not any(fnmatch.fnmatchcase(path.name, t) for t in tokens):
                fail(failures, f"backend.md's {title} section does not name {directory}/{path.name}")


COUNT_WORDS = ("no", "one", "two", "three", "four", "five", "six", "seven", "eight",
               "nine", "ten", "eleven", "twelve")


def open_slices_visible(failures):
    """rv4-PM-03: engineering work that remains must be visible from the resume
    path and the board. While a product-promise slice record exists, HANDOFF may
    not say engineering is closed, must link the enumeration and state how many
    slices are open, and the board must carry a row for the set that is not
    marked finished."""
    gaps = "docs/issues/2026-10-11-product-promise-gaps.md"
    if not (ROOT / gaps).exists():
        return  # the enumeration is retired; every slice shipped
    listed = re.findall(r"^\d+\. \[[^\]]+\]\((2026-[a-z0-9-]+\.md)\)", read(gaps), re.M)
    count = len([name for name in listed if (ROOT / "docs/issues" / name).exists()])
    if not count:
        return
    handoff = read("docs/HANDOFF.md")
    if re.search(r"(?i)everything engineering is closed", handoff):
        fail(failures, f"docs/HANDOFF.md says engineering is closed, but {count} product-promise slices are open")
    if "product-promise-gaps.md" not in handoff:
        fail(failures, "docs/HANDOFF.md does not link the product-promise slice order "
                       "(docs/issues/2026-10-11-product-promise-gaps.md)")
    word = COUNT_WORDS[count] if count < len(COUNT_WORDS) else str(count)
    if not re.search(rf"(?i)\b({count}|{word})\b[\s\w-]{{0,40}}slices?", handoff):
        fail(failures, f"docs/HANDOFF.md does not say that {count} product-promise slices are open")
    row = next((l for l in read("docs/plans/README.md").splitlines() if "product-promise-gaps.md" in l), "")
    if not row:
        fail(failures, "docs/plans/README.md has no board row for the open product-promise slices")
    elif re.search(r"\*\*(Finished|Completed|Done|Closed|Fixed)", row) or "Pending" not in row:
        fail(failures, "docs/plans/README.md's product-promise row is not Pending")


def handoff_flutter_versions_agree(failures):
    """rv4-PM-09: HANDOFF is read as current, so a Flutter version it states
    must be on the minor the repo pins (ci.yml); it may not warn about one SDK
    in one section and record another in the next."""
    pinned = re.search(r"flutter-version:\s*(\d+\.\d+)\.\d+", read(".github/workflows/ci.yml"))
    if not pinned:
        return  # no pinned SDK in CI; nothing to compare against
    for block in re.split(r"\n\s*\n", read("docs/HANDOFF.md")):
        if "Flutter" not in block:
            continue
        for version in re.findall(r"\b\d+\.\d+\.\d+\b", block):
            if not version.startswith(pinned.group(1) + "."):
                fail(failures, f"docs/HANDOFF.md states Flutter {version}, but CI pins {pinned.group(1)}.x")


if __name__ == "__main__":
    facts, failures = {}, []
    env_variables_documented(facts, failures)
    offline_coach_not_overclaimed(failures)
    readme_scopes_github_evidence(failures)
    readme_first_run_path(failures)
    readme_does_not_mislead_prebuilt_users(failures)
    client_surface_documented(failures)
    backend_states_current_facts(failures)
    open_slices_visible(failures)
    handoff_flutter_versions_agree(failures)
    for message in failures:
        print(message, file=sys.stderr)
    if failures:
        sys.exit(1)
    print(f"docs truth (rv4) ok: {facts['env variables']} INSIDEOUT_* variables documented")
