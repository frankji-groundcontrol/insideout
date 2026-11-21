"""Shared helpers for the scripts/tests/check_*.py gates.

A gate runs as `python3 -I scripts/tests/check_<subject>.py`; -I keeps the
script's own folder off sys.path, so each gate inserts it before importing
this module (the same insert scripts/tests/keys/_helpers.py makes):

    HERE = os.path.dirname(os.path.abspath(__file__))
    if HERE not in sys.path:
        sys.path.insert(0, HERE)
    from _gate import ROOT, read, fail  # noqa: E402
"""
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parents[2]


def read(name):
    """Text of a repo-relative file."""
    return (ROOT / name).read_text(encoding="utf-8")


def fail(failures, message):
    failures.append(message)


def cannot_derive(message):
    """Exit 2: a premise the gate derives from the tree is missing."""
    print(message, file=sys.stderr)
    sys.exit(2)


def compose_service(name):
    """Text of one service of docker-compose.yml, or None. The file is a flat
    services map, so a cut at the next two-space key needs no YAML parser."""
    match = re.search(rf"^  {re.escape(name)}:\n(.*?)(?=^  [A-Za-z_][\w-]*:|\Z)",
                      read("docker-compose.yml"), re.M | re.S)
    return match.group(1) if match else None
