#!/usr/bin/env python3
"""Does the app's nginx follow its API upstream when DNS changes? (BUG-027)

Renders client/nginx.conf the way the official image's entrypoint does
(envsubst of NGINX_API_UPSTREAM and NGINX_LOCAL_RESOLVERS), runs it on a real
nginx next to a stub DNS server and two upstreams on two local addresses, then
checks that the proxy

  1. starts while the upstream name does not resolve yet, and answers 502,
  2. finds the upstream once the name resolves,
  3. follows the name to the new address after the upstream is redeployed,
  4. still passes the query string and headers, and a chunked response still
     arrives chunk by chunk (nginx forwards a slow stream as it comes, with
     proxy_buffering on or off, so this guards the request path, not that
     directive).

Usage:  python3 scripts/check_nginx_reresolve.py [--template FILE] [--nginx BIN]

Needs Python 3 and an nginx binary (--nginx, else $NGINX_BIN, else PATH). It
installs nothing and writes only inside a temp directory.
"""
import argparse
import http.client
import os
import re
import shutil
import socket
import subprocess
import sys
import tempfile
import threading
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

HOST = "api.upstream.invalid"  # .invalid never resolves outside the stub


class Dns(threading.Thread):
    """UDP DNS on 127.0.0.1: an A query gets `answer` (NXDOMAIN while None),
    anything else gets an empty NOERROR; the TTL is 1s."""

    def __init__(self):
        super().__init__(daemon=True)
        self.answer = None
        self.sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        self.sock.bind(("127.0.0.1", 0))
        self.port = self.sock.getsockname()[1]

    def run(self):
        while True:
            try:
                query, peer = self.sock.recvfrom(512)
            except OSError:
                return
            end = query.index(b"\0", 12) + 5  # name, then type and class
            qtype = int.from_bytes(query[end - 4 : end - 2], "big")
            ip = self.answer if qtype == 1 else None
            rcode = 0 if (ip or qtype != 1) else 3
            head = query[:2] + (0x8180 | rcode).to_bytes(2, "big")
            head += (1).to_bytes(2, "big") + (1 if ip else 0).to_bytes(2, "big")
            reply = head + bytes(4) + query[12:end]
            if ip:
                reply += b"\xc0\x0c\x00\x01\x00\x01\x00\x00\x00\x01\x00\x04"
                reply += socket.inet_aton(ip)
            self.sock.sendto(reply, peer)

    def stop(self):
        self.sock.close()


def upstream(ip, port, label):
    """An HTTP server that says who it is, echoes what nginx forwarded, and
    can stream two chunks 1.5s apart."""

    class Handler(BaseHTTPRequestHandler):
        protocol_version = "HTTP/1.1"

        def do_GET(self):
            if self.path.startswith("/api/v1/stream"):
                self.send_response(200)
                self.send_header("Transfer-Encoding", "chunked")
                self.end_headers()
                for part in (b"chunk-1\n", b"chunk-2\n"):
                    self.wfile.write(b"%x\r\n%s\r\n" % (len(part), part))
                    self.wfile.flush()
                    time.sleep(1.5)
                self.wfile.write(b"0\r\n\r\n")
                return
            h = self.headers
            body = (
                f"{label} {self.path} host={h.get('Host')} "
                f"proto={h.get('X-Forwarded-Proto')} auth={h.get('Authorization')}\n"
            ).encode()
            self.send_response(200)
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)

        def log_message(self, *_):
            pass

    srv = ThreadingHTTPServer((ip, port), Handler)
    threading.Thread(target=srv.serve_forever, daemon=True).start()
    return srv


def free_port():
    with socket.socket() as s:
        s.bind(("127.0.0.1", 0))
        return s.getsockname()[1]


def second_address():
    """Another local IPv4 address: 127.0.0.2 on Linux, else the primary
    interface (macOS only has 127.0.0.1 on loopback)."""
    try:
        with socket.socket() as s:
            s.bind(("127.0.0.2", 0))
            return "127.0.0.2"
    except OSError:
        pass
    with socket.socket(socket.AF_INET, socket.SOCK_DGRAM) as s:
        s.connect(("192.0.2.1", 9))  # no packet is sent
        return s.getsockname()[0]


def render(template, upstream_url, resolvers, port):
    """The entrypoint's envsubst for the two variables, on a free listen port."""
    text = Path(template).read_text()
    text = text.replace("${NGINX_API_UPSTREAM}", upstream_url)
    text = text.replace("${NGINX_LOCAL_RESOLVERS}", resolvers)
    left = re.findall(r"\$\{[A-Z_]+\}", text)
    if left:
        sys.exit(f"{template}: unsubstituted {left}")
    return re.sub(r"listen (\[::\]:)?8080;", lambda m: f"listen {m.group(1) or ''}{port};", text)


