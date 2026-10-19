#!/usr/bin/env bash
# bootstrap.sh — install the insideout CLI + MCP server from GitHub Releases.
#
# curl | bash safe: needs no repository checkout. Downloads the release
# tarball for this platform, verifies it against checksums.txt, then
# installs both binaries the same way scripts/install.sh does.
#
# Supported systems: macOS and Linux, on arm64 or amd64. Windows is not
# supported: on any other OS the script stops before it downloads anything.
#
#   curl -fsSL <script-url> | bash
#   scripts/bootstrap.sh [install-dir]   # same argument as install.sh
#
#   install-dir    defaults to $INSIDEOUT_INSTALL_DIR, then ~/.local/bin
#   INSIDEOUT_RELEASE_BASE   overrides the download base URL (e.g. a
#                            mirror, or a specific release for testing). It
#                            must be https://; plain http is accepted only
#                            for a mirror on this machine (localhost,
#                            127.0.0.1 or [::1]). Redirects must stay https.

set -euo pipefail

repo=frankji-groundcontrol/insideout
base=${INSIDEOUT_RELEASE_BASE:-https://github.com/$repo/releases/latest/download}

# rv4-SEC-07: the base is where the binaries come from. https, or plain http to
# a mirror on this machine only, and "on this machine" is decided on the
# authority (everything before the first /): exactly localhost, 127.0.0.1 or
# [::1] with an optional all-digit port. A lookalike name (localhost.example)
# or user-info before an @ (http://localhost@example) is not loopback. Builtins
# only, so it holds before the first external command runs.
base_ok() {
  local host
  case $1 in
    https://*) return 0 ;;
    http://*)
      host=${1#http://}
      host=${host%%/*}
      case $host in
        localhost|127.0.0.1|'[::1]') return 0 ;;
        localhost:*|127.0.0.1:*|'[::1]:'*)
          case ${host##*:} in
            ''|*[!0-9]*) return 1 ;;
            *) return 0 ;;
          esac ;;
      esac ;;
  esac
  return 1
}
if ! base_ok "$base"; then
  echo "error: INSIDEOUT_RELEASE_BASE must be an https:// URL — nothing was downloaded." >&2
  echo "Plain http is accepted only for a mirror on this machine: http://localhost," >&2
  echo "http://127.0.0.1 or http://[::1], optionally with a port." >&2
  exit 1
fi

# Every request is https, and so is every redirect it meets: a mirror, or
# anything in the path, cannot step the download down to plain http.
fetch() { curl -fsSL --proto-redir '=https' -o "$1" "$2"; }

os=$(uname -s)
case "$os" in
  Darwin) os=darwin ;;
  Linux)  os=linux ;;
  *)
    echo "error: unsupported OS '$os' — this installer covers macOS and Linux only." >&2
    echo "Build from source instead: clone https://github.com/$repo and run scripts/install.sh." >&2
    exit 1
    ;;
esac

arch=$(uname -m)
case "$arch" in
  arm64|aarch64) arch=arm64 ;;
  x86_64|amd64)  arch=amd64 ;;
  *)
    echo "error: unsupported arch '$arch' — no release binary for $os/$arch." >&2
    echo "Build from source instead: clone https://github.com/$repo and run scripts/install.sh." >&2
    exit 1
    ;;
esac

if command -v shasum >/dev/null 2>&1; then
  sha256() { shasum -a 256 "$1" | awk '{print $1}'; }
elif command -v sha256sum >/dev/null 2>&1; then
  sha256() { sha256sum "$1" | awk '{print $1}'; }
else
  echo "error: neither shasum nor sha256sum found — cannot verify the download." >&2
  echo "Install one of them, or build from source via scripts/install.sh in a checkout." >&2
  exit 1
fi

workdir=$(mktemp -d)
trap 'rm -rf "$workdir"' EXIT

tarball="insideout-$os-$arch.tar.gz"
echo "==> Downloading $tarball and checksums.txt"
fetch "$workdir/$tarball" "$base/$tarball"
fetch "$workdir/checksums.txt" "$base/checksums.txt"

# Match the line for this tarball; strip any leading path (sha256sum
# writes plain names here, but "./name" layouts verify the same way).
expected=$(awk -v f="$tarball" '
  { name = $2; sub(/^.*\//, "", name) }
  name == f { print $1; exit }
' "$workdir/checksums.txt")
if [ -z "$expected" ]; then
  echo "error: checksums.txt has no line for $tarball — refusing to install." >&2
  exit 1
fi

actual=$(sha256 "$workdir/$tarball")
if [ "$actual" != "$expected" ]; then
  echo "error: checksum mismatch for $tarball — nothing was installed." >&2
  echo "  expected $expected" >&2
  echo "  actual   $actual" >&2
  exit 1
fi
echo "==> Checksum verified"

tar -xzf "$workdir/$tarball" -C "$workdir"

# Install-dir resolution mirrors scripts/install.sh exactly.
install_dir=${1:-${INSIDEOUT_INSTALL_DIR:-$HOME/.local/bin}}
mkdir -p "$install_dir"
install_dir=$(cd "$install_dir" && pwd)

echo "==> Installing into $install_dir"
install -m 0755 "$workdir/insideout" "$install_dir/insideout"
install -m 0755 "$workdir/insideout-mcp" "$install_dir/insideout-mcp"

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

cat <<EOF

Done. insideout + insideout-mcp installed from the $os/$arch release.

'insideout setup' is the front door — run it with no flags for the
interactive flow, or go straight there:

  insideout setup                                  # interactive
  insideout setup --hosted                         # use the hosted API
  insideout setup --check                          # verify the current setup

No account yet? Signing up on the hosted service takes an invitation code:
ask the server's administrator for one, then

  insideout register --invite <invitation-code> you@example.com

Docs: docs/usage/installation.md in the $repo repository.
Build from source instead: scripts/install.sh in a repository checkout.
EOF
