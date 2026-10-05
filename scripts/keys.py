#!/usr/bin/env python3
"""Audit and fill the module-grouped key store under hosts/<host>/keys/.

Each functional module (auth, database, github, ...) holds one or more
tracked YAML manifests whose `secrets:` maps name every key —
`<name>: {var: ENV_VAR, kind: value|path}` — plus the module's shared
git-ignored .secrets/<name>.key per entry (mode 0600). This tool never
prints values — audit reports filled/empty, fill copies env-file values
into EMPTY .key files only.

Usage (from the repo root):
  python3 scripts/keys.py audit [env-file]
  python3 scripts/keys.py fill <env-file> [--dry-run]

INSIDEOUT_KEYS_HOST=<name> picks the host when more than one exists;
with exactly one host folder that host is the default. Exit 1 from audit
when every mapped env var is also present-and-set in the given env file is
NOT an error condition — audit only reports.
"""

from __future__ import annotations

import argparse
import glob
import os
import re
import stat
import sys

# Bracket tokens are skeletons only without an @ (cf. <password>);
# display-name senders (Name <addr@host>) are real values.
PLACEHOLDER = re.compile(r"changeme|<[^@<>]*>|your[-_ ]|placeholder|xxx|^$", re.I)


def repo_root() -> str:
    here = os.path.dirname(os.path.abspath(__file__))
    return os.path.dirname(here)


def discover_hosts(root: str) -> list[str]:
    out = []
    for p in sorted(glob.glob(os.path.join(root, "hosts", "*", "host.yml"))):
        out.append(os.path.basename(os.path.dirname(p)))
    return out


def pick_host(root: str, wanted: str | None) -> str:
    hosts = discover_hosts(root)
    if wanted:
        if wanted not in hosts:
            raise SystemExit(f"unknown host {wanted!r}; known: {', '.join(hosts)}")
        return wanted
    if len(hosts) == 1:
        return hosts[0]
    raise SystemExit(f"several hosts ({', '.join(hosts)}) — set INSIDEOUT_KEYS_HOST")


def key_map(root: str, host: str) -> dict[str, tuple[str, str]]:
    """env_var -> (module/name, key path) from every manifest (no yaml dep).

    Walks hosts/<host>/keys/*/*.yaml — one or more manifests per
    functional module — reads each `secrets:` entry (block or flow
    style) for its var/kind, and resolves the value file to the module's
    shared .secrets/<name>.key.
    """
    m: dict[str, tuple[str, str]] = {}
    pattern = os.path.join(root, "hosts", host, "keys", "*", "*.yaml")
    for y in sorted(glob.glob(pattern)):
        module = os.path.basename(os.path.dirname(y))
        secrets_dir = os.path.join(os.path.dirname(y), ".secrets")
        for name, fields in _parse_secrets_block(y).items():
            var = fields.get("var")
            if not var:
                continue
            m[var] = (f"{module}/{name}", os.path.join(secrets_dir, f"{name}.key"))
    return m


def _parse_secrets_block(path: str) -> dict[str, dict[str, str]]:
    """Parse the `secrets:` map of a group manifest: name -> {var, kind, ...}."""
    entries: dict[str, dict[str, str]] = {}
    current: str | None = None
    in_secrets = False
    with open(path, encoding="utf-8") as fh:
        for line in fh:
            if re.match(r"^ *#", line) or not line.strip():
                continue
            if not line.startswith(" ") and line.rstrip().endswith(":") and line.split(":", 1)[0] == "secrets":
                in_secrets = True
                continue
            if not line.startswith(" "):
                in_secrets = False
                current = None
                continue
            if not in_secrets:
                continue
            indent = len(line) - len(line.lstrip())
            flow = re.match(r"^\s{2}([\w.-]+):\s*\{(.+)\}\s*$", line)
            if flow:
                current = flow.group(1)
                entries[current] = {
                    k: v.strip().strip("'\"")
                    for k, v in re.findall(r"(\w+):\s*([^,}]+)", flow.group(2))
                }
                continue
            named = re.match(r"^\s{2}([\w.-]+):\s*$", line)
            if named:
                current = named.group(1)
                entries[current] = {}
                continue
            field = re.match(r"^\s{4}(\w+):\s*(.+?)\s*$", line)
            if field and current:
                entries[current][field.group(1)] = field.group(2).strip("'\"")
    return entries


