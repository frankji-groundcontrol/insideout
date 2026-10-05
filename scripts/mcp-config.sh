#!/usr/bin/env bash
# mcp-config.sh — emit the MCP registration JSON for insideout-mcp.
#
# Usage: scripts/mcp-config.sh [--api URL] [--token TOKEN] [--binary PATH] [-q]
#   --api     API base; default $INSIDEOUT_API, else the CLI's local default
#   --token   API token; default $INSIDEOUT_TOKEN, else a paste-me placeholder
#   --binary  server command; default: insideout-mcp on PATH, else the
#             ~/.local/bin path the installer creates, else the bare name
#   -q        print only the JSON object (no placement hints)
#
# The object uses the standard mcpServers shape (command + env) accepted by
# Claude Code, Cursor, ZCode, and most MCP clients — merge it wherever your
# agent reads MCP config. Pure stdout emitter; never writes any file.

set -euo pipefail

api=${INSIDEOUT_API:-http://127.0.0.1:8080/api/v1}
token=${INSIDEOUT_TOKEN:-PASTE-YOUR-TOKEN}
binary=""
quiet=0

while [ $# -gt 0 ]; do
  case "$1" in
    --api)    api=$2;    shift 2 ;;
    --token)  token=$2;  shift 2 ;;
    --binary) binary=$2; shift 2 ;;
    -q|--quiet) quiet=1; shift ;;
    *) echo "usage: $0 [--api URL] [--token TOKEN] [--binary PATH] [-q]" >&2; exit 2 ;;
  esac
done

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

config=$(printf '{
  "mcpServers": {
    "insideout": {
      "command": "%s",
      "env": {
        "INSIDEOUT_API": "%s",
        "INSIDEOUT_TOKEN": "%s"
      }
    }
  }
}' "$(json_escape "$binary")" "$(json_escape "$api")" "$(json_escape "$token")")

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
if [ "$token" = "PASTE-YOUR-TOKEN" ]; then
  echo "Note: token is a placeholder — set INSIDEOUT_TOKEN or pass --token"
  echo "      (get one with: insideout login you@example.com)."
fi
