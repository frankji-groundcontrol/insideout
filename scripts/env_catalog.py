"""The env catalog: every variable InsideOut knows about, and its current state.

Split from the TUI on purpose. curses needs a tty, so anything that cannot be
exercised headlessly cannot be tested — this module holds all of the decisions
and `env_tui.py` is a thin rendering shell over it. `env.sh edit --list` prints
the same catalog the TUI shows, which is what test_env_catalog.py (read path)
and test_env_writes.py (write path) assert on.

Where the facts come from:

  .env.example (root + each component's)  -> WHICH vars exist, and for whom
  an UNCOMMENTED line (KEY=)             -> REQUIRED
  a COMMENTED line   (#KEY=)             -> OPTIONAL, default in the comment
  .env (root)                            -> what is actually set

Reading the skeleton is the point: a second hardcoded list of variable names
drifts. `scripts/env.sh` used to carry one, and it already disagreed with the
skeleton about two variables (GITHUB_TOKEN, NUXT_API_INTERNAL_BASE).

Value *validity* is deliberately NOT decided here — the 32-character JWT floor,
Go duration syntax and DSN shape all live in `env.sh check`, so there is one
authority for "usable" and one for "supplied".
"""

from __future__ import annotations

import os
import re
from dataclasses import dataclass, field
from pathlib import Path

#: component -> directory holding its .env.example, mirroring env-lib.sh's
#: component_dir(). `server` is absent on purpose: the Go binary reads process
#: env via os.Getenv and has no dotenv dependency, so a server/.env would be a
#: file nothing loads.
COMPONENT_DIRS = {"app": "app"}

# A value is masked when its NAME says it is a credential. Name-based because
# the decision must be made BEFORE the value reaches a widget — inspecting it
# to decide would mean it had already been read onto the screen.
SECRET_RX = re.compile(r"(KEY|SECRET|PASSWORD|PASSWD|TOKEN|CREDENTIAL)", re.I)
#: Not caught by the rule above. DATABASE_URL embeds a password;
#: ANTHROPIC_BASE_URL points at a provider (and gateway URLs routinely carry a
#: key in the path), which repo policy treats as sensitive alongside secrets.
SECRET_NAMES = {"DATABASE_URL", "ANTHROPIC_BASE_URL"}

#: Markers meaning "copied from the skeleton and never filled in". Matched
#: lowercased. Kept in sync with env-lib.sh's is_placeholder — the two must
#: agree about what counts as not-really-set.
#: NOTE: no generic '<' test. A bare-angle heuristic is how Piper's checker
#: started rejecting legitimate values; only these two literals are markers.
PLACEHOLDERS = ("change_me", "your-remote-host")

KEY_RX = re.compile(r"^(#?)\s*([A-Za-z_][A-Za-z0-9_]*)=(.*)$")


def env_path(root: Path) -> Path:
    """Which .env to read and write.

    Honours INSIDEOUT_ENV_FILE, the testing knob env.sh documents, so `edit`
    — the one verb that WRITES — cannot silently target the repo's real file
    while every other verb is pointed at a scratch one. The SKELETONS are
    never overridden: the schema belongs to the repo, exactly as env.sh
    overrides ENV_FILE but not EXAMPLE_FILE.
    """
    override = os.environ.get("INSIDEOUT_ENV_FILE")
    return Path(override).expanduser() if override else root / ".env"


def strip_inline_comment(value: str) -> str:
    """Drop a trailing ` # …` comment from an unquoted dotenv value.

    Every consumer of these files already does this — bash `set -a; . .env`
    treats it as a comment token, and so do compose's and c12's parsers — so a
    catalog that kept it would report a spurious mismatch against the skeleton.
    Whitespace before the `#` is required, which leaves `p@ss#word` intact.
    """
    if value[:1] in ("'", '"'):
        return value
    return re.split(r"\s+#", value, maxsplit=1)[0].rstrip()


