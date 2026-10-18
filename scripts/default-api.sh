#!/usr/bin/env bash
# Print the hosted API base compiled into the CLI and MCP server.
# Source this file to call default_api_base, or execute it to print the URL.
default_api_base() {
  local here repo line
  here=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
  repo=$(cd "$here/.." && pwd)
  line=$(sed -n 's/^[a-z]* DefaultBase = "\(.*\)"$/\1/p' "$repo/server/internal/apiclient/default_base.go")
  if [ -z "$line" ]; then
    echo "error: DefaultBase missing from server/internal/apiclient/default_base.go" >&2
    return 1
  fi
  printf '%s\n' "$line"
}

if [ "${BASH_SOURCE[0]}" = "$0" ]; then
  default_api_base
fi
