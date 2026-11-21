#!/usr/bin/env bash
# setup.sh — a shim for `insideout setup`, the one setup engine.
#
# Every flag and stdin pass through unchanged, run from this checkout:
#
#   scripts/setup.sh                         # the wizard, when stdin is a TTY
#   scripts/setup.sh --hosted                # confirm the hosted path
#   scripts/setup.sh --local --postgres bundled [--no-start]
#   scripts/setup.sh --local --postgres own  # URL on stdin, not echoed
#
# It runs the installed `insideout`, else `go run ./cmd/insideout` in
# server/. `insideout setup --help` lists the flags. See
# docs/usage/installation.md.

set -euo pipefail

repo_root=$(cd "$(dirname "$0")/.." && pwd)
cd "$repo_root"
if command -v insideout >/dev/null 2>&1; then
  exec insideout setup "$@"
fi
if command -v go >/dev/null 2>&1; then
  cd server
  exec go run ./cmd/insideout setup "$@"
fi
echo "error: neither insideout nor go is on PATH. Install the CLI with scripts/install.sh," >&2
echo "then run: insideout setup --local --postgres bundled|own" >&2
exit 127
