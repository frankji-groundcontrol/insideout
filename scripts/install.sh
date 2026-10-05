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
  echo "error: Go toolchain not found on PATH — install Go 1.22+ and re-run." >&2
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
echo "Done. Next steps:"
echo "  export INSIDEOUT_API=https://<your-domain>/api/v1    # or your local server"
echo "  export INSIDEOUT_TOKEN=\"\$(insideout login you@example.com)\""
echo "  insideout whoami"
echo "  scripts/mcp-config.sh    # emit the MCP registration for your agent"
