#!/usr/bin/env python3
"""Does the app's nginx pass GitHub webhook deliveries up to the server's cap?

The server accepts 8 MiB bodies on the GitHub webhook route alone
(maxWebhookBodyBytes, server/internal/api/middleware.go) because a push
delivery lists every commit's files and routinely outgrows 1 MiB. In the
hosted topology the delivery reaches the server through this nginx first
(GitHub -> /api/v1/hooks/github -> app -> server), and nginx's own
client_max_body_size defaults to 1 MiB — without an explicit limit on the
webhook location nginx answers 413 before the server ever sees the body.

Static check, Python 3 only (no nginx binary needed): an exact-match
location wins over the /api/ prefix location (nginx docs, "How nginx
processes a request"), so the config shape is checkable by parsing. Live
nginx behaviour (nginx -t, the BUG-027 re-resolution) stays with
scripts/check_nginx_reresolve.py in the CI nginx job. Fails unless

  1. an exact-match location for /api/v1/hooks/github exists,
  2. its client_max_body_size is at least the server's 8 MiB,
  3. it proxies to the same variable upstream as location /api/
     (set + proxy_pass $api_upstream: `set` runs per matched location,
     it is not inherited, and a variable keeps the per-request resolve),
  4. it carries every proxy directive location /api/ has.

Usage:  python3 scripts/check_nginx_webhook_body.py [--template FILE]
"""
import argparse
import re
import sys
from pathlib import Path

WEBHOOK_PATH = "/api/v1/hooks/github"  # githubWebhookPath in the server
SERVER_CAP = 8 * 1024 * 1024  # maxWebhookBodyBytes, server/internal/api/middleware.go

SIZE = re.compile(r"^(\d+)([kKmMgG]?)$")
UNIT = {"": 1, "k": 1024, "m": 1024**2, "g": 1024**3}


def location_blocks(text):
    """Yield (modifier, path, body) for every `location` block in the text."""
    for m in re.finditer(r"location\s+(?:(=|~|~\*|\^~)\s+)?(\S+)\s*\{", text):
        depth, i = 1, m.end()
        while depth and i < len(text):
            depth += (text[i] == "{") - (text[i] == "}")
            i += 1
        yield m.group(1), m.group(2), text[m.end() : i - 1]


def directives(body):
    """The `;`-terminated statements of a block, comments dropped,
    whitespace collapsed — so blocks compare as directive sets."""
    body = re.sub(r"#[^\n]*", "", body)
    out, depth, start = [], 0, 0
    for i, ch in enumerate(body):
        if ch == "{":
            depth += 1
        elif ch == "}":
            depth -= 1
        elif ch == ";" and depth == 0:
            stmt = " ".join(body[start:i].split())
            if stmt:
                out.append(stmt)
            start = i + 1
    return out


def to_bytes(value):
    """nginx size literal (8, 8k, 8m, 8g — units are case-insensitive)."""
    m = SIZE.match(value.strip())
    return int(m.group(1)) * UNIT[m.group(2).lower()] if m else None


def main():
    root = Path(__file__).resolve().parent.parent
    ap = argparse.ArgumentParser()
    ap.add_argument("--template", default=str(root / "client" / "nginx.conf"))
    args = ap.parse_args()

    blocks = list(location_blocks(Path(args.template).read_text()))
    api = next((b for mod, p, b in blocks if mod is None and p == "/api/"), None)
    hook = next((b for mod, p, b in blocks if mod == "=" and p == WEBHOOK_PATH), None)

    failures = []

    def check(name, ok, detail=""):
        print(("PASS  " if ok else "FAIL  ") + name + ("" if ok else f"  [{detail}]"))
        if not ok:
            failures.append(name)

    check(
        f"exact-match location = {WEBHOOK_PATH} exists (it wins over /api/)",
        hook is not None,
        "locations present: " + ", ".join((mod or "") + " " + p for mod, p, _ in blocks),
    )
    if hook is None or api is None:
        print("OK" if not failures else f"{len(failures)} check(s) failed")
        return 1 if failures else 0

    hook_dirs, api_dirs = directives(hook), directives(api)

    size_stmt = next((d for d in hook_dirs if d.startswith("client_max_body_size ")), None)
    size = to_bytes(size_stmt.split()[1]) if size_stmt else None
    check(
        f"webhook body cap is at least the server's {SERVER_CAP // (1024 * 1024)} MiB",
        size is not None and size >= SERVER_CAP,
        "client_max_body_size missing" if size_stmt is None else f"{size} bytes",
    )

    api_pass = next((d for d in api_dirs if d.startswith("proxy_pass ")), None)
    check(
        f"proxies to /api/'s variable upstream ({api_pass})",
        api_pass is not None
        and api_pass.split()[1].startswith("$")
        and api_pass in hook_dirs,
        "the webhook must set and proxy the same $variable as /api/, or a literal "
        "proxy_pass would freeze the upstream BUG-027 resolved per request",
    )

    missing = [d for d in api_dirs if d not in hook_dirs]
    check(
        "carries every proxy directive location /api/ has",
        not missing,
        f"missing {missing}",
    )

    print("OK" if not failures else f"{len(failures)} check(s) failed")
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
