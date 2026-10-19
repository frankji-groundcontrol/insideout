#!/usr/bin/env python3
"""Fail when the two halves of the shipped compose stack's client-address chain
disagree (rv4-FR-F61, F106, F51, F57, F99, F107, D02).

Usage: python3 -I scripts/tests/check_proxy_hops_rv4.py

The auth throttle and the resend cooldown key on clientIP: the TCP peer at
INSIDEOUT_TRUSTED_PROXY_HOPS=0, otherwise the Nth entry from the right of
X-Forwarded-For. Which address that entry is depends on the nginx in front of
the server:

  ends the header in the address it saw   HOPS 1 is right: the rightmost entry
  ($proxy_add_x_forwarded_for)            is the true peer, forgeries stay left
  passes the client's header through      no HOPS is right with nothing in front:
  ($http_x_forwarded_for)                 0 puts every user in nginx's one
                                          bucket, 1 lets a client pick its own

The pass-through is what the hosted image must keep (the Railway edge in front
already appended the client), so client/nginx.conf takes the value from
${NGINX_XFF}: the Dockerfile bakes the hosted default and docker-compose.yml
overrides it for the stack where nothing sits in front. Every property is read
from file text (stdlib only, no Docker, database or network). A demand
disappears with its premise; exit 1 names each offence, exit 2 means a premise
could not be derived (the style of check_ops_hardening.py). The nginx side runs
on a real nginx in scripts/check_nginx_reresolve.py.
"""
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parents[2]
HOSTED_DEFAULT = "$http_x_forwarded_for"  # the edge appended the client; nginx must not touch the chain
ENDS_IN_PEER = ("$proxy_add_x_forwarded_for", "$remote_addr")  # the rightmost entry is the address nginx saw
SERVER_DEFAULT = 0  # config.go: an unset INSIDEOUT_TRUSTED_PROXY_HOPS trusts no header
DOCS = ("docs/usage/deployment.md", "docs/usage/environment.md", ".env.example")
ADD_ONE = re.compile(r"(?i)\badd one\b.{0,160}?\bproxy")
LOCK_DOWN = re.compile(r"(?i)(?:CLIENT_PORT|client port).{0,240}?(?:127\.0\.0\.1|loopback|firewall)")
RIGHT_AT_ZERO = re.compile(r"(?i)default\s+`?0`?\s+is\s+right")
UNESCAPED_DOCKER = re.compile(r"(?<!\\)\$")
UNESCAPED_COMPOSE = re.compile(r"(?<!\$)\$(?!\$)")


def read(name):
    return (ROOT / name).read_text(encoding="utf-8")


def cannot_derive(message):
    print(message, file=sys.stderr)
    sys.exit(2)


def flat(name):
    """A file's text with comment markers and line breaks folded away, so a
    sentence wrapped across lines reads as one."""
    return " ".join(re.sub(r"(?m)^\s*#+ ?", "", read(name)).split())


def compose_service(name):
    """Text of one service of docker-compose.yml (a flat services map, so a cut
    at the next two-space key needs no YAML parser), or None."""
    match = re.search(rf"^  {re.escape(name)}:\n(.*?)(?=^  [A-Za-z_][\w-]*:|\Z)",
                      read("docker-compose.yml"), re.M | re.S)
    return match.group(1) if match else None


def compose_env(service, name):
    """Raw value of one environment entry of a compose service, or None."""
    match = re.search(rf"^\s+(?:-\s+)?{name}\s*[:=]\s*(\S+)", service, re.M)
    return match.group(1).strip("\"'") if match else None


def compose_hops_default():
    server = compose_service("server")
    if server is None:
        cannot_derive("no `server` service in docker-compose.yml")
    match = re.fullmatch(r"\$\{INSIDEOUT_TRUSTED_PROXY_HOPS:-(\d+)\}", compose_env(server, "INSIDEOUT_TRUSTED_PROXY_HOPS") or "")
    if not match:
        cannot_derive("the compose server service has no `INSIDEOUT_TRUSTED_PROXY_HOPS: ${INSIDEOUT_TRUSTED_PROXY_HOPS:-N}` entry")
    return int(match.group(1))


