#!/usr/bin/env bash
# setup.sh — guided server setup after scripts/install.sh.
#
# The installed CLI already calls the hosted API. This script is the
# other path: a PostgreSQL-backed server on this machine.
#
#   scripts/setup.sh                         # ask, when stdin is a TTY
#   scripts/setup.sh --hosted                # confirm the hosted path
#   scripts/setup.sh --local --postgres bundled
#   scripts/setup.sh --local --postgres own  # URL on stdin, not echoed
#   scripts/setup.sh --local --postgres bundled --no-start
#
# Values written into .env are never printed. An agent should ask the
# human which path they want, then pass the flags. See
# docs/usage/installation.md.

set -euo pipefail

repo_root=$(cd "$(dirname "$0")/.." && pwd)
env_file=${INSIDEOUT_ENV_FILE:-$repo_root/.env}
example=$repo_root/.env.example
mode=""
postgres=""
no_start=0

while [ $# -gt 0 ]; do
  case "$1" in
    --hosted) mode=hosted; shift ;;
    --local) mode=local; shift ;;
    --postgres) postgres=$2; shift 2 ;;
    --no-start) no_start=1; shift ;;
    -h|--help)
      sed -n '2,16p' "$0" | sed 's/^# \{0,1\}//'
      exit 0
      ;;
    *) echo "usage: $0 [--hosted | --local --postgres bundled|own] [--no-start]" >&2; exit 2 ;;
  esac
done

# shellcheck source=default-api.sh
. "$repo_root/scripts/default-api.sh"

hosted_blurb() {
  local api
  api=$(default_api_base)
  cat <<EOF
Hosted path. No environment variables are required.

  $api

Sign in (the password is read from the terminal; the login is saved for the
CLI and the MCP server):

  insideout login you@example.com
  insideout whoami

No account yet? Signing up on the hosted service takes an invitation code:
ask the server's administrator for one, then

  insideout register --invite <invitation-code> you@example.com

Headless or CI: set INSIDEOUT_TOKEN to an access token instead of signing in.
It wins over the saved login and is never renewed.

A local server is a different API. Set INSIDEOUT_API=http://127.0.0.1:8080/api/v1
only after ./scripts/setup.sh --local has finished.
Guide: docs/usage/installation.md
EOF
}

ask() {
  local prompt=$1 default=$2 reply
  printf '%s [%s] ' "$prompt" "$default"
  read -r reply
  printf '%s' "${reply:-$default}"
}

if [ -z "$mode" ]; then
  if [ ! -t 0 ]; then
    echo "error: no TTY. Pass --hosted or --local --postgres bundled|own." >&2
    echo "Guide: docs/usage/installation.md" >&2
    exit 2
  fi
  echo "1) Use the hosted service (no environment variables)"
  echo "2) Run a server on this machine (PostgreSQL)"
  choice=$(ask "Choice" "1")
  case "$choice" in
    1) mode=hosted ;;
    2) mode=local ;;
    *) echo "error: choose 1 or 2" >&2; exit 2 ;;
  esac
fi

if [ "$mode" = "hosted" ]; then
  hosted_blurb
  exit 0
fi

if [ "$mode" != "local" ]; then
  echo "error: unknown mode $mode" >&2
  exit 2
fi

if [ -z "$postgres" ]; then
  if [ ! -t 0 ]; then
    echo "error: --local needs --postgres bundled|own" >&2
    exit 2
  fi
  echo "1) Bundled Postgres in Docker (localhost port 5442)"
  echo "2) A Postgres URL I already have"
  choice=$(ask "Database" "1")
  case "$choice" in
    1) postgres=bundled ;;
    2) postgres=own ;;
    *) echo "error: choose 1 or 2" >&2; exit 2 ;;
  esac
fi

case "$postgres" in
  bundled|own) ;;
  *) echo "error: --postgres must be bundled or own" >&2; exit 2 ;;
esac

if [ ! -f "$env_file" ]; then
  cp "$example" "$env_file"
  chmod 600 "$env_file"
  echo "==> Created $env_file from the skeleton (values not printed)"
fi

