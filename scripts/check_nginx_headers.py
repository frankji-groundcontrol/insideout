#!/usr/bin/env python3
"""Does the app's nginx send its security headers? (SEC-12)

Renders client/nginx.conf the way the official image's entrypoint does, points
its `root` at a stub index.html and runs it on a real nginx, then reads the
response headers of

  - the SPA shell (/) and a deep route that falls back to index.html,
  - /healthz, which nginx answers itself with `return`,
  - /api/ with an upstream that does not resolve, so nginx's own 502 page
    (the headers are set `always`, not only on 2xx/3xx),
  - /fonts/, once for a mirrored font file and once for one that is not
    mirrored (a 404, never the SPA shell: the engine would feed index.html to
    its font parser),
  - /canvaskit/, a .wasm file, which must come gzip'd (the image bundles
    CanvasKit itself, and the CDN it replaces compressed it),

and checks each carries, exactly once, nosniff, a strict Referrer-Policy,
X-Frame-Options DENY and a Content-Security-Policy that denies plugins,
framing and base-tag injection and allows no inline or eval'd script. The
policy must also name no host beyond CSP_HOSTS_ALLOWED and give scripts and
styles to this origin alone (rv4-COMP-03: a first load must not need a CDN).
HSTS must appear only when the request says it came over https (the edge in
front of nginx terminates TLS, so nginx itself only ever sees http).

Usage:  python3 scripts/check_nginx_headers.py [--template FILE] [--nginx BIN]

Needs Python 3 and an nginx binary (--nginx, else $NGINX_BIN, else PATH). It
installs nothing and writes only inside a temp directory.
"""
import argparse
import http.client
import os
import re
import shutil
import subprocess
import sys
import tempfile
import time
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))  # python -I drops the script's dir
from check_nginx_reresolve import HOST, Dns, free_port, render  # noqa: E402

ROOT_DIRECTIVE = "root /usr/share/nginx/html;"
# directive -> sources it must list exactly (the rest of the policy may move
# with what the built app needs; these are the parts that do not)
CSP_EXACT = {
    "default-src": ["'self'"],
    "object-src": ["'none'"],
    "base-uri": ["'self'"],
    "frame-ancestors": ["'none'"],
}
CSP_NEVER_IN_SCRIPT = ("'unsafe-inline'", "'unsafe-eval'")
# The only hosts the policy may name (rv4-COMP-03): none. The engine reads its
# Roboto and Noto fallback fonts from fontFallbackBaseUrl, which
# web/flutter_bootstrap.js points at /fonts/s/ (every file the engine can ask
# for is mirrored there). Keep it equal to engineFontHosts in
# client/test/web_csp_self_test.dart.
CSP_HOSTS_ALLOWED = ()


def fetch(port, path, headers=None):
    """(status, {lowercase header name: [values]}) for a GET."""
    conn = http.client.HTTPConnection("127.0.0.1", port, timeout=8)
    try:
        conn.request("GET", path, headers=headers or {})
        r = conn.getresponse()
        r.read()
        seen = {}
        for name, value in r.getheaders():
            seen.setdefault(name.lower(), []).append(value)
        return r.status, seen
    finally:
        conn.close()


def one(seen, name):
    """The header's single value, or None when it is absent or repeated."""
    values = seen.get(name, [])
    return values[0] if len(values) == 1 else None


def csp(seen):
    """The Content-Security-Policy as {directive: [sources]}."""
    out = {}
    for part in (one(seen, "content-security-policy") or "").split(";"):
        words = part.split()
        if words:
            out[words[0]] = words[1:]
    return out