def nginx_xff_values():
    """The value of every `proxy_set_header X-Forwarded-For` line of client/nginx.conf."""
    values = re.findall(r"^\s*proxy_set_header\s+X-Forwarded-For\s+(\S+);", read("client/nginx.conf"), re.M)
    if not values:
        cannot_derive("no `proxy_set_header X-Forwarded-For` line in client/nginx.conf")
    return values


def docker_value(raw):
    """What Docker stores for `ENV NAME=raw`: `\\$` is a literal dollar sign and
    a bare `$name` expands to nothing at build time."""
    return re.sub(r"(?<!\\)\$\w+", "", raw).replace("\\$", "$")


def image_default():
    """(raw, stored) of the NGINX_XFF the client image bakes in; (None, None) without one."""
    match = re.search(r"^ENV NGINX_XFF=(\S*)\s*$", read("client/Dockerfile"), re.M)
    return (match.group(1), docker_value(match.group(1))) if match else (None, None)


def effective(value, client):
    """The header value nginx writes in the compose stack once envsubst has run."""
    if value != "${NGINX_XFF}":
        return value
    chosen = compose_env(client, "NGINX_XFF")
    return image_default()[1] if chosen is None else chosen.replace("$$", "$")


def template_is_the_switch(failures):
    """The image decides: nginx.conf names the variable, the Dockerfile bakes the
    hosted default (escaped, or Docker expands it to nothing and nginx fails to
    start on an empty header value), and compose escapes its override."""
    values = nginx_xff_values()
    for value in sorted(set(values) - {"${NGINX_XFF}"}):
        failures.append(f"client/nginx.conf: X-Forwarded-For is hard-coded to {value}; write ${{NGINX_XFF}} in "
                        "both locations so the image (hosted: pass the edge's chain through) and compose "
                        "(end the chain in the address nginx saw) can differ")
    if "${NGINX_XFF}" in values:
        raw, stored = image_default()
        if raw is None:
            failures.append("client/Dockerfile: no `ENV NGINX_XFF=` default, so the hosted image renders an "
                            "unknown nginx variable and nginx does not start")
        elif stored != HOSTED_DEFAULT:
            why = ("an unescaped dollar sign: Docker expands it at build time, so the baked value is empty and "
                   "`proxy_set_header X-Forwarded-For ;` stops nginx from starting; write \\$http_x_forwarded_for"
                   if UNESCAPED_DOCKER.search(raw) else "hosted behaviour must not change (the edge already appended the client)")
            failures.append(f"client/Dockerfile: `ENV NGINX_XFF={raw}` stores {stored!r}, not {HOSTED_DEFAULT}: {why}")
    client = compose_service("client")
    chosen = compose_env(client, "NGINX_XFF") if client else None
    if chosen is not None and UNESCAPED_COMPOSE.search(chosen):
        failures.append(f"docker-compose.yml: client NGINX_XFF is {chosen!r}; compose interpolates a single $, so "
                        "nginx would get an empty header value and not start: write $$ for a literal dollar sign")


def hops_follow_nginx(failures):
    """Server hop default and the compose nginx must agree (F61, F106, F51, F57, F99, D02)."""
    client = compose_service("client")
    if client is None:
        return  # no nginx in the compose stack: the server's own port is the front door
    hops = compose_hops_default()
    for value in sorted({effective(v, client) for v in nginx_xff_values()}, key=str):
        if value in ENDS_IN_PEER:
            if hops != 1:
                failures.append(f"docker-compose.yml: nginx ends X-Forwarded-For in the address it saw (one hop) but the "
                                f"server defaults INSIDEOUT_TRUSTED_PROXY_HOPS to {hops}")
        elif value == HOSTED_DEFAULT:
            failures.append("docker-compose.yml: nginx passes the client's X-Forwarded-For through untouched and nothing "
                            f"sits in front of it, so no INSIDEOUT_TRUSTED_PROXY_HOPS is right (default {hops}: "
                            "0 puts every user in nginx's one throttle bucket, 1 lets a client pick its own)")
        else:
            failures.append(f"docker-compose.yml: nginx writes X-Forwarded-For as {value!r}, which this guard cannot place; "
                            "an empty value stops nginx from starting")