set_env() {
  # $1 key, $2 value. The value is not printed.
  ENV_FILE=$env_file ENV_KEY=$1 ENV_VALUE=$2 python3 - <<'PY'
import os
path, key, value = os.environ["ENV_FILE"], os.environ["ENV_KEY"], os.environ["ENV_VALUE"]
prefix = key + "="
lines = open(path).read().splitlines(True)
found = False
out = []
for line in lines:
    body = line.lstrip()
    if body.startswith(prefix):
        nl = "\n" if line.endswith("\n") else ""
        out.append(prefix + value + nl)
        found = True
    else:
        out.append(line)
if not found:
    if out and not out[-1].endswith("\n"):
        out[-1] += "\n"
    out.append(prefix + value + "\n")
open(path, "w").writelines(out)
os.chmod(path, 0o600)
PY
}

read_env() {
  # Print one value, for the script's own use. Callers must not echo it.
  ENV_FILE=$env_file ENV_KEY=$1 python3 - <<'PY'
import os
path, key = os.environ["ENV_FILE"], os.environ["ENV_KEY"]
prefix = key + "="
for line in open(path):
    body = line.strip()
    if body.startswith(prefix):
        print(body[len(prefix):], end="")
        break
PY
}

jwt=$(read_env INSIDEOUT_JWT_SECRET || true)
if [ -z "$jwt" ] || [ "$jwt" = "change_me_to_a_long_random_string" ] || [ "${#jwt}" -lt 32 ]; then
  set_env INSIDEOUT_JWT_SECRET "$(openssl rand -hex 32)"
  echo "==> Wrote INSIDEOUT_JWT_SECRET (generated, not printed)"
fi

if [ "$postgres" = "own" ]; then
  if [ -t 0 ]; then
    printf 'Paste DATABASE_URL (input hidden): '
    read -r -s database_url
    printf '\n'
  else
    database_url=$(cat)
  fi
  if [ -z "$database_url" ]; then
    echo "error: empty DATABASE_URL" >&2
    exit 2
  fi
  set_env DATABASE_URL "$database_url"
  echo "==> Wrote DATABASE_URL (not printed)"
else
  port=$(read_env POSTGRES_PORT || true)
  port=${port:-5442}
  password=$(read_env POSTGRES_APP_PASSWORD || true)
  password=${password:-insideout_app_dev_password}
  set_env DATABASE_URL "postgres://insideout_app:${password}@127.0.0.1:${port}/insideout?sslmode=disable"
  echo "==> Wrote DATABASE_URL for bundled Postgres on 127.0.0.1:${port}"
  # insideout_app cannot apply migrations: pair it with the owner role the
  # postgres image's init script creates (same host, port and database).
  owner_password=$(read_env POSTGRES_OWNER_PASSWORD || true)
  owner_password=${owner_password:-insideout_owner_dev_password}
  set_env DATABASE_OWNER_URL "postgres://insideout_owner:${owner_password}@127.0.0.1:${port}/insideout?sslmode=disable"
  echo "==> Wrote DATABASE_OWNER_URL for bundled Postgres"
fi

if [ "$no_start" -eq 1 ]; then
  echo "==> --no-start: .env updated, database not started, migrations not applied"
  exit 0
fi

if [ "$postgres" = "bundled" ]; then
  if ! command -v docker >/dev/null 2>&1; then
    echo "error: docker is required for bundled Postgres" >&2
    exit 1
  fi
  echo "==> Starting bundled Postgres"
  (cd "$repo_root" && docker compose up -d postgres)
fi

echo "==> Applying migrations"
(cd "$repo_root" && ./scripts/dev.sh -C server go run ./cmd/insideout migrate)

cat <<'EOF'

Local server files are ready. The installed CLI still calls the hosted
API until you point it at this machine:

  export INSIDEOUT_API=http://127.0.0.1:8080/api/v1

Start the API:

  ./scripts/dev.sh -C server go run ./cmd/insideout

First admin, then invites (registration is invitation-only):

  ./scripts/dev.sh -C server go run ./cmd/insideout promote-admin you@example.com

Guide: docs/usage/installation.md
EOF
