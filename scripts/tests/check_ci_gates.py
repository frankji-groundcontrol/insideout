#!/usr/bin/env python3
"""Fail when .github/workflows loses a gate review 4 found missing (rv4 wave 1).

Usage: python3 -I scripts/tests/check_ci_gates.py

Stdlib only. The workflows are read as text: the few properties below need no
YAML parser, and a runner may not ship one. A premise is derived from the tree
where one exists (the Go floor from server/go.mod, the guard set from
scripts/tests/), so a demand moves with the thing it is about. Exits 1 naming
every missing gate. This pins the text of the workflow; a worktree cannot run
one, so the first CI run after a push is what proves the behavior.

- rv4-PM-02: ci.yml runs on a weekly schedule, off the :00 and :30 marks, and
  the scheduled run shuffles test order, so a clock-pinned or order-dependent
  test cannot sit red between pushes.
- rv4-TEST-08: only a pull request's older run is cancelled; a quick follow-up
  push to main must not cancel the previous commit's only verdict. With
  cancel-in-progress off a run still waits behind the running one, and a third
  run then replaces the waiting one, so non-PR runs get a group of their own.
- rv4-PM-01, CLOSE-01, TEST-05, PUB-08: the env-suites job runs every
  scripts/tests/check_*.py with `python3 -I` from the repo root and a failing
  guard fails the step (the step's own script runs against a made-up tree), so
  a guard written red-first is not left for nobody to run. Its checkout fetches
  the tags, because check_docs_truth.py asks git for them.
- rv4-PM-12, PUB-09: the Ubuntu install lane accepts no Go older than
  server/go.mod's floor - it asks the go command (`GOTOOLCHAIN=local go list
  -m` in server/) instead of grepping `go version` for a hand-written range,
  so a floor bump moves the check with it - and no step name announces an
  older floor.
- rv4-PM-07: the analyzer ratchet (INFO_BASELINE in the client job) never
  allows more infos than the pinned toolchain measures (INFO_CEILING below).
- rv4-PM-10 (CI half): the server job fails on `go mod tidy -diff`, so a direct
  dependency marked indirect or a stale go.sum cannot sit in go.mod unseen.
"""
import pathlib
import re
import subprocess
import sys
import tempfile
import textwrap

ROOT = pathlib.Path(__file__).resolve().parents[2]
CI = ".github/workflows/ci.yml"
UBUNTU = ".github/workflows/ubuntu-install.yml"
# rv4-PM-07: the info lines `flutter analyze --no-fatal-infos` prints on the
# toolchain ci.yml pins (Flutter 3.44): 14. INFO_BASELINE may fall as infos are
# fixed; raising it past this is a regression, so raise this only on purpose.
INFO_CEILING = 14


def read(name):
    return (ROOT / name).read_text(encoding="utf-8")


def top(yml, key):
    """A top-level section: its `key:` line up to the next top-level key."""
    match = re.search(rf"^{key}:.*?(?=^\S|\Z)", yml, re.M | re.S)
    return match.group(0) if match else ""


def job(yml, name):
    """One job: its `  name:` line up to the next job key."""
    match = re.search(rf"^  {re.escape(name)}:\n.*?(?=^  [\w-]+:\n|\Z)", yml, re.M | re.S)
    return match.group(0) if match else ""


def steps(block):
    """The steps of a job block, one text chunk per `      - ` item."""
    return re.split(r"^(?=      - )", block, flags=re.M)[1:]


def workdir(block, step):
    """Where a step runs: its own working-directory, else the job default."""
    own = re.search(r"^\s+working-directory:\s*(\S+)", step, re.M)
    default = re.search(r"^    defaults:\n      run:\n        working-directory:\s*(\S+)", block, re.M)
    found = own or default
    return found.group(1) if found else "."


def script_of(step):
    """The `run: |` body of a step, dedented; None when it has none."""
    lines = step.splitlines()
    for i, line in enumerate(lines):
        match = re.match(r"^(\s*)run:\s*\|\s*$", line)
        if not match:
            continue
        body = []
        for nxt in lines[i + 1:]:
            if nxt.strip() and len(nxt) - len(nxt.lstrip()) <= len(match.group(1)):
                break
            body.append(nxt)
        return textwrap.dedent("\n".join(body))
    return None


def code(step):
    """A step without its comment-only lines (a comment may name a command)."""
    return "\n".join(l for l in step.splitlines() if not l.lstrip().startswith("#"))