@dataclass
class Var:
    name: str
    required: bool = False
    components: list[str] = field(default_factory=list)
    example: str = ""          # the placeholder/default from the skeleton
    #: EVERY example value seen for this name. The root skeleton and a
    #: component's may differ, and matching any of them still means the value
    #: came from a skeleton rather than from a person.
    examples: set[str] = field(default_factory=set)
    comment: str = ""          # the comment block above it, for the help pane
    value: str | None = None   # from the root .env; None/"" = not set

    @property
    def secret(self) -> bool:
        return self.name in SECRET_NAMES or bool(SECRET_RX.search(self.name))

    @property
    def is_set(self) -> bool:
        return bool(self.value) and not self.placeholder

    @property
    def placeholder(self) -> bool:
        v = (self.value or "").lower()
        return any(m in v for m in PLACEHOLDERS)

    @property
    def same_as_example(self) -> bool:
        """The value is byte-identical to what a .env.example ships.

        Meaning it came from the skeleton, not from you. That is reported as
        `default` and is never an error: `15m` and `claude-sonnet-4-20250514`
        ARE the intended values. The things only you can supply — the DSN, the
        JWT secret — ship as `change_me…` placeholders, so they surface as
        placeholder/missing and can never masquerade as a default. An empty
        example does not count; an empty value is already unset.
        """
        return bool(self.value) and self.value in {e for e in self.examples if e}

    @property
    def status(self) -> str:
        if self.placeholder:
            return "placeholder"
        if self.same_as_example:
            return "default"
        if self.is_set:
            return "set"
        return "missing" if self.required else "unset"

    def display(self) -> str:
        """What to show in the list — never a raw secret.

        The NAME test comes first and is absolute. An earlier version returned
        the raw value whenever the status was `placeholder` or `default`, on
        the reasoning that a skeleton value is public by construction — but
        `placeholder` is a *substring* test on the LIVE value, so a real
        credential merely containing `change_me` was rendered in the clear.
        A content test can never be allowed to override a name test.

        `default` is still shown for non-secrets; for secrets the state mark
        (`=`) already says it came from the skeleton without printing it.
        """
        if not self.value:
            return ""
        if self.secret:
            return "•" * 6
        return self.value


def quote(value: str) -> str:
    """Render a value so `set -a; . .env` and every dotenv parser read it back.

    A bare `KEY=two words` makes bash treat the second word as a command, so
    the tool would write a file its own reader rejects — and a space is legal
    in a Postgres password, which both the TUI prompt and `init` accept.
    Single quotes suppress bash interpolation entirely; a value containing one
    falls back to double quotes with the four bash-active characters escaped.
    parse_env_file strips the paired quotes back off, as do compose and c12.
    """
    if value == "" or re.fullmatch(r"[A-Za-z0-9_./:@%+,=~^-]*", value):
        return value                      # safe bare, and stays diff-friendly
    if "'" not in value:
        return f"'{value}'"
    return '"' + re.sub(r'([$`"\\])', r"\\\1", value) + '"'


def parse_env_file(path: Path) -> dict[str, str]:
    """KEY=value pairs from a dotenv. Ignores comments; strips paired quotes."""
    out: dict[str, str] = {}
    try:
        if not path.is_file():
            return out
        text = path.read_text(errors="replace")
    except OSError:
        # An unreadable .env (root-owned after a sudo'd or in-container write)
        # must degrade to "nothing is set", not a traceback out of `env.sh`.
        return out
    for line in text.splitlines():
        if not line.strip() or line.lstrip().startswith("#"):
            continue
        if "=" not in line:
            continue
        k, _, v = line.partition("=")
        k = k.strip().removeprefix("export ").strip()
        if not re.fullmatch(r"[A-Za-z_][A-Za-z0-9_]*", k):
            continue
        v = v.strip()
        if len(v) >= 2 and v[0] == v[-1] and v[0] in "\"'":
            v = v[1:-1]
        else:
            v = strip_inline_comment(v)
        out[k] = v
    return out


def parse_skeleton(path: Path) -> list[tuple[str, bool, str, str]]:
    """(name, required, example, comment) from a .env.example.

    A commented `#KEY=` line is an OPTIONAL variable, not documentation — that
    distinction is what the whole catalog rests on. Free-text comment lines
    accumulate and attach to the next key as its help text.
    """
    out: list[tuple[str, bool, str, str]] = []
    if not path.is_file():
        return out
    buf: list[str] = []
    for line in path.read_text(errors="replace").splitlines():
        stripped = line.strip()
        if not stripped:
            buf.clear()
            continue
        m = KEY_RX.match(stripped)
        if m:
            hashed, name, example = m.groups()
            out.append((name, not hashed, strip_inline_comment(example.strip()),
                        " ".join(buf)))
            buf.clear()
        elif stripped.startswith("#"):
            buf.append(stripped.lstrip("# ").rstrip())
    return out


