#!/usr/bin/env python3
"""Interactive editor for InsideOut's environment — `scripts/env.sh edit`.

A scrollable list of every variable the repo knows about: what is set, what
merely came from `.env.example`, what is required and missing, and which
component reads it. ↑/↓ move, Enter sets, `c` clears, `/` filters, `q` quits.

All of the logic lives in env_catalog.py; this file only draws. curses needs a
tty, so keeping the two apart is what makes the behaviour testable at all —
`--list` renders the same rows non-interactively, and that is the surface the
test_env_*.py files assert on.

Secrets are masked in BOTH directions: shown as ••••••, typed as ******. The
catalog decides by variable NAME, before any value reaches the screen.

Writes only the root .env. app/.env is generated from it by `env.sh propagate`,
which this offers to re-run on exit when anything changed.
"""

from __future__ import annotations

import curses
import os
import subprocess
import sys
import unicodedata
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import env_catalog as cat  # noqa: E402

#: `default` gets its own mark so you can see at a glance which values you
#: chose and which merely came from the skeleton.
MARK = {"missing": "!", "placeholder": "~", "unset": "·", "default": "=", "set": "✓"}


def _pair(status: str) -> int:
    try:
        return curses.color_pair(
            {"missing": 1, "placeholder": 2, "set": 3, "unset": 4, "default": 4}[status])
    except curses.error:
        return curses.A_NORMAL      # a terminal with no colour still renders


def _try(fn, *args):
    """Call an optional curses capability; None if the terminal lacks it."""
    try:
        return fn(*args)
    except curses.error:
        return None


def _width(ch: str) -> int:
    """Terminal columns one character occupies (1, or 2 for CJK/fullwidth)."""
    return 2 if unicodedata.east_asian_width(ch) in "WF" else 1


def _clip(text: str, cols: int) -> str:
    """Truncate to `cols` terminal COLUMNS and pad to exactly that width.

    curses' `n` argument counts characters, not columns, so a plain `text[:n]`
    lets a CJK string occupy twice its budget and spill past the edge. The
    skeleton comments in this repo are bilingual, so the hint line hits this on
    most rows.
    """
    out, used = [], 0
    for ch in text:
        cw = _width(ch)
        if used + cw > cols:
            break
        out.append(ch)
        used += cw
    return "".join(out) + " " * (cols - used)


def put(scr, y: int, x: int, text: str, attr: int = curses.A_NORMAL) -> None:
    """Write a line, clamped so it can never touch the bottom-right cell.

    curses refuses a write that would fill the last column of the last row (it
    implies a scroll) and raises `addnwstr() returned ERR`. A full-width status
    line at h-1 therefore crashes the app — in the reference implementation
    that happened on the first Enter keypress, with 29 unit tests green. Clamp
    to w-1 and swallow the error at the edges so a narrow terminal degrades
    instead of aborting.
    """
    h, w = scr.getmaxyx()
    if not (0 <= y < h) or w <= 1:
        return
    room = w - x - 1
    if room <= 0:
        return
    try:
        scr.addstr(y, x, _clip(text, room), attr)
    except curses.error:
        pass


