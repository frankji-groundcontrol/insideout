#!/usr/bin/env python3
"""scripts/setup.sh is a shim — run: python3 scripts/tests/env/test_setup_shim.py

rv5-STRC-02: the Go wizard (`insideout setup`) is the only local-setup engine
and the only .env writer. setup.sh passes its flags and stdin to the installed
binary, or to `go run ./cmd/insideout setup` in server/ when no binary is on
PATH, from the checkout it lives in. Stub `insideout` and `go` commands record
what they were given; nothing real runs.
"""

from __future__ import annotations

import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

sys.path[:0] = [str(Path(__file__).resolve().parent), str(Path(__file__).resolve().parents[2])]  # here, scripts/
from env_testlib import REPO, check, run  # noqa: E402

BASH = shutil.which("bash")
RECORD = 'printf "%s\\n" "$PWD" "$@" > "$SHIM_LOG"; cat > "$SHIM_LOG.stdin"; exit 7'


def shim(args, stubs, stdin=""):
    """Run setup.sh on a PATH holding only dirname and the named stubs.
    Returns (exit code, stderr, recorded cwd + argv lines, recorded stdin,
    whether the env file appeared)."""
    with tempfile.TemporaryDirectory() as d:
        tmp = Path(d)
        bindir = tmp / "bin"
        bindir.mkdir()
        (bindir / "dirname").symlink_to(shutil.which("dirname"))
        (bindir / "cat").symlink_to(shutil.which("cat"))
        for name in stubs:
            (bindir / name).write_text(f"#!{BASH}\n{RECORD}\n")
            (bindir / name).chmod(0o755)
        log, env_file = tmp / "log", tmp / "scratch.env"
        env = {"PATH": str(bindir), "HOME": str(tmp), "LC_ALL": "C",
               "SHIM_LOG": str(log), "INSIDEOUT_ENV_FILE": str(env_file)}
        done = subprocess.run([BASH, str(REPO / "scripts" / "setup.sh"), *args], cwd=tmp, env=env,
                              input=stdin, capture_output=True, text=True, timeout=60)
        lines = log.read_text().splitlines() if log.exists() else []
        piped = Path(f"{log}.stdin").read_text() if Path(f"{log}.stdin").exists() else ""
        return done.returncode, done.stderr, lines, piped, env_file.exists()


def t_forwards_every_flag_to_the_installed_binary():
    flags = ["--local", "--postgres", "own", "--no-start", "--create-roles"]
    code, _, lines, piped, wrote = shim(flags, ["insideout", "go"], stdin="postgres://u@h/db\n")
    check(f"the binary got `setup {' '.join(flags)}` (got {lines[1:]})", lines[1:] == ["setup", *flags])
    check("it runs from the checkout the script lives in", lines[:1] == [str(REPO)])
    check("the DSN on stdin reaches the binary", piped == "postgres://u@h/db\n")
    check(f"the binary's exit code is the script's (got {code})", code == 7)
    check("the shim writes no .env of its own", not wrote)


def t_hosted_and_no_flags_forward_too():
    for flags in (["--hosted"], [], ["--check", "--llm"]):
        _, _, lines, _, _ = shim(flags, ["insideout"])
        check(f"`setup.sh {' '.join(flags)}` forwards as `insideout setup {' '.join(flags)}`",
              lines[1:] == ["setup", *flags])


def t_falls_back_to_go_run_in_server():
    flags = ["--local", "--postgres", "bundled"]
    code, _, lines, _, _ = shim(flags, ["go"])
    check(f"without the binary it runs `go run ./cmd/insideout setup ...` (got {lines[1:]})",
          lines[1:] == ["run", "./cmd/insideout", "setup", *flags])
    check("go runs in server/", lines[:1] == [str(REPO / "server")])
    check("go's exit code is the script's", code == 7)


def t_says_what_to_install_when_neither_is_there():
    code, err, lines, _, wrote = shim(["--local", "--postgres", "own"], [])
    check(f"exits non-zero (got {code})", code != 0)
    check("nothing ran", lines == [])
    check(f"names the installer: {err.strip()!r}", "scripts/install.sh" in err)
    check("writes no .env", not wrote)


if not BASH:
    print("no bash on PATH", file=sys.stderr)
    raise SystemExit(2)
raise SystemExit(run(globals()))
