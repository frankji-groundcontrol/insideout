#!/usr/bin/env bash
# install.sh — build and install the insideout CLI + MCP server.
#
# Usage: scripts/install.sh [install-dir]
#   install-dir    defaults to $INSIDEOUT_INSTALL_DIR, then ~/.local/bin
#
# Builds server/cmd/insideout and server/cmd/insideout-mcp with the local Go
# toolchain, installs both into <install-dir>, and — when that directory is
# not already on PATH — appends one marked export line to the login shell's
# rc file (~/.zshrc for zsh, ~/.bashrc for bash, ~/.profile otherwise).
# Idempotent: safe to re-run; the rc append is skipped when already present.

set -euo pipefail

repo_root=$(cd "$(dirname "$0")/.." && pwd)
install_dir=${1:-${INSIDEOUT_INSTALL_DIR:-$HOME/.local/bin}}
# Normalise to an absolute path so the rc export line stays valid anywhere.
mkdir -p "$install_dir"
install_dir=$(cd "$install_dir" && pwd)

if ! command -v go >/dev/null 2>&1; then
  echo "error: Go toolchain not found on PATH — install Go 1.26+ and re-run." >&2
  exit 1
fi

echo "==> Building insideout + insideout-mcp (Go $(go version | awk '{print $3}'))"
build_dir=$(mktemp -d)
trap 'rm -rf "$build_dir"' EXIT
(cd "$repo_root/server" && go build -o "$build_dir/insideout" ./cmd/insideout)
(cd "$repo_root/server" && go build -o "$build_dir/insideout-mcp" ./cmd/insideout-mcp)

echo "==> Installing into $install_dir"
install -m 0755 "$build_dir/insideout" "$install_dir/insideout"
install -m 0755 "$build_dir/insideout-mcp" "$install_dir/insideout-mcp"

on_path() {
  case ":$PATH:" in
    *":$1:"*) return 0 ;;
    *)        return 1 ;;
  esac
}

if on_path "$install_dir"; then
  echo "==> $install_dir is already on PATH"
else
  shell_bin=${SHELL:-/bin/sh}
  shell_name=$(basename "$shell_bin")
  case "$shell_name" in
    zsh)  rc_file=$HOME/.zshrc ;;
    bash) rc_file=$HOME/.bashrc ;;
    *)    rc_file=$HOME/.profile ;;
  esac
  marker="# added by insideout installer"
  if [ -f "$rc_file" ] && grep -Fq "$marker" "$rc_file"; then
    echo "==> PATH export already present in $rc_file (marker found)"
  else
    printf '\n%s\nexport PATH="%s:$PATH"\n' "$marker" "$install_dir" >> "$rc_file"
    echo "==> Appended PATH export to $rc_file — start a new shell, or run:"
    echo "      export PATH=\"$install_dir:\$PATH\""
  fi
fi

echo
api_base=$("$repo_root/scripts/default-api.sh")
echo "Done. The CLI and MCP server call the hosted API with no environment variables:"
echo "  $api_base"
echo
echo "Sign in (password is read from the terminal; only the token is printed):"
echo "  export INSIDEOUT_TOKEN=\"\$(insideout login you@example.com)\""
echo "  insideout whoami"
echo
echo "Own server, including PostgreSQL: ./scripts/setup.sh"
echo "Full guide: docs/usage/installation.md"
echo "MCP registration: ./scripts/mcp-config.sh"
if [ -t 0 ]; then
  printf 'Set up your own server on this machine now? [y/N] '
  read -r answer
  case "$answer" in
    y|Y|yes|YES) exec "$repo_root/scripts/setup.sh" --local ;;
  esac
fi
