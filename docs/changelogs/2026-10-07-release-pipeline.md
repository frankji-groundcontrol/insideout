# 2026-10-07 — Release pipeline: prebuilt binaries + bootstrap one-liner

**Scope:** slice 4 of the setup TUI
([plan](../plans/2026-10-07-setup-tui.md)) — the piece that makes
the guided setup experience truly instantly installable: no
checkout, no Go toolchain. Built by dynamic workflow
`dwfrun-6e6ca8f5` (builder + five script gates + independent
review), joined and committed after the run.

## What changed

- **`.github/workflows/release.yml`** — on a pushed `v*` tag: build
  both binaries (`insideout`, `insideout-mcp`) for linux/amd64,
  linux/arm64, darwin/amd64, darwin/arm64 with `CGO_ENABLED=0`;
  one tarball per target (binaries at archive top level) +
  `checksums.txt` (sha256) + `bootstrap.sh` itself as a release
  asset; publish via the preinstalled `gh` CLI
  (`permissions: contents: write`), create-or-clobber. No
  third-party actions beyond `checkout@v4`/`setup-go@v5`, matching
  `ci.yml`'s style.
- **`scripts/bootstrap.sh`** — curl|bash-safe installer: OS/arch
  detection (macOS/Ubuntu, amd64/arm64; anything else → clear error
  pointing at building from source), downloads
  `insideout-<os>-<arch>.tar.gz` + `checksums.txt` from
  `${INSIDEOUT_RELEASE_BASE:-GitHub latest-download}` (the override
  also makes staged/file:// testing possible), verifies the sha256
  (shasum or sha256sum; mismatch → nonzero exit, nothing
  installed), installs both binaries 0755 exactly like `install.sh`
  (arg / `INSIDEOUT_INSTALL_DIR` / `~/.local/bin`, marked PATH
  rc-append, idempotent), and points at `insideout setup` as the
  front door.
- **Docs** teach the new path: installation.md (prebuilt section),
  cli-mcp-and-guidance.md, README (bilingual quick-install), the
  install skill's Step 3 (prebuilt alternative; `--local` still
  needs the checkout).

## Verification (workflow gates, all exit 0)

- `python3 yaml.safe_load` on release.yml (re-run after the join
  added the bootstrap.sh asset copy).
- Cross-compile matrix: 4 targets × 2 binaries, `CGO_ENABLED=0`.
- Staged end-to-end: darwin/arm64 tarball + checksums built,
  `bootstrap.sh` installed from a `file://` release base, installed
  `insideout setup --hosted` exited 0, `insideout-mcp` installed
  executable.
- Tamper rejection: appended bytes → bootstrap exits nonzero,
  installs nothing.
- `cd server && go test ./...` green.
- Independent review of both files (fresh context, first-tag-push
  and first-curl-user lenses, per-finding confirmation): **zero
  findings**.

## Operator notes

Nothing runs until the first `v*` tag is pushed — that first real
release (and a real HTTPS download) is the remaining user-owned
check. Evidence artifact: "Release pipeline evidence" on workflow
run `dwfrun-6e6ca8f5`.

*Superseded 2026-10-11: `v0.1.0` was tagged at the deployed commit and
published — assets built with the workflow's own commands and sha256-verified,
and the anonymous download verified once the repository went public
([owner-deploy record](2026-10-11-owner-deploy-and-mail-truth.md)).*
