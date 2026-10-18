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
  push to main must not cancel the previous commit's only verdict.
"""
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parents[2]
CI = ".github/workflows/ci.yml"


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
    """PM-02: the scheduled run is the one that shuffles test order."""
    tests = [s for s in steps(job(read(CI), "server")) if re.search(r"go test \./\.\.\. .*-race", s)]
    if any("-shuffle=on" in s and "github.event_name == 'schedule'" in s for s in tests):
        return []
    return [
        f"{CI}: no server-job step runs `go test ./... -race -shuffle=on` for the "
        "scheduled event - an order dependence stays invisible (rv4-PM-02)"
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


CHECKS = [weekly_schedule, scheduled_shuffle, cancel_only_pull_requests]

if __name__ == "__main__":
    failures = [message for check in CHECKS for message in check()]
    for message in failures:
        print(message, file=sys.stderr)
    if failures:
        sys.exit(1)
    print("ci gates ok: weekly schedule with a shuffled scheduled run")
