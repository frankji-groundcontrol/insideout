#!/usr/bin/env python3
"""Pin the first-run copy that lives in the install and setup scripts,
the CLI help and the two docs that still named both codes the same way
(review-4 wave 3: rv4-PM-08 code half, LEAD-SETUP-BLURB; wave 4:
rv4-USER-10, the hosted blurb half of rv4-PM-08).

Usage: python3 -I scripts/tests/check_first_run_copy.py

Stdlib only; no network, database or Go toolchain. The scripts are run, not
just read, wherever they stop before doing any work: scripts/install.sh stops
at its Go check when no go is on PATH, scripts/bootstrap.sh stops at its OS
check when uname names another system, and scripts/setup.sh --hosted prints
its blurb and exits before it looks at any env file. Each check derives its
premise from the tree first (the style of check_ops_hardening.py). Exits 1
naming every offending property; exit 2 when a premise cannot be derived.
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
    with a scratch HOME and an env file that does not exist. `{scratch}` in
    `args` is that directory. Returns (exit code, stdout, stderr)."""
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
        env = {"PATH": str(bindir), "HOME": str(scratch), "LC_ALL": "C",
               "INSIDEOUT_ENV_FILE": str(scratch / "absent.env")}
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


def paragraph(lines, at):
    """The blank-line-delimited block of `lines` that holds index `at`."""
    start = end = at
    while start > 0 and lines[start - 1].strip():
        start -= 1
    while end + 1 < len(lines) and lines[end + 1].strip():
        end += 1
    return "\n".join(lines[start:end + 1])


def line_at(lines, text):
    """Index of the first line that is `text` once stripped, else -1."""
    return next((i for i, line in enumerate(lines) if line.strip() == text), -1)


def hosted_blurb_teaches_login_then_whoami(failures):
    """LEAD-SETUP-BLURB: `setup.sh --hosted` taught
    `export INSIDEOUT_TOKEN="$(insideout login ...)"`, an access token that
    nothing renews, though `insideout login` already saves a login that does
    renew. The blurb teaches login then whoami, keeps INSIDEOUT_TOKEN as the
    headless or CI alternative, and says how a stranger gets an account: an
    invitation code from the server's administrator."""
    code, out, err = run("setup.sh", ["--hosted"], tools=["dirname", "sed", "cat"])
    if code != 0:
        fail(failures, f"scripts/setup.sh --hosted: exit {code}, stderr {err.strip()!r}; want 0")
        return
    lines = out.splitlines()
    login = line_at(lines, "insideout login you@example.com")
    whoami = line_at(lines, "insideout whoami")
    if login < 0 or whoami < 0 or login > whoami:
        fail(failures, "scripts/setup.sh --hosted does not teach `insideout login you@example.com` "
                       "and then `insideout whoami`")
    for no, line in enumerate(lines):
        if "INSIDEOUT_TOKEN" not in line:
            continue
        if "insideout login" in line:
            fail(failures, "scripts/setup.sh --hosted teaches an exported INSIDEOUT_TOKEN from `insideout login` "
                           "as the sign-in; the saved login renews, an exported token does not")
        elif whoami < 0 or no < whoami or not re.search(r"headless|\bCI\b", paragraph(lines, no), re.I):
            fail(failures, "scripts/setup.sh --hosted mentions INSIDEOUT_TOKEN other than as the headless or CI "
                           "alternative after the sign-in lines")
    register = line_at(lines, "insideout register --invite <invitation-code> you@example.com")
    if register < 0 or register < whoami:
        fail(failures, "scripts/setup.sh --hosted does not give the sign-up route "
                       "`insideout register --invite <invitation-code> you@example.com` after the sign-in lines")
    name_invitation(failures, "scripts/setup.sh --hosted", out)


def name_invitation(failures, where, text):
    """The one name for the account invitation, shared with the web copy
    (USER-10): an invitation code, from the server's administrator."""
    if "invitation code" not in text or "administrator" not in text:
        fail(failures, f"{where}: the sign-up route does not say an invitation code comes from the "
                       "server's administrator")
    if re.search(r"invite code", text, re.I):
        fail(failures, f"{where}: calls the account invitation an \"invite code\"; that is the workspace "
                       "code's old name, the account's is \"invitation code\"")


