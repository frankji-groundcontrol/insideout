#!/usr/bin/env bash
# mcp-config.sh — emit the MCP registration JSON for insideout-mcp.
#
# Usage: scripts/mcp-config.sh [--api URL] [--token TOKEN] [--binary PATH] [-q]
#   --api     API base; default $INSIDEOUT_API, else the config file, else
#             the hosted API
#   --token   API token; default $INSIDEOUT_TOKEN, else the config file.
#             Omitted when unset so the MCP process can sign in with the
#             auth_* tools.
#   --binary  server command; default: insideout-mcp on PATH, else the
#             ~/.local/bin path the installer creates, else the bare name
#   -q        print only the JSON object (no placement hints)
#
# The config file is the 0600 config.json that `insideout setup` writes
# into the OS config dir. Precedence: flags > environment > config file.
# INSIDEOUT_CONFIG_FILE overrides the path (test affordance); the default
# mirrors os.UserConfigDir on purpose.
#
# The object uses the standard mcpServers shape (command + env) accepted by
# Claude Code, Cursor, ZCode, and most MCP clients — merge it wherever your
# agent reads MCP config. Pure stdout emitter; never writes any file.

set -euo pipefail

repo_root=$(cd "$(dirname "$0")" && pwd)
# shellcheck source=default-api.sh
. "$repo_root/default-api.sh"

api_flag=""
token_flag=""
api=""
token=""
binary=""
quiet=0

while [ $# -gt 0 ]; do
  case "$1" in
    --api)    api_flag=$2; shift 2 ;;
    --token)  token_flag=$2; shift 2 ;;
    --binary) binary=$2; shift 2 ;;
    -q|--quiet) quiet=1; shift ;;
    *) echo "usage: $0 [--api URL] [--token TOKEN] [--binary PATH] [-q]" >&2; exit 2 ;;
  esac
done

# The config file fills only what flags and environment left empty.
config_file=${INSIDEOUT_CONFIG_FILE:-}
if [ -z "$config_file" ]; then
  case "$(uname -s)" in
    Darwin) config_file="$HOME/Library/Application Support/insideout/config.json" ;;
    *)      config_file="${XDG_CONFIG_HOME:-$HOME/.config}/insideout/config.json" ;;
  esac
fi
cfg_api=""
cfg_token=""
if [ -f "$config_file" ] && command -v python3 >/dev/null 2>&1; then
  cfg_api=$(CONFIG_FILE="$config_file" python3 -c '
import json, os, sys
try:
    with open(os.environ["CONFIG_FILE"]) as f:
        cfg = json.load(f)
    v = cfg.get("api_base", "")
    print(v if isinstance(v, str) else "")
except Exception:
    pass
')
  cfg_token=$(CONFIG_FILE="$config_file" python3 -c '
import json, os, sys
try:
    with open(os.environ["CONFIG_FILE"]) as f:
        cfg = json.load(f)
    v = cfg.get("token", "")
    print(v if isinstance(v, str) else "")
except Exception:
    pass
')
fi
api=${api_flag:-${INSIDEOUT_API:-$(default_api_base)}}
[ -n "$cfg_api" ] && [ -z "$api_flag" ] && [ -z "${INSIDEOUT_API:-}" ] && api=$cfg_api
token=${token_flag:-${INSIDEOUT_TOKEN:-$cfg_token}}

if [ -z "$binary" ]; then
  if command -v insideout-mcp >/dev/null 2>&1; then
    binary=$(command -v insideout-mcp)
  elif [ -x "$HOME/.local/bin/insideout-mcp" ]; then
    binary=$HOME/.local/bin/insideout-mcp
  else
    binary=insideout-mcp
  fi
fi

# Minimal JSON string escaping: backslash then double quote. The values here
# (URLs, JWT-shaped tokens, paths) carry no control characters.
json_escape() {
  printf '%s' "$1" | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g'
}

token_json=""
if [ -n "$token" ]; then
  token_json=$(printf ',\n        "INSIDEOUT_TOKEN": "%s"' "$(json_escape "$token")")
fi

config=$(printf '{
  "mcpServers": {
    "insideout": {
      "command": "%s",
      "env": {
        "INSIDEOUT_API": "%s"%s
      }
    }
  }
}' "$(json_escape "$binary")" "$(json_escape "$api")" "$token_json")

if [ "$quiet" = 1 ]; then
  printf '%s\n' "$config"
  exit 0
fi

cat <<HINTS
Merge the "insideout" entry below into your agent's MCP config — the JSON
shape is the de-facto standard, so the same snippet ports across agents:

  Claude Code   project: .mcp.json in the repo root  (or: claude mcp add)
  Cursor        global: ~/.cursor/mcp.json  project: .cursor/mcp.json
  ZCode / other desktop agents: their mcpServers settings block

$config

HINTS
if [ -z "$token" ]; then
  echo "Note: no token in the snippet. Sign in with \`insideout setup\` or"
  echo "\`insideout login <email>\` (the config file then feeds this snippet),"
  echo "or use the MCP auth_* tools. Do not set INSIDEOUT_API unless this"
  echo "machine runs its own server (see docs/usage/installation.md)."
fi
