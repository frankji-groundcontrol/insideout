"""Shared harness for the per-key tests.

The tree mirrors the key store itself: one module folder per store
module, one test file per key, filename = the key name. The harness
locates a key's `.key` file across every host under hosts/ and skips
cleanly where it is absent (a CI checkout carries manifests but no
values). Tests read bytes only to validate shape and placement — a
value is never printed, logged, or compared to a literal.
"""

import os
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


def read_bytes(path):
    with open(path, "rb") as fh:
        return fh.read()