def weekly_schedule():
    """PM-02: a weekly cron on a fixed minute that is not :00 or :30."""
    crons = re.findall(r"^\s+-\s*cron:\s*[\"']([^\"']+)[\"']", top(read(CI), "on"), re.M)
    if any(re.fullmatch(r"\d+ \d+ \* \* [0-7]", c) and int(c.split()[0]) not in (0, 30) for c in crons):
        return []
    return [
        f"{CI}: no weekly `schedule:` cron (a fixed minute that is not :00 or :30, "
        "one weekday) under `on:` - a time-based failure stays unseen between "
        "pushes (rv4-PM-02)"
    ]


def scheduled_shuffle():
    """PM-02: only the scheduled run shuffles, and the test command gets the flag."""
    for step in steps(job(read(CI), "server")):
        text = code(step)
        test = re.search(r"^\s*(go test \./\.\.\. .*-race.*)$", text, re.M)
        if not test:
            continue
        if "-shuffle=on" in test.group(1) and re.search(r"^\s+if:.*github\.event_name == 'schedule'", text, re.M):
            return []
        # NAME: ${{ github.event_name == 'schedule' && '-shuffle=on' || '-shuffle=off' }}
        for name, expr in re.findall(r"^\s+(\w+):\s*(\$\{\{.*\}\})\s*$", text, re.M):
            if ("github.event_name == 'schedule'" in expr and "-shuffle=on" in expr
                    and re.search(rf"\$\{{?{name}\b", test.group(1))):
                return []
    return [
        f"{CI}: no server-job step runs `go test ./... -race -shuffle=on` for the "
        "scheduled event only, with the flag reaching the command - an order "
        "dependence stays invisible (rv4-PM-02)"
    ]


def cancel_only_pull_requests():
    """TEST-08: cancel-in-progress is an expression on the pull_request event."""
    value = re.search(r"cancel-in-progress:\s*(.*)", top(read(CI), "concurrency"))
    if value and "github.event_name == 'pull_request'" in value.group(1):
        return []
    return [
        f"{CI}: concurrency cancels in-progress runs on every event - a quick "
        "follow-up push to main cancels the previous commit's only verdict; "
        "scope cancel-in-progress to pull_request (rv4-TEST-08)"
    ]


def pushes_get_their_own_group():
    """TEST-08: GitHub replaces a run that is waiting in a concurrency group
    when another is queued behind the running one, even without
    cancel-in-progress; a push to main needs a group no other run shares."""
    group = re.search(r"group:\s*(.*)", top(read(CI), "concurrency"))
    expr = group.group(1) if group else ""
    if ("github.run_id" in expr or "github.sha" in expr) and "github.ref" in expr:
        return []
    return [
        f"{CI}: concurrency group {expr!r}: key a pull request on github.ref and "
        "every other run on github.run_id - a group shared by pushes lets a third "
        "quick push replace the second one's waiting run, though nothing is "
        "cancelled in progress (rv4-TEST-08)"
    ]


def contract_guards_run():
    """PM-01, CLOSE-01, TEST-05, PUB-08: one glob step runs every guard and a
    failing guard fails it. The step's own script runs against a made-up tree,
    since a gate that cannot fail is the very bug this row closes."""
    block = job(read(CI), "env-suites")
    loop = next((s for s in steps(block) if "check_*.py" in s and "python3 -I" in s), None)
    if loop is None:
        return [
            f"{CI}: no env-suites step runs scripts/tests/check_*.py with "
            "`python3 -I` - the contract guards pass by hand and run in no CI "
            "job (rv4-PM-01, CLOSE-01, TEST-05, PUB-08)"
        ]
    script, where = script_of(loop), workdir(block, loop)
    if script is None or "${{" in script:
        return [f"{CI}: the guard step has no plain `run: |` script to exercise (rv4-PM-01)"]
    with tempfile.TemporaryDirectory() as tmp:
        tests = pathlib.Path(tmp, "scripts", "tests")
        tests.mkdir(parents=True)
        for name in ("one", "two"):
            (tests / f"check_{name}.py").write_text(f"print('ran {name}')\n")
        cwd = pathlib.Path(tmp, where)
        # bash -e is what an Actions `run:` gets when no shell is named.
        green = subprocess.run(["bash", "-e", "-c", script], cwd=cwd, capture_output=True, text=True)
        (tests / "check_two.py").write_text("import sys\nsys.exit(1)\n")
        red = subprocess.run(["bash", "-e", "-c", script], cwd=cwd, capture_output=True, text=True)
    problems = []
    if green.returncode != 0 or not all(f"ran {n}" in green.stdout for n in ("one", "two")):
        problems.append(
            f"{CI}: from {where!r} the guard step does not run every "
            "scripts/tests/check_*.py - the job's default directory is scripts/, "
            "so set `working-directory: .` (rv4-PM-01)"
        )
    if red.returncode == 0:
        problems.append(f"{CI}: a failing guard does not fail the guard step (rv4-PM-01)")
    return problems


