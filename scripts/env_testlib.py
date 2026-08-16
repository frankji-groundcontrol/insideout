"""Shared harness for the env-toolchain test files.

Plain asserts, no framework — the whole suite is two files plus this, with no
fixtures and no dependencies. Kept separate only so each test file stays inside
the repo's ~350-line budget and covers one responsibility:

    test_env_catalog.py   reading   — the schema convention, states, masking,
                                      attribution, parsing, and the linters
                                      that hold THIS repo's skeletons honest
    test_env_writes.py    writing   — set_var / clear_var / quote / env_path,
                                      permissions, and the one piece of
                                      rendering logic testable without a tty

Run them the same way:  python3 scripts/test_env_catalog.py
                        python3 scripts/test_env_writes.py
"""

from __future__ import annotations

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import env_catalog as cat  # noqa: E402

REPO = Path(__file__).resolve().parent.parent

_ok = 0
_fail = 0


def check(label: str, cond: bool) -> None:
    global _ok, _fail
    if cond:
        _ok += 1
    else:
        _fail += 1
        print(f"  FAIL  {label}")


def scaffold(tmp: Path, root_example: str, root_env: str = "",
             comp: dict | None = None) -> Path:
    """A throwaway repo: a root skeleton, an optional .env, component contracts."""
    (tmp / ".env.example").write_text(root_example)
    if root_env:
        (tmp / ".env").write_text(root_env)
    for name, body in (comp or {}).items():
        d = tmp / cat.COMPONENT_DIRS[name]
        d.mkdir(parents=True, exist_ok=True)
        (d / ".env.example").write_text(body)
    return tmp


def run(namespace: dict) -> int:
    """Run every `t_*` in `namespace`, print the tally, return an exit code."""
    for fn in [v for k, v in sorted(namespace.items()) if k.startswith("t_")]:
        fn()
    print(f"{_ok} passed, {_fail} failed")
    return 1 if _fail else 0
