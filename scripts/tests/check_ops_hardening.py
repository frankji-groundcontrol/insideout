#!/usr/bin/env python3
"""Pin the operator and first-run hardening of review-4 wave 1 (rv4-USER-06,
rv4-REL-05).

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


def server_shutdown_seconds():
    """The Shutdown window runServe grants (its `shutdownCtx` timeout), in seconds."""
    src = read("server/cmd/insideout/main.go")
    match = re.search(r"shutdownCtx, cancel := context\.WithTimeout\([^,]+, (\d+)\*time\.Second\)", src)
    if not match:
        print("no shutdownCtx timeout in server/cmd/insideout/main.go", file=sys.stderr)
        sys.exit(2)
    return int(match.group(1))


def compose_service(name):
    """Text of one service of docker-compose.yml. The file is a flat services
    map, so a cut at the next two-space key needs no YAML parser."""
    match = re.search(rf"^  {re.escape(name)}:\n(.*?)(?=^  [A-Za-z_][\w-]*:|\Z)",
                      read("docker-compose.yml"), re.M | re.S)
    if not match:
        print(f"no `{name}` service in docker-compose.yml", file=sys.stderr)
        sys.exit(2)
    return match.group(1)


def compose_seconds(value):
    """A compose duration (`35s`, `1m30s`) in seconds, or None when malformed."""
    if not re.fullmatch(r"(\d+[hms])+", value):
        return None
    unit = {"h": 3600, "m": 60, "s": 1}
    return sum(int(n) * unit[u] for n, u in re.findall(r"(\d+)([hms])", value))


def compose_grace_outlives_shutdown(failures):
    """REL-05: runServe gives Shutdown a fixed window (a 20 s drain plus the
    coach's 10 s terminal writes), but Docker SIGKILLs a stopping container
    after its 10 s default. Compose must set stop_grace_period past the window
    and the architecture doc must say why."""
    window = server_shutdown_seconds()
    match = re.search(r"^    stop_grace_period:\s*(\S+)\s*(?:#.*)?$", compose_service("server"), re.M)
    if not match:
        fail(failures, "docker-compose.yml: the server service has no stop_grace_period, so Docker's "
                       f"10 s default SIGKILLs it inside runServe's {window} s Shutdown window")
    else:
        grace = compose_seconds(match.group(1).strip("\"'"))
        if grace is None or grace <= window:
            fail(failures, f"docker-compose.yml: the server's stop_grace_period ({match.group(1)}) "
                           f"must exceed runServe's {window} s Shutdown window")
    if "stop_grace_period" not in read("docs/architecture/deployment.md"):
        fail(failures, "docs/architecture/deployment.md does not mention the server's stop_grace_period")


CHECKS = [readme_prd_update_flags_precede_id, compose_grace_outlives_shutdown]

if __name__ == "__main__":
    failures = []
    for check in CHECKS:
        check(failures)
    for message in failures:
        print(message, file=sys.stderr)
    if failures:
        sys.exit(1)
    print(f"ops hardening ok: {len(CHECKS)} checks")
