#!/usr/bin/env python3
"""Fail when the smoke suite cannot see the hosted box or the providers.

Usage: python3 -I scripts/tests/check_smoke_wiring.py

The nightly smoke boots its own server on a throwaway database, so by
itself it can never notice a broken hosted deploy (review-3 TEST-03), and
both external providers are exercised only through fakes the codebase
itself authored (review-3 TEST-05). This check pins the wiring that closes
both gaps, deriving its premises from the tree so a demand disappears when
its premise does:

- the public origin comes from apiclient.DefaultBase (the CLI's hosted API),
  so a moved deployment moves the smoke with it;
- the admin mail probe route (review-3 ARCH-03) is only demanded while
  server/internal/api still mounts it — it is the one mail surface whose
  response tells the truth about delivery.

Everything else is a structural demand on .github/workflows/smoke.yml and
server/scripts/smoke.sh: a hosted dispatch lane that probes the deployment
read-only with the authenticated probe gated on INSIDEOUT_TOKEN (a missing
secret skips, never fails), provider keys passed through from repo
secrets, and smoke.sh legs that run for real when a key exists and print a
SKIP line naming the missing variable otherwise. Exits 1 naming every
offending demand; exit 2 when a premise cannot be derived.
"""
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parents[2]

WORKFLOW = ".github/workflows/smoke.yml"
SMOKE = "server/scripts/smoke.sh"
DEFAULT_BASE = "server/internal/apiclient/default_base.go"
MAIL_PROBE_ROUTE = "/api/v1/admin/mail/probe"


def read(name):
    return (ROOT / name).read_text(encoding="utf-8")


def public_origin():
    """The hosted deployment's origin, from the CLI's DefaultBase constant."""
    text = read(DEFAULT_BASE)
    match = re.search(r'DefaultBase = "([^"]+)"', text)
    if not match:
        print(f"no DefaultBase constant in {DEFAULT_BASE}", file=sys.stderr)
        sys.exit(2)
    base = match.group(1)
    if "/api/v1" not in base:
        print(f"DefaultBase {base!r} carries no /api/v1 to strip", file=sys.stderr)
        sys.exit(2)
    return base[: base.index("/api/v1")]


def dispatch_block(yml):
    """The workflow_dispatch section, up to the next key at its level or top."""
    lines = yml.splitlines()
    # workflow_dispatch: sits indented under `on:` — match it at any depth.
    start = next((i for i, l in enumerate(lines) if l.lstrip().startswith("workflow_dispatch:")), None)
    if start is None:
        return ""
    # The block ends at the next key indented no deeper (a sibling under
    # `on:`) or at any top-level key (permissions:, jobs:, ...). Blank
    # lines and the section's own deeper-indented children stay inside.
    end = next(
        (
            i
            for i in range(start + 1, len(lines))
            if lines[i].strip()
            and len(lines[i]) - len(lines[i].lstrip()) <= 2
            and re.match(r"\s*[A-Za-z-]+:", lines[i])
        ),
        len(lines),
    )
    return "\n".join(lines[start:end])


def smoke_yml_failures(failures, origin):
    yml = read(WORKFLOW)
    block = dispatch_block(yml)

    def want(condition, message):
        if not condition:
            failures.append(f"{WORKFLOW}: {message}")

    # TEST-03: a hosted dispatch mode must exist and be selectable.
    want(
        re.search(r"^\s*-\s*hosted\s*$", block, re.M) or re.search(r"options:.*hosted", block),
        "no 'hosted' option under workflow_dispatch — the suite cannot be "
        "aimed at the deployed stack (TEST-03)",
    )
    want(
        "inputs.mode == 'hosted'" in yml,
        "no lane is gated on inputs.mode == 'hosted' — a hosted dispatch "
        "would run the same throwaway-database job as the nightly (TEST-03)",
    )
    want(
        f"SMOKE_BASE: {origin}" in yml,
        f"no lane sets SMOKE_BASE: {origin} (apiclient.DefaultBase's origin) "
        "— nothing probes the public deployment (TEST-03)",
    )
    want(
        re.search(r'^\s*SMOKE_HOSTED: ["\']?1', yml, re.M) is not None,
        "no lane sets SMOKE_HOSTED — smoke.sh cannot know to stay read-only "
        "against the deployment (TEST-03)",
    )
    want(
        "secrets.INSIDEOUT_TOKEN" in yml,
        "the hosted lane does not pass INSIDEOUT_TOKEN — the authenticated "
        "probe can never run, and a missing secret must skip, not fail "
        "(TEST-03)",
    )
    # TEST-05: the provider keys must reach smoke.sh from repo secrets.
    for name, what in [
        ("INSIDEOUT_LLM_API_KEY", "LLM"),
        ("INSIDEOUT_MAIL_API_KEY", "mail"),
        ("SMOKE_MAIL_TO", "mail sink recipient"),
    ]:
        want(
            f"secrets.{name}" in yml,
            f"{what} provider leg can never run for real: secrets.{name} is "
            "not passed through (TEST-05)",
        )