def get(port, path, headers=None):
    conn = http.client.HTTPConnection("127.0.0.1", port, timeout=8)
    try:
        conn.request("GET", path, headers=headers or {})
        r = conn.getresponse()
        return r.status, r.read().decode(errors="replace")
    finally:
        conn.close()


def stream_gap(port):
    """Seconds between the first and the second chunk reaching the client."""
    s = socket.create_connection(("127.0.0.1", port), timeout=10)
    s.sendall(b"GET /api/v1/stream HTTP/1.1\r\nHost: app.example\r\nConnection: close\r\n\r\n")
    t0, seen, buf = time.monotonic(), {}, b""
    while len(seen) < 2:
        data = s.recv(4096)
        if not data:
            break
        buf += data
        for tag in (b"chunk-1", b"chunk-2"):
            if tag in buf and tag not in seen:
                seen[tag] = time.monotonic() - t0
    s.close()
    return seen[b"chunk-2"] - seen[b"chunk-1"] if len(seen) == 2 else None


def main():
    root = Path(__file__).resolve().parent.parent
    ap = argparse.ArgumentParser()
    ap.add_argument("--template", default=str(root / "client" / "nginx.conf"))
    ap.add_argument("--nginx", default=os.environ.get("NGINX_BIN") or shutil.which("nginx"))
    args = ap.parse_args()
    if not args.nginx:
        sys.exit("no nginx binary: pass --nginx or set NGINX_BIN")

    ip_a, ip_b = "127.0.0.1", second_address()
    up_port, web_port = free_port(), free_port()
    dns = Dns()
    dns.start()
    tmp = Path(tempfile.mkdtemp(prefix="nginx-reresolve-"))
    site = render(args.template, f"http://{HOST}:{up_port}", f"127.0.0.1:{dns.port}", web_port)
    (tmp / "logs").mkdir()  # the default error log path, relative to -p
    (tmp / "site.conf").write_text(site)
    (tmp / "nginx.conf").write_text(
        f"daemon off;\nworker_processes 1;\npid {tmp}/nginx.pid;\nerror_log {tmp}/error.log info;\n"
        f"events {{ worker_connections 64; }}\nhttp {{\n  access_log {tmp}/access.log;\n"
        + "".join(f"  {d}_temp_path {tmp}/{d};\n" for d in ("client_body", "proxy", "fastcgi", "uwsgi", "scgi"))
        + f"  include {tmp}/site.conf;\n}}\n"
    )
    m = re.search(r"valid=(\d+)s", site)
    wait = int(m.group(1)) + 2 if m else 3  # past the resolver's cache window
    up_a = up_b = proc = None
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
                if get(web_port, "/healthz")[0] == 200:
                    break
            except OSError:
                time.sleep(0.1)
        up = proc.poll() is None
        check("nginx starts while the upstream name does not resolve", up, (proc.stdout.read() if not up else "").strip())
        if not up:
            return 1

        status, _ = get(web_port, "/api/v1/x?q=1")
        check("unresolvable upstream answers 502, not a hang", status == 502, f"status {status}")

        up_a = upstream(ip_a, up_port, "A")
        dns.answer = ip_a
        time.sleep(wait)
        hdr = {"Host": "app.example", "Authorization": "Bearer t0k"}
        status, body = get(web_port, "/api/v1/x?q=1", hdr)
        check("finds the upstream once the name resolves", body.startswith("A /api/v1/x?q=1"), f"{status} {body!r}")
        check(
            "passes Host, Authorization and X-Forwarded-Proto",
            "host=app.example proto=http auth=Bearer t0k" in body,
            body,
        )

        up_b = upstream(ip_b, up_port, "B")
        dns.answer = ip_b
        up_a.shutdown()
        up_a.server_close()  # the redeploy: the old address is gone
        time.sleep(wait)
        status, body = get(web_port, "/api/v1/x?q=1", hdr)
        check("follows the name to the new address after a redeploy", body.startswith("B /api/v1/x?q=1"), f"{status} {body!r}")

        gap = stream_gap(web_port)
        check("a chunked response still arrives chunk by chunk", gap is not None and gap >= 1.0, f"gap {gap}")
    finally:
        if proc and proc.poll() is None:
            proc.terminate()
            proc.wait(timeout=10)
        for srv in (up_a, up_b):
            if srv:
                srv.shutdown()
        dns.stop()
        shutil.rmtree(tmp, ignore_errors=True)
    print("OK" if not failures else f"{len(failures)} check(s) failed")
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
