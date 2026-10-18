"""Shared harness for the per-key tests.

The tree mirrors the key store itself: one module folder per store
module, one test file per key, filename = the key name. The harness
locates a key's `.key` file across every host under hosts/ and skips
cleanly where it is absent (a CI checkout carries manifests but no
values). Tests read bytes only to validate shape and placement — a
value is never printed, logged, or compared to a literal.
"""

import os
import re
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
SCRIPTS = os.path.dirname(os.path.dirname(HERE))  # scripts/
REPO = os.path.dirname(SCRIPTS)
for p in (HERE, SCRIPTS):
    if p not in sys.path:
        sys.path.insert(0, p)

import keys  # noqa: E402


def key_hosts(key_name):
    """[(host, key_path, env_var)] for every host whose manifests map
    `key_name` and whose `.key` file exists on this machine."""
    out = []
    for host in keys.discover_hosts(REPO):
        for var, (_slug, path) in keys.key_map(REPO, host).items():
            if os.path.basename(path) == f"{key_name}.key" and os.path.exists(path):
                out.append((host, path, var))
    return out


def key_placements(key_name):
    """[(host, key_path, env_var)] for every host whose manifests map
    `key_name` — existence NOT required (the never-absent rule is the
    caller's to assert once the host is present)."""
    out = []
    for host in keys.discover_hosts(REPO):
        for var, (_slug, path) in keys.key_map(REPO, host).items():
            if os.path.basename(path) == f"{key_name}.key":
                out.append((host, path, var))
    return out


def present_hosts():
    """Hosts with at least one `.key` file on this machine (a CI
    checkout carries manifests but no values)."""
    present = set()
    for host in keys.discover_hosts(REPO):
        for _var, (_slug, path) in keys.key_map(REPO, host).items():
            if os.path.exists(path):
                present.add(host)
                break
    return sorted(present)


def never_absent_cases(key_name):
    """[(host, key_path, env_var)] for present hosts — the caller
    asserts the file exists (a kind may be blank, never absent)."""
    present = set(present_hosts())
    return [(h, p, v) for h, p, v in key_placements(key_name) if h in present]


def read_bytes(path):
    with open(path, "rb") as fh:
        return fh.read()


def without_terminal_newline(raw):
    """Return a key value with one conventional final LF removed.

    ``keys.py fill`` writes values without a final LF, while an operator may
    edit a single-line key with a text editor that adds one.  A second LF, or
    an LF anywhere else, remains in the returned value and is rejected by
    ``is_single_line``.
    """
    return raw[:-1] if raw.endswith(b"\n") else raw


def is_single_line(raw):
    """Whether ``raw`` has no embedded LF and at most one terminal LF."""
    return b"\n" not in without_terminal_newline(raw)


def is_optional_blank(raw):
    """Whether an optional key is empty, allowing one editor-added final LF."""
    return without_terminal_newline(raw) == b""


def assert_single_line(testcase, raw, message):
    """Assert line shape without letting unittest render the secret operand."""
    testcase.assertTrue(is_single_line(raw), message)


def assert_regex(testcase, pattern, value, message):
    """Assert a pattern without unittest rendering ``value`` on failure."""
    testcase.assertTrue(re.search(pattern, value) is not None, message)