def client_port_is_public():
    """True while the client service publishes a port on every interface, the way
    `"${CLIENT_PORT:-5173}:8080"` does (nginx listens on 8080 in the container)."""
    client = compose_service("client") or ""
    published = re.findall(r'^\s+- "?([^"\s#]*:8080)"?\s*(?:#.*)?$', client, re.M)
    return any(not re.match(r"(127\.0\.0\.1|\[::1\]):", port) for port in published)


def docs_state_the_rules(failures):
    hops = compose_hops_default()
    appends = any(effective(v, compose_service("client") or "") in ENDS_IN_PEER for v in nginx_xff_values())
    for name in DOCS:
        text = flat(name)
        if appends and not ADD_ONE.search(text):
            failures.append(f"{name}: does not say to add one to INSIDEOUT_TRUSTED_PROXY_HOPS for every further proxy in "
                            "front of nginx (a proxy that appends the client address moves the client one entry left)")
        if client_port_is_public() and not LOCK_DOWN.search(text):
            failures.append(f"{name}: does not say to bind the published client port to loopback or firewall it when a "
                            "proxy sits in front: a direct connection could write the entry the server counts")
        if compose_service("client") and RIGHT_AT_ZERO.search(text):
            failures.append(f"{name}: calls the default 0 right for the compose stack; nginx sits in front of the server "
                            f"there and compose defaults to {hops}")
    row = re.search(r"^\| `INSIDEOUT_TRUSTED_PROXY_HOPS` \|(.*)$", read("docs/usage/environment.md"), re.M)
    if row is None:
        failures.append("docs/usage/environment.md: no INSIDEOUT_TRUSTED_PROXY_HOPS row")
    elif hops != SERVER_DEFAULT and not re.search(rf"(?i)compose.*`{hops}`", row.group(1)):
        failures.append(f"docs/usage/environment.md: the INSIDEOUT_TRUSTED_PROXY_HOPS row does not say that compose defaults "
                        f"to `{hops}` while the server's own default is `{SERVER_DEFAULT}`")
    if re.search(r"^\s*#\s*INSIDEOUT_TRUSTED_PROXY_HOPS=(\d+)\s*$", read(".env.example"), re.M) is None:
        failures.append(".env.example: no commented `#INSIDEOUT_TRUSTED_PROXY_HOPS=N` default line")
    else:
        shown = int(re.search(r"^\s*#\s*INSIDEOUT_TRUSTED_PROXY_HOPS=(\d+)\s*$", read(".env.example"), re.M).group(1))
        if shown != hops:
            failures.append(f".env.example: the skeleton shows INSIDEOUT_TRUSTED_PROXY_HOPS={shown} as the default in effect "
                            f"but docker-compose.yml defaults it to {hops}")


def chain_is_described(failures):
    """Wherever the chain is documented it must name the variable; once nginx.conf
    takes the header from ${NGINX_XFF}, the architecture doc and the clientIP
    comment must name it too, or they keep describing a pass-through (F62)."""
    for name in ("docs/usage/deployment.md", "docs/usage/environment.md", "docs/architecture/backend.md"):
        if "INSIDEOUT_TRUSTED_PROXY_HOPS" not in read(name):
            failures.append(f"{name}: does not name INSIDEOUT_TRUSTED_PROXY_HOPS")
    if "${NGINX_XFF}" in nginx_xff_values():
        for name in ("docs/architecture/backend.md", "server/internal/api/client_ip.go"):
            if "NGINX_XFF" not in read(name):
                failures.append(f"{name}: describes the client-address chain without naming NGINX_XFF, the switch "
                                "between the hosted pass-through and the compose append")


CHECKS = [template_is_the_switch, hops_follow_nginx, docs_state_the_rules, chain_is_described]

if __name__ == "__main__":
    failures = []
    for check in CHECKS:
        check(failures)
    for message in failures:
        print(message, file=sys.stderr)
    if failures:
        sys.exit(1)
    print(f"proxy hops ok: {len(CHECKS)} checks")
