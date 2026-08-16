#!/usr/bin/env python3
"""Reading the environment — run: python3 scripts/test_env_catalog.py

`env.sh edit` needs a tty and cannot be exercised headlessly, so every decision
it makes lives in env_catalog and is asserted here instead. This file covers the
READ path: the schema convention, the five states, masking, attribution and
parsing — plus the linters that hold THIS repo's own skeletons honest. The write
path is scripts/test_env_writes.py; the shared harness is env_testlib.py.

These cover the LOGIC. They cannot see a curses crash: the rendering layer is
driven separately in tmux (`tmux capture-pane -p` returns the rendered cell
grid; a raw pty returns the escape stream and reports phantom misalignment).
See docs/practices/2026-07-30-verifying-a-tty-only-tool.md.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import env_catalog as cat  # noqa: E402
from env_testlib import REPO, check, run, scaffold  # noqa: E402

import tempfile  # noqa: E402


# ── the schema convention · 形式即约定 ───────────────────────────────────────

def t_required_vs_optional():
    with tempfile.TemporaryDirectory() as d:
        root = scaffold(Path(d), "REQ=\n#OPT=default\n")
        v = {x.name: x for x in cat.build(root)}
        check("uncommented => required", v["REQ"].required is True)
        check("commented   => optional", v["OPT"].required is False)
        check("required+unset => missing", v["REQ"].status == "missing")
        check("optional+unset => unset", v["OPT"].status == "unset")


def t_value_equal_to_the_example_is_a_default_not_a_choice():
    with tempfile.TemporaryDirectory() as d:
        root = scaffold(Path(d), "#AI_MODEL=claude-sonnet-4-20250514\n#X=\n",
                        "AI_MODEL=claude-sonnet-4-20250514\nX=mine\n")
        v = {x.name: x for x in cat.build(root)}
        check("copied from the skeleton => default", v["AI_MODEL"].status == "default")
        check("a value you supplied     => set", v["X"].status == "set")


def t_default_is_not_an_error():
    # `15m` IS the intended access-token lifetime; flagging it would cry wolf.
    with tempfile.TemporaryDirectory() as d:
        root = scaffold(Path(d), "INSIDEOUT_ACCESS_TTL=15m\n",
                        "INSIDEOUT_ACCESS_TTL=15m\n")
        v = {x.name: x for x in cat.build(root)}["INSIDEOUT_ACCESS_TTL"]
        check("required + on default is not 'missing'", v.status == "default")
        check("and counts as having a value", v.is_set is True)


def t_credentials_ship_as_placeholders_so_they_cannot_pass_as_defaults():
    with tempfile.TemporaryDirectory() as d:
        root = scaffold(
            Path(d),
            "DATABASE_URL=postgres://insideout_app:change_me@your-remote-host:5432/x\n",
            "DATABASE_URL=postgres://insideout_app:change_me@your-remote-host:5432/x\n")
        v = {x.name: x for x in cat.build(root)}["DATABASE_URL"]
        check("skeleton credential reads as placeholder, not default",
              v.status == "placeholder")
        check("placeholder is not 'set'", v.is_set is False)


def t_placeholder_markers_are_case_insensitive_and_specific():
    with tempfile.TemporaryDirectory() as d:
        root = scaffold(Path(d), "A=\nB=\nC=\n",
                        "A=CHANGE_ME_upper\n"
                        "B=postgres://u:p@your-remote-host:5432/x\n"
                        "C=postgres://u:p@10.0.0.1:5432/x\n")
        v = {x.name: x for x in cat.build(root)}
        check("CHANGE_ME (any case) is a placeholder", v["A"].status == "placeholder")
        check("your-remote-host is a placeholder", v["B"].status == "placeholder")
        check("a real host is NOT a placeholder", v["C"].status == "set")


def t_angle_brackets_are_not_a_placeholder_heuristic():
    # Regression guard: a generic '<' test is how Piper's checker began
    # rejecting legitimate values. No variable here should inherit that bug.
    with tempfile.TemporaryDirectory() as d:
        root = scaffold(Path(d), "#X=\n", "X=a<b\n")
        check("'<' alone does not mean placeholder",
              {v.name: v for v in cat.build(root)}["X"].status == "set")


# ── masking · 脱敏 ───────────────────────────────────────────────────────────

def t_secrets_are_masked_by_name():
    with tempfile.TemporaryDirectory() as d:
        root = scaffold(Path(d),
                        "DATABASE_URL=\nINSIDEOUT_JWT_SECRET=\n#GITHUB_TOKEN=\n"
                        "#ANTHROPIC_AUTH_TOKEN=\n#POSTGRES_APP_PASSWORD=\n#AI_MODEL=\n",
                        "DATABASE_URL=postgres://u:hunter2@10.0.0.1/db\n"
                        "INSIDEOUT_JWT_SECRET=deadbeef" + "0" * 40 + "\n"
                        "GITHUB_TOKEN=ghp_realtoken123\n"
                        "ANTHROPIC_AUTH_TOKEN=sk-ant-real\n"
                        "POSTGRES_APP_PASSWORD=pw123\n"
                        "AI_MODEL=claude-opus-5\n")
        v = {x.name: x for x in cat.build(root)}
        for name in ("DATABASE_URL", "INSIDEOUT_JWT_SECRET", "GITHUB_TOKEN",
                     "ANTHROPIC_AUTH_TOKEN", "POSTGRES_APP_PASSWORD"):
            check(f"{name} is masked", v[name].display() == "•" * 6)
        check("DATABASE_URL password never rendered",
              "hunter2" not in v["DATABASE_URL"].display())
        check("GITHUB_TOKEN value never rendered",
              "ghp_realtoken123" not in v["GITHUB_TOKEN"].display())
        check("a non-secret is shown plainly", v["AI_MODEL"].display() == "claude-opus-5")


def t_no_repo_variable_is_a_secret_by_accident():
    # The masking rule is a NAME regex; assert it does not sweep in the
    # non-secrets, or the TUI becomes useless for the values worth reading.
    # ANTHROPIC_BASE_URL is deliberately absent: it is classified as a secret
    # (see t_provider_endpoint_is_treated_as_sensitive).
    plain = ("AI_MODEL", "SERVER_PORT", "APP_PORT", "POSTGRES_PORT",
             "INSIDEOUT_ADDR", "INSIDEOUT_ACCESS_TTL", "INSIDEOUT_DEV_CORS",
             "NUXT_API_INTERNAL_BASE")
    for name in plain:
        check(f"{name} is not treated as a secret", cat.Var(name).secret is False)


# ── attribution · 归属 ───────────────────────────────────────────────────────

def t_component_attribution():
    with tempfile.TemporaryDirectory() as d:
        root = scaffold(Path(d), "DATABASE_URL=\n", "",
                        {"app": "#NUXT_API_INTERNAL_BASE=\n"})
        v = {x.name: x for x in cat.build(root)}
        check("app var attributed to app",
              v["NUXT_API_INTERNAL_BASE"].components == ["app"])
        check("a root-only var has no component", v["DATABASE_URL"].components == [])


def t_required_anywhere_wins():
    with tempfile.TemporaryDirectory() as d:
        root = scaffold(Path(d), "#SHARED=\n", "", {"app": "SHARED=x\n"})
        check("required in a component => required overall",
              {v.name: v for v in cat.build(root)}["SHARED"].required is True)


def t_missing_sorts_first():
    with tempfile.TemporaryDirectory() as d:
        root = scaffold(Path(d), "ZZZ_REQUIRED=\n#AAA_OPTIONAL=\n")
        names = [v.name for v in cat.build(root)]
        check("required-missing sorts above optional",
              names.index("ZZZ_REQUIRED") < names.index("AAA_OPTIONAL"))


# ── parsing · 解析 ───────────────────────────────────────────────────────────

def t_inline_comments_are_stripped_like_bash_does():
    # `set -a; . .env` treats ` # …` as a comment, and so do compose and c12.
    # A catalog that kept it would report a spurious mismatch vs the skeleton.
    with tempfile.TemporaryDirectory() as d:
        root = scaffold(Path(d), "#PORT=5442\n#PW=\n",
                        "PORT=5442   # host side\nPW=p@ss#word\n")
        v = {x.name: x for x in cat.build(root)}
        check("' #' comment stripped => matches the example", v["PORT"].status == "default")
        check("'#' with no leading space is kept as value", v["PW"].value == "p@ss#word")


# ── writing · 写入 ───────────────────────────────────────────────────────────

# ── linting THIS repo's skeletons · 校验本仓库骨架 ───────────────────────────

def t_repo_skeleton_marks_only_the_fail_fast_vars_required():
    """The form is machine-read, so a contradiction is a bug, not a typo.

    Exactly the two variables `config.Load` refuses to boot without may be
    uncommented. Uncommenting anything else makes `env.sh check` report a
    non-problem — and a checker that cries wolf gets ignored.
    """
    got = {n for n, req, _, _ in cat.parse_skeleton(REPO / ".env.example") if req}
    want = {"DATABASE_URL", "INSIDEOUT_JWT_SECRET"}
    check(f"root skeleton required set == {sorted(want)} (got {sorted(got)})", got == want)


def t_repo_app_contract_declares_only_what_nuxt_reads():
    declared = {n for n, _, _, _ in cat.parse_skeleton(REPO / "app" / ".env.example")}
    check("app contract == {NUXT_API_INTERNAL_BASE}",
          declared == {"NUXT_API_INTERNAL_BASE"})
    check("and it is optional (nuxt.config.ts has a working default)",
          all(not req for _, req, _, _ in
              cat.parse_skeleton(REPO / "app" / ".env.example")))


def t_repo_skeleton_declares_nothing_dead():
    """Every declared variable must be read by a real consumer.

    Two bug classes at once. It catches a variable that outlived its consumer,
    and — the reason it exists — a phantom declaration produced by prose: a
    `NAME` followed by `=` written inside a comment parses as a declaration
    like any other, so an explanatory line can silently invent a variable.
    That happened while writing these very skeletons.
    """
    consumers = sorted(
        [p for p in (REPO / "server").rglob("*.go") if "_test.go" not in p.name]
        + [REPO / "app" / "nuxt.config.ts", REPO / "docker-compose.yml"])
    haystack = "\n".join(p.read_text(errors="replace")
                         for p in consumers if p.is_file())
    # Match the CONSUMPTION syntax, not the bare token. A plain word-boundary
    # search passed a deliberately-planted phantom `KEY`, because Go's SQL says
    # `FOR KEY SHARE` — a guard that cannot fail in the direction of the bug is
    # not a guard. Go names the var as a complete string literal (os.Getenv or
    # the getenv/parseDuration wrappers), Nuxt as process.env.NAME, compose as
    # ${NAME} / ${NAME:-…} / ${NAME:?…}.
    def read_somewhere(name: str) -> bool:
        n = re.escape(name)
        return bool(re.search(rf'"{n}"|process\.env\.{n}\b|\$\{{{n}[:}}]', haystack))

    declared = {n for f in (REPO / ".env.example", REPO / "app" / ".env.example")
                for n, _, _, _ in cat.parse_skeleton(f)}
    dead = sorted(n for n in declared if not read_somewhere(n))
    check(f"every declared variable has a consumer (orphans: {dead})", not dead)


def t_repo_skeleton_has_no_inline_comment_after_a_value():
    """An inline `# …` makes the example value un-comparable to a live one.

    Everything downstream strips it, so a skeleton that carries one silently
    breaks the set-vs-default signal for that variable.
    """
    bad = [ln for ln in (REPO / ".env.example").read_text().splitlines()
           if cat.KEY_RX.match(ln.strip()) and "#" in ln.split("=", 1)[-1]]
    check(f"no inline comment after a value (got {bad})", not bad)


# ── regressions from the 2026-07-30 adversarial review · 对抗性复核回归 ──────

def t_a_name_test_always_beats_a_content_test():
    """A secret must stay masked even when its status says `placeholder`.

    `placeholder` is a SUBSTRING test on the live value, so an earlier
    "a skeleton value is public by construction" shortcut rendered a real
    credential in the clear as soon as it happened to contain `change_me`.
    """
    with tempfile.TemporaryDirectory() as d:
        root = scaffold(Path(d), "DATABASE_URL=\n#GITHUB_TOKEN=\n#AI_MODEL=x\n",
                        "DATABASE_URL=postgres://u:change_me_but_REAL@h/db\n"
                        "GITHUB_TOKEN=ghp_change_me_yet_REAL\n"
                        "AI_MODEL=x\n")
        v = {x.name: x for x in cat.build(root)}
        check("placeholder DATABASE_URL still masked", v["DATABASE_URL"].display() == "•" * 6)
        check("placeholder GITHUB_TOKEN still masked", v["GITHUB_TOKEN"].display() == "•" * 6)
        check("no real material rendered",
              not any("REAL" in x.display() for x in v.values()))
        check("a non-secret default is still shown", v["AI_MODEL"].display() == "x")


def t_provider_endpoint_is_treated_as_sensitive():
    # A gateway base URL routinely carries the API key in its path, and repo
    # policy treats a provider identifier as sensitive regardless.
    check("ANTHROPIC_BASE_URL is a secret", cat.Var("ANTHROPIC_BASE_URL").secret is True)

raise SystemExit(run(globals()))