class Editor:
    def __init__(self, root: Path):
        self.root = root
        self.vars = cat.build(root)
        #: The unfiltered catalog. The header must always describe the WHOLE
        #: environment — summarising the filtered view reports "0 yours" the
        #: moment you press `/`, which reads as data loss.
        self.all = list(self.vars)
        self.filtered = False
        self.sel = 0
        self.top = 0
        self.dirty: set[str] = set()
        self.msg = ""

    # ---- drawing ---------------------------------------------------------
    def draw(self, scr) -> None:
        scr.erase()
        h, w = scr.getmaxyx()
        body = max(1, h - 6)          # header(2) + hint(1) + keys(1) + msg(1)
        self._scroll(body)

        put(scr, 0, 0, " InsideOut environment ", curses.A_REVERSE)
        shown = "" if not self.filtered else f"  [showing {len(self.vars)}]"
        put(scr, 1, 0, f" {cat.env_path(self.root)} — {cat.summary(self.all)}{shown}",
            curses.A_DIM)

        for i in range(body):
            idx = self.top + i
            if idx >= len(self.vars):
                break
            self._row(scr, 2 + i, self.vars[idx], idx == self.sel)

        if self.vars:
            put(scr, h - 3, 0, f" {self._hint(self.vars[self.sel])}", curses.A_DIM)
        put(scr, h - 2, 0,
            " ↑/↓ move · enter set · c clear · / filter outstanding · q quit ",
            curses.A_REVERSE)
        if self.msg:
            put(scr, h - 1, 0, f" {self.msg}", curses.A_BOLD)
        scr.refresh()

    def _row(self, scr, y: int, v: cat.Var, active: bool) -> None:
        attr = curses.A_REVERSE if active else curses.A_NORMAL
        req = "required" if v.required else "optional"
        who = ",".join(v.components) or "server"
        line = (f" {MARK[v.status]} {v.name:<29} {v.display()[:26]:<28} "
                f"{req:<9} {who}")
        put(scr, y, 0, line, attr | _pair(v.status))

    def _hint(self, v: cat.Var) -> str:
        """Help text for the selected row.

        A skeleton comment block attaches to the FIRST key under it, so later
        keys in the same block have none. Fall back to what we do know rather
        than showing a bare `NAME:` with nothing after it.
        """
        if v.comment:
            return f"{v.name}: {v.comment}"
        bits = ["required" if v.required else "optional"]
        bits.append("read by " + (", ".join(v.components) or "the Go server"))
        if v.example:
            bits.append(f"example: {v.example}")
        if v.status == "default":
            bits.append("currently the .env.example value, not one you chose")
        return f"{v.name}: " + " · ".join(bits)

    def _scroll(self, body: int) -> None:
        self.sel = max(0, min(self.sel, len(self.vars) - 1))
        if self.sel < self.top:
            self.top = self.sel
        elif self.sel >= self.top + body:
            self.top = self.sel - body + 1

    # ---- editing ---------------------------------------------------------
    def prompt(self, scr, v: cat.Var) -> str | None:
        """Read a value. Secrets echo as `*`, so nothing sensitive reaches the
        screen or a scrollback buffer. Esc cancels; empty input keeps the
        current value (clearing is the explicit `c` key instead)."""
        h, w = scr.getmaxyx()
        label = f" {v.name} = " + ("(hidden) " if v.secret else "")
        buf: list[str] = []
        _try(curses.curs_set, 1)          # optional capability, see run()
        try:
            while True:
                shown = "*" * len(buf) if v.secret else "".join(buf)
                put(scr, h - 1, 0, label + shown, curses.A_BOLD)
                try:
                    scr.move(h - 1, min(len(label) + len(shown), w - 2))
                except curses.error:
                    pass
                scr.refresh()
                ch = scr.get_wch()
                if ch == "\x1b":                          # Esc
                    return None
                if ch in ("\n", "\r", curses.KEY_ENTER):
                    return "".join(buf)
                if ch in ("\x7f", "\b", curses.KEY_BACKSPACE, 263):
                    if buf:
                        buf.pop()
                elif isinstance(ch, str) and ch.isprintable():
                    buf.append(ch)
        finally:
            _try(curses.curs_set, 0)

    def _reload(self, keep: str) -> None:
        """Rebuild after a write and follow the row that moved.

        Re-sorting relocates the edited variable (it just became `set`); without
        following it the cursor silently lands on an unrelated one — which is
        how Piper's TUI overran its target and clobbered a real value.
        """
        self.all = cat.build(self.root)
        self.vars = ([v for v in self.all if v.status != "set"]
                     if self.filtered else list(self.all))
        self.sel = next((i for i, x in enumerate(self.vars) if x.name == keep),
                        min(self.sel, max(0, len(self.vars) - 1)))

    def apply(self, v: cat.Var, value: str | None) -> None:
        if value is None:
            cat.clear_var(self.root, v.name)
            self.msg = f"{v.name} cleared in .env"
        else:
            cat.set_var(self.root, v.name, value)
            self.msg = f"{v.name} saved to .env"
        self.dirty.add(v.name)
        self._reload(v.name)

    # ---- loop ------------------------------------------------------------
    def run(self, scr) -> None:
        # curses waits ESCDELAY (1000ms by default) after a bare ESC to see
        # whether it begins an escape sequence, which makes cancelling feel
        # broken. 25ms is still long enough to assemble a real arrow key.
        curses.set_escdelay(25)
        # Every capability below is optional. TERM=vt100/vt220/ansi/dumb have
        # no cursor-visibility or default-colour support, and an unguarded call
        # aborts the whole tool with a raw traceback — the opposite of put()'s
        # promise that a limited terminal degrades rather than crashes.
        _try(curses.curs_set, 0)
        if _try(curses.use_default_colors) is not None:
            for i, c in ((1, curses.COLOR_RED), (2, curses.COLOR_YELLOW),
                         (3, curses.COLOR_GREEN), (4, curses.COLOR_BLUE)):
                _try(curses.init_pair, i, c, -1)
        while True:
            self.draw(scr)
            try:
                key = scr.get_wch()
            except curses.error:
                continue
            if key in ("q", "Q"):
                return
            if key in (curses.KEY_UP, "k"):
                self.sel = max(0, self.sel - 1)
            elif key in (curses.KEY_DOWN, "j"):
                self.sel = min(len(self.vars) - 1, self.sel + 1)
            elif key == curses.KEY_NPAGE:
                self.sel = min(len(self.vars) - 1, self.sel + 10)
            elif key == curses.KEY_PPAGE:
                self.sel = max(0, self.sel - 10)
            elif key == "/":
                self.filtered = not self.filtered
                self.all = cat.build(self.root)
                self.vars = ([v for v in self.all if v.status != "set"]
                             if self.filtered else list(self.all))
                if not self.vars:                      # nothing outstanding
                    self.vars, self.filtered = list(self.all), False
                    self.msg = "nothing outstanding — showing everything"
                self.sel = self.top = 0
            elif key in ("\n", "\r", curses.KEY_ENTER):
                if not self.vars:
                    continue
                v = self.vars[self.sel]
                got = self.prompt(scr, v)
                if got:
                    self.apply(v, got)
                else:
                    self.msg = "cancelled"
            elif key in ("c", "C"):
                if not self.vars:
                    continue
                v = self.vars[self.sel]
                if v.value:
                    self.apply(v, None)