def smoke_sh_failures(failures):
    sh = read(SMOKE)

    def want(condition, message):
        if not condition:
            failures.append(f"{SMOKE}: {message}")

    # TEST-05: each provider leg runs for real when its key exists and says
    # why it cannot otherwise — silence is the defect the review filed.
    want(
        re.search(r"\b(if|elif)\b.*INSIDEOUT_LLM_API_KEY", sh) is not None,
        "no branch on INSIDEOUT_LLM_API_KEY — the LLM leg cannot run for "
        "real when the key is configured (TEST-05)",
    )
    want(
        re.search(r"SKIP.*INSIDEOUT_LLM_API_KEY", sh) is not None,
        "no SKIP line names INSIDEOUT_LLM_API_KEY — a skipped LLM leg is "
        "silent, which is what the review filed (TEST-05)",
    )
    want(
        re.search(r"\b(if|elif)\b.*INSIDEOUT_MAIL_API_KEY", sh) is not None,
        "no branch on INSIDEOUT_MAIL_API_KEY — the mail leg cannot run for "
        "real when the key is configured (TEST-05)",
    )
    want(
        re.search(r"SKIP.*INSIDEOUT_MAIL_API_KEY", sh) is not None,
        "no SKIP line names INSIDEOUT_MAIL_API_KEY — a skipped mail leg is "
        "silent (TEST-05)",
    )
    # The five-surface server must stay deterministic: the offline-template
    # force has to be per-command on that boot, because a script-wide
    # export would also strip the key from the provider leg's own server.
    want(
        'INSIDEOUT_LLM_API_KEY= "$TMP/insideout"' in sh,
        "the booted server is not offline per-command "
        "('INSIDEOUT_LLM_API_KEY= \"$TMP/insideout\"') — either the five "
        "surfaces lost their determinism or a global export strips the key "
        "from the provider leg's server (TEST-05)",
    )
    # Background server boots only: the build (-o) and the mail leg's
    # foreground promote-admin call do not count. Two boots = the
    # deterministic five-surface server plus the leg's key-carrying one.
    launches = len(re.findall(r'"\$TMP/insideout".*&\s*$', sh, re.M))
    want(
        launches >= 2,
        f"{launches} background server boot(s) — the LLM leg boots no "
        "key-carrying server of its own (TEST-05)",
    )
    # The mail leg must go through the one surface whose response reports
    # the provider's verdict — demanded only while the API still mounts it.
    api_files = [p for p in (ROOT / "server/internal/api").glob("*.go") if not p.name.endswith("_test.go")]
    if any(MAIL_PROBE_ROUTE in p.read_text(encoding="utf-8") for p in api_files):
        want(
            MAIL_PROBE_ROUTE in sh,
            f"the mail leg does not call {MAIL_PROBE_ROUTE} — the auth "
            "flows answer 200 regardless of delivery, so a shape break "
            "would stay invisible (TEST-05)",
        )


if __name__ == "__main__":
    failures = []
    smoke_yml_failures(failures, public_origin())
    smoke_sh_failures(failures)
    for message in failures:
        print(message, file=sys.stderr)
    if failures:
        sys.exit(1)
    print("smoke wiring ok: hosted dispatch lane + provider legs present")