def parse_env(path: str) -> dict[str, str]:
    out: dict[str, str] = {}
    with open(path, encoding="utf-8", errors="replace") as fh:
        for line in fh:
            if line.lstrip().startswith("#"):
                continue
            m = re.match(r"\s*(?:export\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.*?)\s*$", line)
            if m:
                out[m.group(1)] = m.group(2).strip("'\"")
    return out


def env_state(value: str | None) -> str:
    if value is None:
        return "missing in env"
    return "placeholder" if PLACEHOLDER.search(value) else "set"


def audit(root: str, host: str, envfile: str | None) -> int:
    env = parse_env(envfile) if envfile else {}
    rows = 0
    for var, (slug, path) in sorted(key_map(root, host).items()):
        filled = os.path.exists(path) and os.path.getsize(path) > 0
        state = "filled" if filled else ("empty" if os.path.exists(path) else "missing")
        note = env_state(env.get(var)) if envfile else "-"
        print(f"  {var:32} key={state:8} env={note:16} {os.path.relpath(path, root)}")
        rows += 1
    if envfile:
        km = key_map(root, host)
        unmapped = sorted(
            k for k, v in env.items()
            if k not in km and env_state(v) == "set" and re.search(r"PASSWORD|KEY|TOKEN|SECRET|DSN", k)
        )
        suffix = f": {', '.join(unmapped)}" if unmapped else ""
        print(f"set in env but not mapped to a host key: {len(unmapped)}{suffix}")
    print(f"{rows} key kinds audited (values never printed)")
    return 0


def init_cmd(root: str, host: str) -> int:
    """Create an empty `.key` placeholder for every mapped kind — the
    never-absent rule: a kind may be blank (fallback in effect) but
    never missing. Never touches a non-empty file."""
    created = kept = 0
    for var, (_slug, path) in sorted(key_map(root, host).items()):
        if os.path.exists(path):
            kept += 1
            action = "exists"
        else:
            os.makedirs(os.path.dirname(path), mode=0o700, exist_ok=True)
            fd = os.open(path, os.O_WRONLY | os.O_CREAT, 0o600)
            os.fdopen(fd, "w").close()
            os.chmod(path, 0o600)
            created += 1
            action = "created (empty — fallback in effect)"
        print(f"  {var:32} {action:38} {os.path.relpath(path, root)}")
    print(f"{created} created, {kept} exist (values never printed)")
    return 0


def fill(root: str, host: str, envfile: str, dry_run: bool) -> int:
    env = parse_env(envfile)
    written = kept = skipped = 0
    for var, (slug, path) in sorted(key_map(root, host).items()):
        filled = os.path.exists(path) and os.path.getsize(path) > 0
        v = env.get(var)
        if filled:
            kept += 1
            action = "keep (already filled)"
        elif env_state(v) == "set":
            action = "write" if not dry_run else "would write"
            if not dry_run:
                os.makedirs(os.path.dirname(path), mode=0o700, exist_ok=True)
                fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_TRUNC, 0o600)
                with os.fdopen(fd, "w") as fh:
                    fh.write(v)
                os.chmod(path, 0o600)
            written += 1
        else:
            skipped += 1
            action = f"skip (env {env_state(v)})"
        print(f"  {var:32} {action:22} {os.path.relpath(path, root)}")
    print(f"{written} written, {kept} kept, {skipped} skipped (values never printed)")
    return 0


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("command", choices=["audit", "fill", "init"])
    ap.add_argument("envfile", nargs="?", default=".env")
    ap.add_argument("--dry-run", action="store_true")
    ap.add_argument("--root", default=repo_root())
    a = ap.parse_args()
    host = pick_host(a.root, os.environ.get("INSIDEOUT_KEYS_HOST"))
    if a.command == "audit":
        return audit(a.root, host, a.envfile if os.path.exists(a.envfile) else None)
    if a.command == "init":
        return init_cmd(a.root, host)
    if not os.path.exists(a.envfile):
        raise SystemExit(f"env file not found: {a.envfile}")
    return fill(a.root, host, a.envfile, a.dry_run)


if __name__ == "__main__":
    sys.exit(main())