def closing_messages_teach_the_sign_up_route(failures):
    """LEAD-SETUP-BLURB: the installers' closing messages end where a stranger
    needs an account; they give the same route as the blurb."""
    install = read("scripts/install.sh")
    if "api_base=" not in install:
        print("no api_base= closing block in scripts/install.sh", file=sys.stderr)
        sys.exit(2)
    bootstrap = re.search(r"cat <<EOF\n(.*?)\nEOF(?:\n|\Z)", read("scripts/bootstrap.sh"), re.S)
    if not bootstrap:
        print("no closing `cat <<EOF` message in scripts/bootstrap.sh", file=sys.stderr)
        sys.exit(2)
    for where, text in (("scripts/install.sh", install.split("api_base=", 1)[1]),
                        ("scripts/bootstrap.sh", bootstrap.group(1))):
        if "insideout register --invite <invitation-code> you@example.com" not in text:
            fail(failures, f"{where}: the closing message does not give the sign-up route "
                           "`insideout register --invite <invitation-code> you@example.com`")
        name_invitation(failures, f"{where} closing message", text)


def dart_value(path, key):
    """One single-quoted string from a client l10n map, the vocabulary the
    screens actually render."""
    match = re.search(r"'" + re.escape(key) + r"':\s*(?:\n\s*)?'([^']*)'", read(path))
    if not match or not match.group(1):
        print(f"no {key} string in {path}", file=sys.stderr)
        sys.exit(2)
    return match.group(1)


def two_code_names():
    """USER-10: the workspace share code and the account invitation, as the
    English catalogue names them. Callers compare case-insensitively."""
    workspace = dart_value("client/lib/l10n/en_workspace.dart", "workspace.joinCodePlaceholder")
    invitation = dart_value("client/lib/l10n/en_account.dart", "register.invite")
    return workspace.lower(), invitation.lower()


def verb_summaries():
    found = dict(re.findall(r'\{group\w+, "([a-z][a-z-]*)", "([^"]*)"\}', read("server/cmd/insideout/cli.go")))
    if "join" not in found or "register" not in found:
        print("cannot read join and register from the verb table in server/cmd/insideout/cli.go", file=sys.stderr)
        sys.exit(2)
    return found


def hosted_case_text():
    """The `case *hosted:` arm of runSetupCommand, and nothing it prints for
    the local path."""
    text = read("server/cmd/insideout/cli_setup.go")
    start = text.find("case *hosted:")
    end = text.find("case *local:", start)
    if start < 0 or end < 0:
        print("no hosted case in server/cmd/insideout/cli_setup.go", file=sys.stderr)
        sys.exit(2)
    return text[start:end]


def setup_sh_taught_lines():
    """The sign-in lines scripts/setup.sh --hosted teaches. Exit 2 when the
    script no longer has them: the Go blurb is checked against the script,
    not against a second copy of the words."""
    match = re.search(r"cat <<EOF\n(.*?)\nEOF", read("scripts/setup.sh"), re.S)
    if not match:
        print("no hosted blurb in scripts/setup.sh", file=sys.stderr)
        sys.exit(2)
    blurb = match.group(1)
    lines = ("insideout whoami", "insideout register --invite <invitation-code> you@example.com")
    for line in lines:
        if line not in blurb:
            print(f"scripts/setup.sh blurb has no `{line}`", file=sys.stderr)
            sys.exit(2)
    return lines