def render_rows(root: Path) -> list[str]:
    """The same rows the TUI shows, as plain text. The testable surface."""
    vars_ = cat.build(root)
    rows = [cat.summary(vars_)]
    for v in vars_:
        rows.append(f"{MARK[v.status]} {v.status:<11} {v.name:<29} "
                    f"{v.display()[:26]:<28} "
                    f"{'required' if v.required else 'optional':<9} "
                    f"{','.join(v.components) or 'server'}")
    return rows


def main() -> int:
    # The skeletons always come from the repo; which .env is read and written
    # is decided by env_catalog.env_path, which honours INSIDEOUT_ENV_FILE —
    # exactly as env.sh overrides ENV_FILE but not EXAMPLE_FILE.
    root = Path(__file__).resolve().parent.parent
    if "--list" in sys.argv:
        print("\n".join(render_rows(root)))
        return 0
    if not sys.stdin.isatty() or not sys.stdout.isatty():
        print("env.sh edit needs a terminal; use `./scripts/env.sh edit --list` "
              "for a non-interactive view.", file=sys.stderr)
        return 2
    # TERM=dumb (and an unset TERM) have no cursor addressing at all, so curses
    # cannot start — it hangs or aborts rather than returning. Refuse up front
    # and name the working alternative, instead of leaving a dead terminal.
    if os.environ.get("TERM", "") in ("", "dumb"):
        print(f"env.sh edit cannot drive TERM={os.environ.get('TERM', '')!r}; "
              "use `./scripts/env.sh edit --list`.", file=sys.stderr)
        return 2
    ed = Editor(root)
    try:
        curses.wrapper(ed.run)
    except curses.error as exc:
        # Any remaining terminal-capability failure degrades to the list view
        # rather than a traceback — same promise put() makes about narrow ones.
        print(f"env.sh edit: this terminal could not run the curses UI ({exc}); "
              "use `./scripts/env.sh edit --list`.", file=sys.stderr)
        return 2
    if ed.dirty:
        print("updated in .env: " + ", ".join(sorted(ed.dirty)))
        # The generated component copies are now stale by checksum, and dev.sh
        # refuses to launch on stale config — so offer the re-propagate rather
        # than letting the next run fail. On EOF (Ctrl-D, or a piped stdin)
        # default to yes: the write already happened, so declining silently
        # would leave the copies stale and the next `dev.sh -C app` refusing,
        # with a traceback as the only clue.
        try:
            ans = input("re-run `env.sh propagate` so the components see it? [Y/n]: ")
        except EOFError:
            ans = ""
            print("(no input — running propagate)")
        if ans.strip().lower() in ("", "y", "yes"):
            subprocess.run([str(root / "scripts" / "env.sh"), "propagate"], cwd=root)
        else:
            print("skipped — run `./scripts/env.sh propagate` before the next "
                  "`dev.sh -C app`, which will otherwise refuse on a stale copy.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
