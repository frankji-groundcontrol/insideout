#!/usr/bin/env python3
"""Writing the environment — run: python3 scripts/test_env_writes.py

The write path: set_var / clear_var / quote / env_path, file permissions, and
_clip — the one piece of env_tui's rendering logic that can be exercised
without a terminal. The read path is scripts/test_env_catalog.py; the shared
harness is env_testlib.py.

Several of these are regressions from the 2026-07-30 adversarial review, where
a writer was found disagreeing with its own reader. See
docs/practices/2026-07-30-verifying-a-tty-only-tool.md.
"""

from __future__ import annotations

import os
import sys
import tempfile
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import env_catalog as cat  # noqa: E402
from env_testlib import check, run  # noqa: E402


def t_set_var_replaces_in_place_and_preserves_the_rest():
    with tempfile.TemporaryDirectory() as d:
        root = Path(d)
        (root / ".env").write_text("A=1\n# a comment\n#B=\nC=3\n")
        cat.set_var(root, "B", "two")
        got = (root / ".env").read_text()
        vals = cat.parse_env_file(root / ".env")
        check("commented key is uncommented in place", vals["B"] == "two")
        check("kept its position", got.splitlines()[2] == "B=two")
        check("siblings preserved", vals["A"] == "1" and vals["C"] == "3")
        check("comment preserved", "# a comment" in got)
        check("no duplicate key", got.count("B=") == 1)
        check("file is chmod 600", oct((root / ".env").stat().st_mode)[-3:] == "600")


def t_set_var_appends_when_the_key_is_absent():
    with tempfile.TemporaryDirectory() as d:
        root = Path(d)
        (root / ".env").write_text("A=1\n")
        cat.set_var(root, "NEW", "v")
        check("absent key appended", cat.parse_env_file(root / ".env")["NEW"] == "v")


def t_set_var_handles_a_dsn():
    # A DSN is full of / : @ & — the reason set_var rebuilds lines instead of
    # running a sed substitution.
    with tempfile.TemporaryDirectory() as d:
        root = Path(d)
        (root / ".env").write_text("DATABASE_URL=\n")
        dsn = "postgres://insideout_app:p@ss/w:rd&x@10.0.0.1:5432/insideout?sslmode=require"
        cat.set_var(root, "DATABASE_URL", dsn)
        check("DSN round-trips intact",
              cat.parse_env_file(root / ".env")["DATABASE_URL"] == dsn)


def t_clear_var_round_trips_to_the_skeleton_form():
    with tempfile.TemporaryDirectory() as d:
        root = Path(d)
        (root / ".env").write_text("A=1\nGITHUB_TOKEN=ghp_x\nC=3\n")
        cat.clear_var(root, "GITHUB_TOKEN")
        got = (root / ".env").read_text()
        check("cleared key is commented out, not deleted", "#GITHUB_TOKEN=" in got)
        check("value is gone", "ghp_x" not in got)
        check("no longer parsed as set", "GITHUB_TOKEN" not in cat.parse_env_file(root / ".env"))
        check("siblings preserved", got.splitlines()[0] == "A=1" and got.splitlines()[2] == "C=3")
        cat.set_var(root, "GITHUB_TOKEN", "again")
        check("and set_var finds it again in place",
              (root / ".env").read_text().splitlines()[1] == "GITHUB_TOKEN=again")


def t_quote_round_trips_awkward_values():
    for raw in ["two words", "with'single", 'with"double', "with$dollar",
                "back`tick", "trailing ", "a=b c", "plain", "", "p@ss/w:rd&x"]:
        line = f"K={cat.quote(raw)}"
        with tempfile.TemporaryDirectory() as d:
            p = Path(d) / ".env"
            p.write_text(line + "\n")
            got = cat.parse_env_file(p).get("K", "<missing>")
        check(f"quote round-trips {raw!r} (got {got!r})", got == raw)


def t_writes_target_the_last_occurrence():
    """Every reader resolves a duplicated key to the LAST assignment.

    Editing the first would save a value nothing then uses, while reporting it
    saved — the tool disagreeing with bash, compose and c12 at once.
    """
    with tempfile.TemporaryDirectory() as d:
        root = Path(d)
        (root / ".env").write_text("K=first\n# note\nK=second\n")
        cat.set_var(root, "K", "edited")
        lines = (root / ".env").read_text().splitlines()
        check("first occurrence untouched", lines[0] == "K=first")
        check("last occurrence edited", lines[2] == "K=edited")
        check("effective value is the edit", cat.parse_env_file(root / ".env")["K"] == "edited")
        cat.clear_var(root, "K")
        lines = (root / ".env").read_text().splitlines()
        check("clear also targets the last", lines[2] == "#K=" and lines[0] == "K=first")


def t_env_path_honours_the_testing_override():
    with tempfile.TemporaryDirectory() as d:
        root = Path(d)
        scratch = root / "scratch.env"
        check("default is <root>/.env", cat.env_path(root) == root / ".env")
        os.environ["INSIDEOUT_ENV_FILE"] = str(scratch)
        try:
            check("override is honoured", cat.env_path(root) == scratch)
            cat.set_var(root, "K", "v")
            check("and writes land there", scratch.is_file()
                  and not (root / ".env").exists())
        finally:
            del os.environ["INSIDEOUT_ENV_FILE"]


def t_unreadable_env_degrades_instead_of_raising():
    with tempfile.TemporaryDirectory() as d:
        p = Path(d) / ".env"
        p.write_text("A=1\n")
        p.chmod(0o000)
        try:
            check("unreadable .env parses as empty, no traceback",
                  cat.parse_env_file(p) == {})
        finally:
            p.chmod(0o600)


def t_tui_clip_measures_columns_not_characters():
    """The one piece of rendering logic that IS testable headlessly.

    curses' `n` counts characters; a CJK glyph occupies two columns, so a plain
    slice lets a bilingual line spill past the edge. Every hint in this repo
    carries Chinese, so this is the common case, not the exotic one.
    """
    import env_tui as tui
    for text, cols in [("plain ascii", 10), ("中文提示很长很长", 10),
                       ("a中b文c", 5), ("", 8), ("中", 1), ("x", 0)]:
        out = tui._clip(text, cols)
        width = sum(tui._width(c) for c in out)
        check(f"clip({text!r}, {cols}) occupies exactly {cols} columns", width == cols)

raise SystemExit(run(globals()))