def cli_and_docs_name_the_two_codes(failures):
    """USER-10 and the hosted half of PM-08. The web copy has two names. The
    CLI help, the register usage line, `setup --hosted` and two docs still
    said "invite code" for both, and the hosted blurb did not teach whoami
    or the invitation. `insideout setup --local` in that blurb exits 2 with
    no checkout (cli_setup_local_env.go); the script's blurb does not say it."""
    workspace, invitation = two_code_names()
    join = verb_summaries()["join"]
    if workspace not in join.lower():
        fail(failures, f"server/cmd/insideout/cli.go: join summary {join!r} does not name the {workspace}")
    usage = next((line for line in read("server/cmd/insideout/cli_auth.go").splitlines()
                  if "usage: insideout register" in line), "")
    if not usage:
        print("no register usage line in server/cmd/insideout/cli_auth.go", file=sys.stderr)
        sys.exit(2)
    if invitation not in usage.lower():
        fail(failures, "server/cmd/insideout/cli_auth.go: the register usage line does not say "
                       f"{invitation!r} (the flag stays --invite): {usage.strip()}")
    hosted = hosted_case_text()
    for line in setup_sh_taught_lines():
        if line not in hosted:
            fail(failures, "server/cmd/insideout/cli_setup.go hosted blurb does not teach "
                           f"`{line}`, which scripts/setup.sh does")
    name_invitation(failures, "server/cmd/insideout/cli_setup.go hosted blurb", hosted)
    if "insideout setup --local" in hosted and "no repository checkout found" in read(
            "server/cmd/insideout/cli_setup_local_env.go"):
        fail(failures, "server/cmd/insideout/cli_setup.go hosted blurb tells the reader to run "
                       "`insideout setup --local`, which exits 2 when there is no repository checkout")
    surfaces = ("server/cmd/insideout/cli.go", "server/cmd/insideout/cli_auth.go",
                "server/cmd/insideout/cli_setup.go", "docs/usage/cli-mcp-and-guidance.md",
                "docs/architecture/database-and-rls.md", "README.md", "docs/usage/installation.md")
    for where in surfaces:
        if re.search(r"invite code", read(where), re.I):
            fail(failures, f"{where}: still says \"invite code\"; the account's word is {invitation!r} "
                           f"and the workspace's is {workspace!r}")
    joining = next((line for line in read("docs/architecture/database-and-rls.md").splitlines()
                    if "Joining a workspace" in line), "")
    if workspace not in joining.lower():
        fail(failures, "docs/architecture/database-and-rls.md: the workspace-join sentence does not say "
                       f"{workspace!r}")
    if invitation not in read("docs/usage/cli-mcp-and-guidance.md").lower():
        fail(failures, f"docs/usage/cli-mcp-and-guidance.md: does not call the sign-up code an {invitation!r}")


def quoted_commands_exist(failures):
    """Every `insideout <verb>` a script prints is a verb of the CLI, and the
    sign-up flag it quotes is one `register` documents."""
    verbs = dict(re.findall(r'\{group\w+, "([a-z][a-z-]*)", "([^"]*)"\}', read("server/cmd/insideout/cli.go")))
    if len(verbs) < 20 or "register" not in verbs:
        print("cannot read the verb table in server/cmd/insideout/cli.go", file=sys.stderr)
        sys.exit(2)
    _, blurb, _ = run("setup.sh", ["--hosted"], tools=["dirname", "sed", "cat"])
    bootstrap = re.search(r"cat <<EOF\n(.*?)\nEOF(?:\n|\Z)", read("scripts/bootstrap.sh"), re.S)
    texts = {"scripts/setup.sh --hosted": blurb, "scripts/install.sh": read("scripts/install.sh"),
             "scripts/bootstrap.sh": bootstrap.group(1) if bootstrap else ""}
    for where, text in texts.items():
        for verb in re.findall(r'^\s*(?:echo\s+")?\s*insideout (?!-)([a-z][a-z-]*)', text, re.M):
            if verb not in verbs:
                fail(failures, f"{where} prints `insideout {verb}`, which is not a verb of the CLI")
        if "register --invite" in text and "--invite" not in verbs["register"]:
            fail(failures, f"{where} quotes `register --invite`, which `insideout help` no longer lists")


CHECKS = [installer_headers_state_the_supported_systems,
          bootstrap_refuses_another_system_by_naming_the_supported_two,
          install_without_go_offers_the_prebuilt_installer_first,
          hosted_blurb_teaches_login_then_whoami,
          closing_messages_teach_the_sign_up_route,
          cli_and_docs_name_the_two_codes,
          quoted_commands_exist]

if __name__ == "__main__":
    failures = []
    for check in CHECKS:
        check(failures)
    for message in failures:
        print(message, file=sys.stderr)
    if failures:
        sys.exit(1)
    print(f"first-run copy ok: {len(CHECKS)} checks")
