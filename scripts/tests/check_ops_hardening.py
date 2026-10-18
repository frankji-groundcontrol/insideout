#!/usr/bin/env python3
"""Pin the operator and first-run hardening of review-4 wave 1 (rv4-USER-06,
rv4-REL-05, rv4-USER-09, rv4-COMP-07).

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


def cli_local_needs_checkout():
    """True while `insideout setup --local` refuses to run outside a repository checkout."""
    return any("no repository checkout found" in path.read_text(encoding="utf-8")
               for path in (ROOT / "server/cmd/insideout").glob("*.go")
               if not path.name.endswith("_test.go"))


def bootstrap_closing_message_omits_setup_local(failures):
    """USER-09: the prebuilt installer ships two binaries and no checkout, and
    `setup --local` exits 2 there, so its closing message must not offer it.
    The interactive flow, --hosted and --check run anywhere and stay."""
    if not cli_local_needs_checkout():
        return  # --local runs without a checkout; the line may stay
    script = read("scripts/bootstrap.sh")
    match = re.search(r"cat <<EOF\n(.*?)\nEOF(?:\n|\Z)", script, re.S)
    if not match:
        print("no closing `cat <<EOF` message in scripts/bootstrap.sh", file=sys.stderr)
        sys.exit(2)
    message, first = match.group(1), script[:match.start(1)].count("\n") + 1
    for no, line in enumerate(message.splitlines(), start=first):
        if "setup --local" in line:
            fail(failures, f"scripts/bootstrap.sh:{no}: the closing message offers `setup --local`, which "
                           "exits 2 without a repository checkout, and the prebuilt installer ships none")
    for needle in (r"insideout setup\s+# interactive", r"insideout setup --hosted", r"insideout setup --check"):
        if not re.search(needle, message):
            fail(failures, f"scripts/bootstrap.sh: the closing message lost its `{needle}` line")


def schema_holds_credential_hashes():
    """True while a migration stores password, token or code hashes the dump would carry."""
    return any(re.search(r"\b(password|token|code)_hash\b", path.read_text(encoding="utf-8"))
               for path in (ROOT / "server/db/migrations").glob("*.sql"))


def backup_dump_is_owner_only(failures):
    """COMP-07: the dump carries password, refresh-token, emailed-token and
    invitation-code hashes plus workspace join codes, and pg_dump creates its
    file with the caller's umask (0644 under the usual 022). The script must
    set umask 077 before the dump, re-assert chmod 600 after it, and the
    runbook must say what the file holds."""
    if not schema_holds_credential_hashes():
        return  # nothing sensitive in the schema; the default umask is fine
    code = "\n".join(line for line in read("scripts/backup-db.sh").splitlines()
                     if not line.lstrip().startswith("#"))
    dump = re.search(r"^\s*pg_dump\b", code, re.M)
    if not dump:
        print("no pg_dump call in scripts/backup-db.sh", file=sys.stderr)
        sys.exit(2)
    umask = re.search(r"^\s*umask\s+0*77\s*(?:#.*)?$", code, re.M)
    if not umask or umask.start() > dump.start():
        fail(failures, "scripts/backup-db.sh: no `umask 077` before the pg_dump call, so the dump is "
                       "written with the shell's umask (mode 0644 under the usual 022)")
    if not re.search(r'^\s*chmod\s+0?600\s+"?\$\{?file\}?"?\s*(?:#.*)?$', code[dump.start():], re.M):
        fail(failures, 'scripts/backup-db.sh: no `chmod 600 "$file"` after the pg_dump call')
    doc = read("docs/usage/backup-restore.md")
    for needle, claim in ((r"password hashes", "holds password hashes"),
                          (r"refresh-token hashes", "holds refresh-token hashes"),
                          (r"\b0600\b", "is written with mode 0600")):
        if not re.search(needle, doc, re.I):
            fail(failures, f"docs/usage/backup-restore.md does not say the dump {claim}")


CHECKS = [readme_prd_update_flags_precede_id, compose_grace_outlives_shutdown,
          bootstrap_closing_message_omits_setup_local, backup_dump_is_owner_only]

if __name__ == "__main__":
    failures = []
    for check in CHECKS:
        check(failures)
    for message in failures:
        print(message, file=sys.stderr)
    if failures:
        sys.exit(1)
    print(f"ops hardening ok: {len(CHECKS)} checks")