def build(root: Path) -> list[Var]:
    """Every known var, ordered by how much attention it needs.

    Sorting puts what blocks a launch at the top, which is the whole reason to
    open this screen.
    """
    by_name: dict[str, Var] = {}

    def absorb(path: Path, component: str) -> None:
        for name, required, example, comment in parse_skeleton(path):
            v = by_name.setdefault(name, Var(name))
            # Required anywhere => required. A component that must have it wins
            # over one for which it is optional.
            v.required = v.required or required
            if component and component not in v.components:
                v.components.append(component)
            v.example = v.example or example
            if example:
                v.examples.add(example)
            v.comment = v.comment or comment

    absorb(root / ".env.example", "")
    for comp, d in COMPONENT_DIRS.items():
        absorb(root / d / ".env.example", comp)

    for name, val in parse_env_file(env_path(root)).items():
        by_name.setdefault(name, Var(name)).value = val

    order = {"missing": 0, "placeholder": 1, "unset": 2, "default": 3, "set": 4}
    return sorted(by_name.values(), key=lambda v: (order[v.status], v.name))


def _write(path: Path, lines: list[str]) -> None:
    tmp = path.with_name(path.name + ".tmp")
    tmp.write_text("\n".join(lines) + "\n")
    # chmod BEFORE the rename, not after: doing it after leaves a window in
    # which .env exists at the umask default. The rename is atomic, so the file
    # is never observable with the wrong mode. (.env.tmp is gitignored by the
    # root .gitignore's `.env.*` rule.)
    tmp.chmod(0o600)
    os.replace(tmp, path)


def _last_match(lines: list[str], name: str) -> int:
    """Index of the LAST line assigning `name` (commented or not), else -1.

    Last, not first. Every reader of a dotenv resolves a duplicated key to the
    last occurrence — bash `set -a; . .env` executes the assignments in order,
    and compose and c12 do the same — and appending an override at the bottom
    of the file is ordinary practice. Editing the first occurrence would write
    a value that nothing then uses, while reporting it saved.
    """
    rx = re.compile(rf"^#?\s*{re.escape(name)}=")
    live = [i for i, ln in enumerate(lines) if rx.match(ln) and not ln.lstrip().startswith("#")]
    if live:
        return live[-1]
    hits = [i for i, ln in enumerate(lines) if rx.match(ln)]
    return hits[-1] if hits else -1


def set_var(root: Path, name: str, value: str) -> None:
    """Write one KEY into the root .env, preserving everything else.

    Replaces the effective line in place rather than appending, so the file
    keeps the skeleton's grouping and comments. Substitution rebuilds the line
    list; it is never `sed s///`, because a DSN is full of `/`, `:` and `@`.
    """
    path = env_path(root)
    lines = path.read_text().splitlines() if path.is_file() else []
    i = _last_match(lines, name)
    if i < 0:
        lines.append(f"{name}={quote(value)}")
    else:
        lines[i] = f"{name}={quote(value)}"
    _write(path, lines)


def clear_var(root: Path, name: str) -> None:
    """Comment the key out, so nothing is set and the line keeps its place.

    Deleting the line would lose the variable's position in the file; writing
    an empty `KEY=` would propagate an explicit empty override to a component.
    `#KEY=` round-trips exactly back through set_var. Note this does not
    restore the skeleton's default text into the comment — `.env.example`
    remains where that is recorded, and it is what the catalog reads.
    """
    path = env_path(root)
    if not path.is_file():
        return
    lines = path.read_text().splitlines()
    i = _last_match(lines, name)
    if i < 0:
        return
    lines[i] = f"#{name}="
    _write(path, lines)


def summary(vars_: list[Var]) -> str:
    n = {k: 0 for k in ("set", "missing", "placeholder", "unset", "default")}
    for v in vars_:
        n[v.status] += 1
    return (f"{n['set']} yours · {n['default']} default · "
            f"{n['missing']} missing · {n['placeholder']} placeholder · "
            f"{n['unset']} unset")