def guards_see_the_tags():
    """PM-01: a default checkout has no tags, so CI would judge a different
    premise than a clone when a guard lists them (check_docs_truth.py does)."""
    me = pathlib.Path(__file__).name
    wants = any(
        re.search(r'"tag",\s*"--list"', p.read_text(encoding="utf-8"))
        for p in (ROOT / "scripts/tests").glob("check_*.py")
        if p.name != me
    )
    checkout = next((s for s in steps(job(read(CI), "env-suites")) if "actions/checkout" in s), "")
    if not wants or "fetch-tags: true" in checkout or "fetch-depth: 0" in checkout:
        return []
    return [
        f"{CI}: a guard lists git tags but the env-suites checkout fetches none "
        "(`fetch-tags: true`) - CI would judge a different premise than a clone (rv4-PM-01)"
    ]


def ubuntu_go_floor():
    """PM-12, PUB-09: the premise (go.mod's floor) is read from the tree by the go
    command itself - GOTOOLCHAIN=local makes it refuse an older toolchain - so
    the lane follows a floor bump instead of freezing the number it was written
    with (check_go_floor.py bans the hand-written range outright)."""
    yml = read(UBUNTU)
    major, floor = map(int, re.search(r"^go (\d+)\.(\d+)", read("server/go.mod"), re.M).groups())
    problems = []
    if not re.search(r"cd server && GOTOOLCHAIN=local go (?:list|build|vet|test)\b", yml):
        problems.append(
            f"{UBUNTU}: nothing asks the go command whether the installed Go meets "
            "server/go.mod; run `cd server && GOTOOLCHAIN=local go list -m` after the "
            "install block (rv4-PM-12, PUB-09)"
        )
    for named in re.findall(rf"Go {major}\.(\d+)\+", yml):
        if int(named) < floor:
            problems.append(
                f"{UBUNTU}: a step name says Go {major}.{named}+, below the "
                f"{major}.{floor} floor of server/go.mod (rv4-PM-12)"
            )
    return problems


def analyzer_ratchet():
    """PM-07: a baseline above the measured count is slack the tree can fill."""
    found = re.search(r"INFO_BASELINE:\s*\"?(\d+)\"?", job(read(CI), "client"))
    if found and int(found.group(1)) <= INFO_CEILING:
        return []
    return [
        f"{CI}: INFO_BASELINE is {found.group(1) if found else 'missing'}, past the "
        f"{INFO_CEILING} infos the pinned toolchain measures - slack that new "
        "infos can fill unseen (rv4-PM-07)"
    ]


def mod_tidy_checked():
    """PM-10: a step runs `go mod tidy -diff` where go.mod is, after setup-go."""
    block = job(read(CI), "server")
    chunks = [code(c) for c in steps(block)]
    tidy = next((i for i, c in enumerate(chunks) if "go mod tidy -diff" in c), None)
    if tidy is None:
        return [
            f"{CI}: no server-job step runs `go mod tidy -diff` - a direct "
            "dependency can sit marked indirect, and a stale go.sum stay, "
            "unseen (rv4-PM-10)"
        ]
    where, problems = workdir(block, chunks[tidy]), []
    if not (ROOT / where / "go.mod").is_file():
        problems.append(f"{CI}: `go mod tidy -diff` runs from {where!r}, which holds no go.mod (rv4-PM-10)")
    setup = next((i for i, c in enumerate(chunks) if "actions/setup-go" in c), None)
    if setup is None or setup > tidy:
        problems.append(f"{CI}: `go mod tidy -diff` runs before actions/setup-go (rv4-PM-10)")
    if re.search(r"\|\|\s*(true|:)(\s|$)", chunks[tidy]):
        problems.append(f"{CI}: the `go mod tidy -diff` step swallows its failure (rv4-PM-10)")
    return problems


CHECKS = [weekly_schedule, scheduled_shuffle, cancel_only_pull_requests,
          pushes_get_their_own_group, contract_guards_run, guards_see_the_tags,
          ubuntu_go_floor, analyzer_ratchet, mod_tidy_checked]

if __name__ == "__main__":
    failures = [message for check in CHECKS for message in check()]
    for message in failures:
        print(message, file=sys.stderr)
    if failures:
        sys.exit(1)
    print(f"ci gates ok: {len(CHECKS)} checks")
