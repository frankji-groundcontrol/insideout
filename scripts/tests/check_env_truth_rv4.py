#!/usr/bin/env python3
"""Fail when env-facing text says more than the code does (review-4 wave 4).

Usage: python3 -I scripts/tests/check_env_truth_rv4.py

Same method as check_docs_truth_rv4.py: derive each premise from the tree
(which code reads GITHUB_TOKEN, which variables the server process reads,
which variables docker-compose.yml interpolates), then assert the text agrees,
so a sentence cannot outlive the code it describes. A demand disappears with
its premise. Exits 1 naming every offending claim and 2 when a premise cannot
be derived (rv4-FEAT-04 residue R3-27, rv4-FULL-09 residue R3-35).
"""
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parents[2]
# Host-side verbs: they run on the operator's machine, never in the server container.
SETUP_VERBS = re.compile(r"^cli_setup\w*\.go$")
NAME = r"\b[A-Z][A-Z0-9]*_[A-Z0-9_]+\b"  # an environment variable name
TOKEN_CLAIM = re.compile(r"(?i)unauthenticated|rate[- ]?limit|\bGo server\b")
COMMENT = re.compile(r"\s*(?:#|//)\s?(.*)")  # shell, YAML, Python, .env and Go comment lines
# "docker-compose.yml does not interpolate X": X (or "it", the row's variable) is the object.
NOT_INTERPOLATED = re.compile(
    r"(?i:compose(?:\.ya?ml)?[`'\"*]*\s+(?:does\s+not|doesn't|do\s+not|don't)\s+(?:yet\s+)?"
    r"(?:interpolate|pass|forward|carry))\s+"
    rf"(?P<object>`?{NAME}`?(?:\s*(?:,|/|and|or)\s*`?{NAME}`?)*|(?i:it|them)\b)")
# Living docs a reader follows; dated records (changelogs, plans, review) are history.
LIVING_FILES = ("README.md", "PRODUCT.md", "docs/SETENV.md", ".env.example")
LIVING_TREES = ("docs/usage", "docs/architecture")


def read(name):
    return (ROOT / name).read_text(encoding="utf-8")


def rel(path):
    return path.relative_to(ROOT).as_posix()


def fail(failures, message):
    failures.append(message)


def cannot_derive(message):
    print(message, file=sys.stderr)
    sys.exit(2)


def go_sources(*trees):
    """Non-test Go files under the given repo-relative trees."""
    return [p for tree in trees for p in sorted((ROOT / tree).rglob("*.go"))
            if not p.name.endswith("_test.go")]


def numbered_lines(path):
    return enumerate(path.read_text(encoding="utf-8").splitlines(), 1)


def compose_service(name):
    """Text of one service of docker-compose.yml. The file is a flat services
    map, so a cut at the next two-space key needs no YAML parser."""
    match = re.search(rf"^  {re.escape(name)}:\n(.*?)(?=^  [A-Za-z_][\w-]*:|\Z)",
                      read("docker-compose.yml"), re.M | re.S)
    if not match:
        cannot_derive(f"no `{name}` service in docker-compose.yml")
    return match.group(1)


def server_env_names():
    """Names the server service passes to the container."""
    block = re.search(r"^    environment:\n((?:      .*\n|[ \t]*\n)*)", compose_service("server"), re.M)
    names = re.findall(r"^      (?:-\s+)?([A-Z][A-Z0-9_]*)\s*[:=]", block.group(1), re.M) if block else []
    if not names:
        cannot_derive("no environment entries could be read from the docker-compose.yml server service")
    return names


def server_reads_github_token():
    """True while server code outside the host-side setup verbs reads GITHUB_TOKEN."""
    return any('"GITHUB_TOKEN"' in p.read_text(encoding="utf-8")
               for p in go_sources("server") if not SETUP_VERBS.match(p.name))


def statements(path):
    """(first line number, text) pairs. A run of consecutive comment lines is one
    text, so a sentence wrapped over several lines stays whole; any other line
    stands alone."""
    run = []
    for number, line in [*numbered_lines(path), (0, "")]:  # the sentinel closes a final run
        comment = COMMENT.match(line)
        if comment:
            run.append((number, comment.group(1)))
            continue
        if run:
            yield run[0][0], " ".join(text for _, text in run)
            run = []
        if line.strip():
            yield number, line


def token_sentences(text):
    """The sentences of one text that speak about GITHUB_TOKEN: those naming it,
    or every sentence of a comment block that ends on its assignment (how
    .env.example describes a variable: prose above `#GITHUB_TOKEN=`)."""
    owned = re.search(r"\bGITHUB_TOKEN=\s*$", text)
    return [s for s in re.split(r"(?<=\.)\s+", text) if owned or "GITHUB_TOKEN" in s]


