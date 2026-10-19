#!/usr/bin/env python3
"""Fail when scripts/bootstrap.sh trusts a download more than it should
(review 4, wave 4: rv4-SEC-07 and rv4-PUB-03, the consumer side of a release).

Usage: python3 -I scripts/tests/check_bootstrap_release.py

Stdlib only and no network: the release is a throwaway mirror on 127.0.0.1.
bootstrap.sh is piped to bash on stdin, the way `curl | bash` runs it, on a PATH
that holds only symlinks to the real tools it calls (plus the stubs a case
asks for) and a scratch HOME, so a curl or gh configured on the machine running
the guard changes nothing. /bin/bash is used when it exists: on a Mac that is
bash 3.2, the oldest the script has to run under. Exits 1 naming every failed
property, 2 when a premise cannot be derived.

- rv4-SEC-07: INSIDEOUT_RELEASE_BASE must be https. Plain http is accepted only
  for a mirror on this machine, and only when the host really is loopback: a
  name that starts with localhost, or user-info before an @, is not. Every
  request, and every redirect, is https-only (curl --proto-redir =https), so a
  mirror cannot step the download down to plain http. The docs say so.

CI-config property of a script a release ships: the first real proof is the
first tag run after the workflow lands; this runs it against a local mirror.
"""
import collections
import hashlib
import http.server
import io
import pathlib
import platform
import re
import shutil
import subprocess
import sys
import tarfile
import tempfile
import threading

ROOT = pathlib.Path(__file__).resolve().parents[2]
SCRIPT = ROOT / "scripts" / "bootstrap.sh"
BASH = "/bin/bash" if pathlib.Path("/bin/bash").exists() else shutil.which("bash")
NEED = ("uname", "mktemp", "rm", "mkdir", "curl", "tar", "awk", "install", "cat", "grep", "basename")
MAYBE = ("gzip", "shasum", "sha256sum")  # GNU tar calls gzip; the script takes either hasher
STUB_CURL = 'echo "$*" >> "$STUB_LOG"; exit 22'  # records the request, then fails like a refused download

REFUSED = (
    "http://example.invalid/releases",  # plain http away from this machine
    "http://localhost.example.invalid/releases",  # looks local, is not
    "http://127.0.0.1.example.invalid/releases",
    "http://127.0.0.1evil.example.invalid/releases",
    "http://localhost@example.invalid/releases",  # the host is what follows the @
    "http://127.0.0.1:80@example.invalid/releases",
    "http://[::1]@example.invalid/releases",
    "ftp://example.invalid/releases",
    "file:///var/empty/releases",
    "example.invalid/releases",  # no scheme at all
)
ACCEPTED = (
    "https://example.invalid/releases",
    "http://localhost",  # a mirror on this machine, default port
    "http://localhost:8080/releases",
    "http://127.0.0.1:8080/releases",
    "http://[::1]:8080/releases",
)

Run = collections.namedtuple("Run", "code out err log installed seen")


def premise(message):
    print(message, file=sys.stderr)
    sys.exit(2)


def release_files():
    """The one archive bootstrap.sh asks for on this machine: its name, bytes and sha256."""
    kind = {"Darwin": "darwin", "Linux": "linux"}.get(platform.system())
    arch = {"arm64": "arm64", "aarch64": "arm64", "x86_64": "amd64", "AMD64": "amd64"}.get(platform.machine())
    if not (kind and arch):
        premise(f"cannot name this machine's release archive ({platform.system()}/{platform.machine()})")
    buf = io.BytesIO()
    with tarfile.open(fileobj=buf, mode="w:gz") as tar:
        for binary in ("insideout", "insideout-mcp"):
            body = f"#!/bin/sh\necho {binary} from the test release\n".encode()
            info = tarfile.TarInfo(binary)
            info.size, info.mode = len(body), 0o755
            tar.addfile(info, io.BytesIO(body))
    data = buf.getvalue()
    return f"insideout-{kind}-{arch}.tar.gz", data, hashlib.sha256(data).hexdigest()


class Handler(http.server.BaseHTTPRequestHandler):
    def log_message(self, *args):
        pass

    def do_GET(self):
        self.server.seen.append(self.path)
        if self.path.startswith("/redirect/"):  # every request is sent on to /good/ over plain http
            self.send_response(302)
            self.send_header("Location", f"http://127.0.0.1:{self.server.server_port}/good/{self.path[10:]}")
            self.send_header("Content-Length", "0")
            self.end_headers()
            return
        body = self.server.tree.get(self.path)
        if body is None:
            self.send_error(404)
            return
        self.send_response(200)
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)


class Mirror(http.server.ThreadingHTTPServer):
    """/good/ is a release, /badsum/ one whose checksums.txt lies, /redirect/ sends on to /good/."""

    def __init__(self, tree):
        super().__init__(("127.0.0.1", 0), Handler)
        self.tree, self.seen = tree, []


