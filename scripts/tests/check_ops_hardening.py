#!/usr/bin/env python3
"""Pin the operator and first-run hardening of review-4 wave 1 (rv4-USER-06).

Usage: python3 -I scripts/tests/check_ops_hardening.py

Stdlib only. Every property is read from file text, so the guard needs no
Docker, database or network. Each check derives its premise from the tree
first and the demand disappears with the premise (the style of
check_docs_truth.py). Exits 1 naming every offending property; exit 2 when a
premise cannot be derived.
"""
import pathlib
import re
import shlex
import sys

ROOT = pathlib.Path(__file__).resolve().parents[2]


def read(name):
    return (ROOT / name).read_text(encoding="utf-8")


def fail(failures, message):
    failures.append(message)


def flag_after_positional(args):
    """True when a flag follows the first positional in a shell-style command
    tail. Every flag of the CRUD verbs takes a value, so `--flag value` is one
    unit and the value is not mistaken for the positional."""
    try:
        tokens = shlex.split(args, comments=True)
    except ValueError:  # prose with a stray quote: fall back to whitespace
        tokens = args.split()
    seen_positional = skip_value = False
    for token in tokens:
        if skip_value:
            skip_value = False
        elif token.startswith("-"):
            if seen_positional:
                return True
            skip_value = "=" not in token
        else:
            seen_positional = True
    return False


def cli_wants_id_last():
    """The CLI's own usage line for `prd update`: flags first, then the id."""
    usages = []
    for path in sorted((ROOT / "server/cmd/insideout").glob("*.go")):
        if not path.name.endswith("_test.go"):
            usages += re.findall(r"insideout prd update ([^\"\n]*)", path.read_text(encoding="utf-8"))
    if not usages:
        print("no `insideout prd update` usage line under server/cmd/insideout", file=sys.stderr)
        sys.exit(2)
    return any(usage.rstrip().endswith("<id>") for usage in usages)


def readme_prd_update_flags_precede_id(failures):
    """USER-06: Go's flag package stops at the first positional, so an example
    that puts the id before a flag parses no flag and prints only the usage."""
    if not cli_wants_id_last():
        return  # the CLI takes the id first; the example may follow it
    for no, line in enumerate(read("README.md").splitlines(), start=1):
        match = re.search(r"\bprd update\b(.*)", line)
        if match and flag_after_positional(match.group(1)):
            fail(failures, f"README.md:{no}: `prd update` puts the id before a flag; Go's flag "
                           "package stops at the first positional, so the flag is never parsed "
                           "and the CLI prints only its usage line")


if __name__ == "__main__":
    failures = []
    readme_prd_update_flags_precede_id(failures)
    for message in failures:
        print(message, file=sys.stderr)
    if failures:
        sys.exit(1)
    print("ops hardening ok: README prd update example parses")