def main():
    root = Path(__file__).resolve().parent.parent
    ap = argparse.ArgumentParser()
    ap.add_argument("--template", default=str(root / "client" / "nginx.conf"))
    ap.add_argument("--nginx", default=os.environ.get("NGINX_BIN") or shutil.which("nginx"))
    args = ap.parse_args()
    if not args.nginx:
        sys.exit("no nginx binary: pass --nginx or set NGINX_BIN")

    web_port = free_port()
    dns = Dns()  # answers NXDOMAIN, so /api/ gets nginx's own 502 at once
    dns.start()
    tmp = Path(tempfile.mkdtemp(prefix="nginx-headers-"))
    web = tmp / "web"
    web.mkdir()
    (web / "index.html").write_text("<!doctype html><title>stub</title>\n")
    (web / "fonts" / "s").mkdir(parents=True)
    (web / "fonts" / "s" / "stub.woff2").write_bytes(b"wOF2")
    (web / "canvaskit").mkdir()
    (web / "canvaskit" / "stub.wasm").write_bytes(b"\0asm" + bytes(8192))  # compressible
    site = render(args.template, f"http://{HOST}:1", f"127.0.0.1:{dns.port}", web_port)
    if ROOT_DIRECTIVE not in site:
        sys.exit(f"{args.template}: no `{ROOT_DIRECTIVE}` to repoint at the stub site")
    (tmp / "logs").mkdir()  # the default error log path, relative to -p
    (tmp / "site.conf").write_text(site.replace(ROOT_DIRECTIVE, f"root {web};"))
    (tmp / "nginx.conf").write_text(
        f"daemon off;\nworker_processes 1;\npid {tmp}/nginx.pid;\nerror_log {tmp}/error.log info;\n"
        # The image's nginx.conf includes mime.types, which maps .wasm to
        # application/wasm (nginx 1.27.0 has the line); only that mapping is
        # declared here, so gzip_types matching it is what gets tested.
        f"events {{ worker_connections 64; }}\nhttp {{\n  types {{ application/wasm wasm; }}\n  access_log {tmp}/access.log;\n"
        + "".join(f"  {d}_temp_path {tmp}/{d};\n" for d in ("client_body", "proxy", "fastcgi", "uwsgi", "scgi"))
        + f"  include {tmp}/site.conf;\n}}\n"
    )
    proc = None
    failures = []

    def check(name, ok, detail=""):
        print(("PASS  " if ok else "FAIL  ") + name + ("" if ok else f"  [{detail}]"))
        if not ok:
            failures.append(name)

    try:
        proc = subprocess.Popen(
            [args.nginx, "-p", str(tmp), "-c", str(tmp / "nginx.conf")],
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
        )
        for _ in range(50):
            if proc.poll() is not None:
                break
            try:
                if fetch(web_port, "/healthz")[0] == 200:
                    break
            except OSError:
                time.sleep(0.1)
        up = proc.poll() is None
        check("nginx starts with the rendered template", up, (proc.stdout.read() if not up else "").strip())
        if not up:
            return 1

        pages = [
            ("the SPA shell /", "/", 200),
            ("a deep SPA route", "/projects/p1/roadmap", 200),
            ("/healthz", "/healthz", 200),
            ("nginx's own 502 page", "/api/v1/x", 502),
        ]
        for label, path, want in pages:
            status, seen = fetch(web_port, path)
            check(f"{label} answers {want}", status == want, f"status {status}")
            check(f"{label}: X-Content-Type-Options nosniff", one(seen, "x-content-type-options") == "nosniff", seen.get("x-content-type-options"))
            check(
                f"{label}: Referrer-Policy strict-origin-when-cross-origin",
                one(seen, "referrer-policy") == "strict-origin-when-cross-origin",
                seen.get("referrer-policy"),
            )
            check(f"{label}: X-Frame-Options DENY", one(seen, "x-frame-options") == "DENY", seen.get("x-frame-options"))
            policy = csp(seen)
            for directive, want_sources in CSP_EXACT.items():
                check(
                    f"{label}: CSP {directive} {' '.join(want_sources)}",
                    policy.get(directive) == want_sources,
                    seen.get("content-security-policy"),
                )
            script = policy.get("script-src", policy.get("default-src", []))
            check(
                f"{label}: CSP lets no inline or eval'd script run",
                bool(policy) and not any(s in script for s in CSP_NEVER_IN_SCRIPT),
                seen.get("content-security-policy"),
            )
            hosts = {s for sources in policy.values() for s in sources if s.startswith(("http://", "https://"))}
            check(
                f"{label}: CSP names no host beyond {', '.join(CSP_HOSTS_ALLOWED) or 'none'}",
                bool(policy) and hosts <= set(CSP_HOSTS_ALLOWED),
                sorted(hosts),
            )
            for directive in ("default-src", "script-src", "style-src"):
                listed = [s for s in policy.get(directive, []) if s.startswith(("http://", "https://"))]
                check(f"{label}: CSP {directive} lists no host", bool(policy) and not listed, listed)
            check(f"{label}: no HSTS without an https request", "strict-transport-security" not in seen, seen.get("strict-transport-security"))

        for label, path, want in (
            ("a mirrored font file", "/fonts/s/stub.woff2", 200),
            ("a font file that is not mirrored", "/fonts/s/missing.woff2", 404),
        ):
            status, seen = fetch(web_port, path)
            check(f"{label} answers {want}, never the SPA shell", status == want, f"status {status}")
            check(
                f"{label}: nosniff and one CSP",
                one(seen, "x-content-type-options") == "nosniff" and one(seen, "content-security-policy") is not None,
                seen.get("content-security-policy"),
            )

        status, seen = fetch(web_port, "/canvaskit/stub.wasm", {"Accept-Encoding": "gzip"})
        check("a bundled CanvasKit .wasm answers 200", status == 200, f"status {status}")
        check("a bundled CanvasKit .wasm is gzip'd", one(seen, "content-encoding") == "gzip", seen.get("content-encoding"))

        _, seen = fetch(web_port, "/", {"X-Forwarded-Proto": "https"})
        age = re.match(r"max-age=(\d+)", one(seen, "strict-transport-security") or "")
        check("HSTS on a request the edge marked https", bool(age) and int(age.group(1)) >= 31536000, seen.get("strict-transport-security"))
        _, seen = fetch(web_port, "/", {"X-Forwarded-Proto": "http"})
        check("no HSTS when the edge marked the request http", "strict-transport-security" not in seen, seen.get("strict-transport-security"))
    finally:
        if proc and proc.poll() is None:
            proc.terminate()
            proc.wait(timeout=10)
        dns.stop()
        shutil.rmtree(tmp, ignore_errors=True)
    print("OK" if not failures else f"{len(failures)} check(s) failed")
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