def run(base, curl=None, gh=None, env=None):
    """bootstrap.sh on stdin. `base` is INSIDEOUT_RELEASE_BASE (None leaves it unset)."""
    with tempfile.TemporaryDirectory() as tmp:
        tmp = pathlib.Path(tmp)
        bindir, home, inst = tmp / "bin", tmp / "home", tmp / "inst"
        bindir.mkdir()
        home.mkdir()
        for name in NEED + MAYBE:
            real = shutil.which(name)
            if real is None and name in NEED:
                premise(f"no {name} on PATH: the guard cannot build the tool directory bootstrap.sh runs with")
            if real and not (name == "curl" and curl):
                (bindir / name).symlink_to(real)
        for name, body in (("curl", curl), ("gh", gh)):
            if body:
                (bindir / name).write_text("#!/bin/sh\n" + body + "\n", encoding="utf-8")
                (bindir / name).chmod(0o755)
        environ = {"PATH": str(bindir), "HOME": str(home), "SHELL": "/bin/bash", "LC_ALL": "C",
                   "INSIDEOUT_INSTALL_DIR": str(inst), "STUB_LOG": str(tmp / "log"),
                   "STUB_INST": str(inst), "STUB_SEEN": str(tmp / "seen")}
        if base is not None:
            environ["INSIDEOUT_RELEASE_BASE"] = base
        environ.update(env or {})
        done = subprocess.run([BASH, "-s"], input=SCRIPT.read_text(encoding="utf-8"), cwd=tmp, env=environ,
                              capture_output=True, text=True, timeout=60)
        log = (tmp / "log").read_text(encoding="utf-8") if (tmp / "log").exists() else ""
        seen = (tmp / "seen").read_bytes() if (tmp / "seen").exists() else None
        installed = sorted(p.name for p in inst.iterdir()) if inst.exists() else []
        return Run(done.returncode, done.stdout, done.stderr, log, installed, seen)


def release_base_is_https(name, repo):
    """SEC-07: a base that is not https (or this machine) is refused before any request; the others are
    reached, and every request keeps redirects on https."""
    failures = []
    for base in REFUSED:
        r = run(base, curl=STUB_CURL)
        if r.code == 0 or r.log or "https" not in r.err:
            failures.append(f"INSIDEOUT_RELEASE_BASE={base}: want a refusal that names https before any request; "
                            f"got exit {r.code}, {len(r.log.splitlines())} request(s), stderr {r.err.strip()!r}")
    for base in ACCEPTED + (None,):
        r = run(base, curl=STUB_CURL)
        label = f"INSIDEOUT_RELEASE_BASE={base}" if base else "the default release base"
        want = f"{base}/{name}" if base else f"https://github.com/{repo}/releases/latest/download/{name}"
        lines = r.log.splitlines()
        if not any(want in line for line in lines):
            failures.append(f"{label}: no request for {want}; refused though it is https or this machine "
                            f"(stderr {r.err.strip()!r})")
        elif not all("--proto-redir =https" in line for line in lines):
            failures.append(f"{label}: a request leaves redirects open to plain http (no --proto-redir =https)")
    return failures


def redirects_stay_https(mirror):
    """SEC-07: a mirror that sends the download on to plain http is not followed."""
    good = run(f"http://127.0.0.1:{mirror.server_port}/good")
    if good.code != 0 or good.installed != ["insideout", "insideout-mcp"]:
        return [f"a release served from a loopback mirror over http was not installed: exit {good.code}, "
                f"installed {good.installed}, stderr {good.err.strip()!r}"]
    del mirror.seen[:]
    sent = run(f"http://127.0.0.1:{mirror.server_port}/redirect")
    if sent.code == 0 or sent.installed or any(p.startswith("/good/") for p in mirror.seen):
        return [f"a redirect to plain http was followed: exit {sent.code}, installed {sent.installed}, "
                f"requested {mirror.seen}"]
    return []


def docs_say_https():
    text = " ".join((ROOT / "docs/usage/installation.md").read_text(encoding="utf-8").split())
    if not re.search(r"INSIDEOUT_RELEASE_BASE[^.]*\bhttps\b", text):
        return ["docs/usage/installation.md: INSIDEOUT_RELEASE_BASE is described without saying it must be https"]
    return []


def main():
    if not SCRIPT.is_file() or not BASH:
        premise("scripts/bootstrap.sh or a bash to run it with is missing")
    if subprocess.run(["curl", "--proto-redir", "=https", "--version"], capture_output=True).returncode != 0:
        premise("this curl has no --proto-redir, so the redirect rule cannot be exercised")
    repo = re.search(r"^repo=(\S+)$", SCRIPT.read_text(encoding="utf-8"), re.M)
    if not repo:
        premise("no repo= line in scripts/bootstrap.sh")
    name, data, digest = release_files()
    tree = {}
    for folder, sha in (("good", digest), ("badsum", "0" * 64)):
        tree[f"/{folder}/{name}"] = data
        tree[f"/{folder}/checksums.txt"] = f"{sha}  {name}\n".encode()
    mirror = Mirror(tree)
    threading.Thread(target=mirror.serve_forever, daemon=True).start()
    try:
        failures = release_base_is_https(name, repo.group(1)) + redirects_stay_https(mirror) + docs_say_https()
    finally:
        mirror.shutdown()
        mirror.server_close()
    for failure in failures:
        print(failure, file=sys.stderr)
    if failures:
        sys.exit(1)
    print("bootstrap release ok: https-only release base and redirects, loopback mirror installs")


if __name__ == "__main__":
    main()