def token_not_oversold(failures):
    """rv4-FEAT-04 (R3-27): commit sync reads a private repo with the GitHub
    App's installation token and a public one anonymously, so the server never
    reads GITHUB_TOKEN; only `insideout setup --check` does. While that holds,
    no env script, compose file, env example or CLI source may tie the token to
    sync (a rate limit, an unauthenticated run) or to the Go server, however
    the sentence is wrapped, and the prompt that asks for it must say who
    reads it."""
    if server_reads_github_token():
        return  # the server reads it again; the wording may change
    scripts = sorted((ROOT / "scripts").glob("*.sh"))
    texts = scripts + [p for p in sorted((ROOT / "scripts").glob("*.py")) if not p.name.startswith("test_")]
    texts += [ROOT / "docker-compose.yml", ROOT / ".env.example"] + go_sources("server/cmd")
    for path in texts:
        for number, text in statements(path):
            for sentence in token_sentences(text):
                if TOKEN_CLAIM.search(sentence):
                    fail(failures, f"{rel(path)}:{number} ties GITHUB_TOKEN to sync or to the Go server "
                                   f"({sentence.strip()[:100]!r}), but only `insideout setup --check` reads it")
    for path in scripts:
        for number, line in numbered_lines(path):
            if re.match(r"\s*printf\b", line) and "GITHUB_TOKEN" in line and "setup --check" not in line:
                fail(failures, f"{rel(path)}:{number} asks for GITHUB_TOKEN without saying that only "
                               "`insideout setup --check` reads it")


def compose_carries_what_the_server_reads(facts, failures):
    """rv4-FULL-09 (R3-35): every INSIDEOUT_* variable the server process reads
    (server/internal/config and server/internal/agent, the definition
    check_docs_truth_rv4.py uses) is passed by the compose server service;
    otherwise setting it in .env does nothing under compose."""
    read_at = {}
    for path in go_sources("server/internal/config", "server/internal/agent"):
        for name in re.findall(r'"(INSIDEOUT_[A-Z0-9_]+)"', path.read_text(encoding="utf-8")):
            read_at.setdefault(name, rel(path))
    if not read_at:
        cannot_derive("no INSIDEOUT_* variable is read under server/internal/config or server/internal/agent")
    facts["server-read variables"] = len(read_at)
    passed = set(server_env_names())
    for name, where in sorted(read_at.items()):
        if name not in passed:
            fail(failures, f"docker-compose.yml: the server service does not pass {name} (read at {where}), "
                           "so setting it in .env has no effect under compose")


def compose_passes_only_what_the_server_reads(failures):
    """rv4-FEAT-04 (R3-27): a variable the server service passes must be read by
    some server code, the host-side setup verbs excluded because they never run
    in the container (the image's only other command is `health`). GITHUB_TOKEN
    was passed through while nothing in the server read it."""
    literals = "\n".join(p.read_text(encoding="utf-8")
                         for p in go_sources("server") if not SETUP_VERBS.match(p.name))
    for name in server_env_names():
        if f'"{name}"' not in literals:
            fail(failures, f"docker-compose.yml: the server service passes {name}, but no server code "
                           "reads it (the host-side `insideout setup --check` does not run in the container)")


def living_docs():
    found = [ROOT / name for name in LIVING_FILES]
    for tree in LIVING_TREES:
        found += sorted((ROOT / tree).rglob("*.md"))
    return found


def units(text):
    """Each table row alone; every other paragraph joined onto one line."""
    for block in re.split(r"\n\s*\n", text):
        prose = []
        for line in block.splitlines():
            if line.lstrip().startswith("|"):
                yield line
            else:
                prose.append(line.strip().lstrip("#").strip())  # `#` of a wrapped comment
        if prose:
            yield " ".join(prose)


def docs_do_not_deny_what_compose_does(facts, failures):
    """rv4-FULL-09 (R3-35): no living doc may say compose does not interpolate a
    variable that docker-compose.yml interpolates. In a table row "it" is the
    variable of the first cell ("... does not interpolate it yet")."""
    interpolated = set(re.findall(r"\$\{([A-Z][A-Z0-9_]*)", read("docker-compose.yml")))
    if not interpolated:
        cannot_derive("docker-compose.yml interpolates no variable")
    facts["interpolated variables"] = len(interpolated)
    for path in living_docs():
        for unit in units(path.read_text(encoding="utf-8")):
            subject = set(re.findall(NAME, unit.split("|")[1])) if unit.lstrip().startswith("|") else set()
            for claim in NOT_INTERPOLATED.finditer(unit):
                for name in sorted((set(re.findall(NAME, claim.group("object"))) or subject) & interpolated):
                    fail(failures, f"{rel(path)} says docker-compose.yml does not interpolate {name}, but it does")


if __name__ == "__main__":
    facts, failures = {}, []
    token_not_oversold(failures)
    compose_carries_what_the_server_reads(facts, failures)
    compose_passes_only_what_the_server_reads(failures)
    docs_do_not_deny_what_compose_does(facts, failures)
    for message in failures:
        print(message, file=sys.stderr)
    if failures:
        sys.exit(1)
    print(f"env truth (rv4) ok: compose carries all {facts['server-read variables']} server-read "
          f"INSIDEOUT_* variables; {facts['interpolated variables']} interpolated variables checked against the docs")
