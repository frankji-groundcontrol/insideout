#!/usr/bin/env python3
"""Pin the first-run copy that lives in the install and setup scripts
(review-4 wave 3: rv4-PM-08 code half, LEAD-SETUP-BLURB).

Usage: python3 -I scripts/tests/check_first_run_copy.py

Stdlib only; no network, database or Go toolchain. The scripts are run, not
just read, wherever they stop before doing any work: scripts/install.sh stops
at its Go check when no go is on PATH, and scripts/bootstrap.sh stops at its
OS check when uname names another system. Each check derives its premise from
the tree first (the style of check_ops_hardening.py). Exits 1 naming every
offending property; exit 2 when a premise cannot be derived.
"""
import os
import pathlib
import re
import shutil
import subprocess
import sys
import tempfile

ROOT = pathlib.Path(__file__).resolve().parents[2]
BASH = shutil.which("bash")


def read(name):
    return (ROOT / name).read_text(encoding="utf-8")


def fail(failures, message):
    failures.append(message)


def header(name):
    """The comment block that opens a script, up to its first command."""
    lines = []
    for line in read(name).splitlines()[1:]:  # past the shebang
        if line and not line.startswith("#"):
            break
        lines.append(line)
    return "\n".join(lines)


def run(script, args, tools, fake=None):
    """Run `bash scripts/<script> <args>` on a PATH that holds only `tools`
    (real commands) and `fake` (name -> sh body standing in for a command),
    with a scratch HOME. `{scratch}` in `args` is that directory. Returns
    (exit code, stdout, stderr)."""
    if not BASH:
        print("no bash on PATH", file=sys.stderr)
        sys.exit(2)
    with tempfile.TemporaryDirectory() as scratch:
        scratch = pathlib.Path(scratch)
        bindir = scratch / "bin"
        bindir.mkdir()
        for name in tools:
            real = shutil.which(name)
            if not real:
                print(f"no {name} on PATH", file=sys.stderr)
                sys.exit(2)
            (bindir / name).symlink_to(real)
        for name, body in (fake or {}).items():
            (bindir / name).write_text(f"#!/bin/sh\n{body}\n", encoding="utf-8")
            (bindir / name).chmod(0o755)
        env = {"PATH": str(bindir), "HOME": str(scratch), "LC_ALL": "C"}
        args = [arg.replace("{scratch}", str(scratch)) for arg in args]
        done = subprocess.run([BASH, str(ROOT / "scripts" / script), *args], cwd=scratch, env=env,
                              capture_output=True, text=True, timeout=60)
        return done.returncode, done.stdout, done.stderr


def prebuilt_one_liner():
    """The command the README tells a stranger to run, derived from the
    release base that scripts/bootstrap.sh downloads from."""
    script = read("scripts/bootstrap.sh")
    repo = re.search(r"^repo=(\S+)$", script, re.M)
    base = re.search(r"^base=\$\{INSIDEOUT_RELEASE_BASE:-(\S+)\}$", script, re.M)
    if not repo or not base:
        print("no repo= or base= default in scripts/bootstrap.sh", file=sys.stderr)
        sys.exit(2)
    return "curl -fsSL " + base.group(1).replace("$repo", repo.group(1)) + "/bootstrap.sh | bash"


def installer_headers_state_the_supported_systems(failures):
    """PM-08: the prebuilt installer exits on any OS but macOS and Linux and
    nothing said so up front; both installers now open by naming the supported
    systems and that Windows is not one of them."""
    for name in ("scripts/bootstrap.sh", "scripts/install.sh"):
        text = header(name)
        if "macOS and Linux" not in text:
            fail(failures, f"{name}: the opening comment does not say macOS and Linux are the supported systems")
        if "Windows" not in text:
            fail(failures, f"{name}: the opening comment does not say Windows is not supported")


def bootstrap_refuses_another_system_by_naming_the_supported_two(failures):
    """PM-08: on any other OS the prebuilt installer stops before downloading
    and says which systems it covers."""
    code, _, err = run("bootstrap.sh", [], tools=[], fake={"uname": "echo MINGW64_NT-10.0"})
    if code != 1 or "macOS and Linux" not in err:
        fail(failures, f"scripts/bootstrap.sh on an unsupported OS: exit {code}, stderr {err.strip()!r}; "
                       "want exit 1 naming macOS and Linux")


def install_without_go_offers_the_prebuilt_installer_first(failures):
    """PM-08: a stranger without a Go toolchain was told to install one; the
    prebuilt installer needs none, so the error names it first, with the
    systems it covers, and keeps the from-source route second."""
    one_liner = prebuilt_one_liner()
    code, _, err = run("install.sh", ["{scratch}/installed"], tools=["dirname", "mkdir"])
    if code != 1:
        fail(failures, f"scripts/install.sh with no go on PATH: exit {code}, want 1")
        return
    if one_liner not in err:
        fail(failures, f"scripts/install.sh with no go on PATH does not print the prebuilt one-liner `{one_liner}`")
        return
    if "macOS and Linux" not in err:
        fail(failures, "scripts/install.sh with no go on PATH does not say the prebuilt installer covers macOS and Linux")
    if "Go 1." not in err or err.index(one_liner) > err.index("Go 1."):
        fail(failures, "scripts/install.sh with no go on PATH names the Go toolchain route before the prebuilt installer")


CHECKS = [installer_headers_state_the_supported_systems,
          bootstrap_refuses_another_system_by_naming_the_supported_two,
          install_without_go_offers_the_prebuilt_installer_first]

if __name__ == "__main__":
    failures = []
    for check in CHECKS:
        check(failures)
    for message in failures:
        print(message, file=sys.stderr)
    if failures:
        sys.exit(1)
    print(f"first-run copy ok: {len(CHECKS)} checks")
