# Modules — install steps per platform

One section per required module: what it is for, which destinations
need it (A hosted · B own server · C full development — see
[SKILL.md](../SKILL.md#step-0--platform-and-destination)), exact
macOS and Ubuntu steps, and the verify command that ends the module.
**Check first**: run the verify command; skip the module when it
already passes. Every module must end green before the next one.

Versions that matter (this repo's own pins — do not guess):

- **Go**: read the floor from `server/go.mod` at install time (its `go`
  directive is the minimum; newer releases are fine). Never assume a
  version — the directive moves.
- **Flutter**: 3.44.x — CI and `client/Dockerfile` pin **3.44.0**.
- **PostgreSQL**: 18 for a server you bring yourself (older majors are
  untested); the bundled Docker path runs the `postgres:18` image.
- **Docker**: Engine + Compose **v2** (`docker compose`, not the old
  `docker-compose` binary).

## 1. Base tools

Needed by: A, B, C. `git` (clone + version control), `curl` (the
smoke test and downloads), `jq` (smoke test), `python3` (the env
scripts `env.sh`/`setup.sh` write `.env` through it, and the key-store
tool runs on it), `openssl` (`setup.sh` generates the JWT secret with
`openssl rand -hex 32`).

**macOS**

```bash
xcode-select --install        # git + toolchain; click through the dialog
brew install python3 jq       # curl and openssl ship with macOS
```

**Ubuntu**

```bash
sudo apt-get update
sudo apt-get install -y git curl jq python3 openssl
```

Verify: `git --version && curl --version | head -1 && jq --version &&
python3 --version && openssl version`. (macOS `openssl` is LibreSSL —
it supports `rand -hex`, which is all the scripts use.)

## 2. Homebrew (macOS only)

Needed by: A, B, C on macOS. The package manager the steps below use.
Ubuntu uses `apt` — skip this module.

```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

Then do what the installer prints at the end (usually adding
`eval "$(/homebrew/bin/brew shellenv)"` to `~/.zprofile` on Apple
silicon) — without it `brew` is not found in new shells.

Verify: `brew --version`.

## 3. Go

Needed by: A, B, C — both binaries (`insideout`, `insideout-mcp`) are
built from `server/` with it. The minimum is the `go` directive in
`server/go.mod` — read it now and use it below; do not assume a version.

**macOS**

```bash
brew install go
```

**Ubuntu**

`apt`'s `golang-go` is frequently older than the `go.mod` floor (1.22
on 24.04 LTS). Check it, and only use apt when it is new enough;
otherwise install the official tarball:

```bash
apt policy golang-go         # use apt only if it offers >= the go.mod floor
GO_VERSION=1.27.2            # pinned together with the two sha256 values below
case "$(uname -m)" in
  x86_64) GO_ARCH=amd64; GO_SHA256=ecbadb99091a3f46e31f5f934b068b1864eafa7995211b39eaddf76996045fe5 ;;
  aarch64|arm64) GO_ARCH=arm64; GO_SHA256=94f3e30b8e374bc285e7dadc11e0865726b9bc6e85b841ccceaabc0214c6b7c8 ;;
  *) echo "unsupported arch"; exit 1 ;;
esac
curl -fsSL -o /tmp/go.tgz "https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
echo "${GO_SHA256}  /tmp/go.tgz" | sha256sum -c - || { echo "go.tgz does not match the pinned sha256: not installing" >&2; exit 1; }
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf /tmp/go.tgz
echo 'export PATH=/usr/local/go/bin:$PATH' >> ~/.bashrc    # or ~/.profile / ~/.zshrc
export PATH=/usr/local/go/bin:$PATH
```

The tarball is unpacked as root over `/usr/local/go`, so it is verified first:
`sha256sum -c` runs before `rm -rf`, and a mismatch stops the block with the
existing toolchain untouched (`exit 1` rather than `&&`: a script run under
`set -e` does not stop on a failed command in the middle of an `&&` list). To
install a newer release, change `GO_VERSION` and both hashes together — take
them from the sha256 column of <https://go.dev/dl/> — and keep the version at
or above the `go.mod` floor (`scripts/tests/check_go_floor.py` enforces it).

The directory goes **first** on `PATH`: an older `go` already on the machine
(the apt `golang-go` 1.22, a preinstalled toolchain) would otherwise win and
the verify step below would print the stale version. The Ubuntu CI job caught
exactly that.

Verify: `go version` — must print the `go` directive from
`server/go.mod` or newer. (On macOS a
stale `go` from an old install wins the PATH conflict silently; the
verify step is not optional.)

## 4. Docker + Compose v2

Needed by: B (bundled-Postgres path), C. Runs `docker compose up -d
postgres` (`postgres:18` on `127.0.0.1:5442`) and the full-compose
topology. **Not needed** when the human brings their own Postgres URL.

**macOS**

```bash
brew install --cask docker
open -a Docker                # first launch accepts the license; wait for the whale
```

**Ubuntu**

```bash
sudo apt-get install -y docker.io docker-compose-v2   # Ubuntu 24.04; if the name
                              # differs, use docs.docker.com's apt repository setup
sudo usermod -aG docker "$USER"
```

The group change needs a **new session** (`newgrp docker` or a fresh
login) to take effect.

Verify: `docker version && docker compose version` — Compose must
report v2.x. Then prove the daemon actually runs containers:
`docker run --rm hello-world`.

## 5. PostgreSQL 18 (own-instance path only)

Needed by: B only when the human chose *their own* Postgres instead of
the bundled Docker one. On a shared multi-tenant instance the app role
owns only the `insideout` schema — the role/database creation rules
live in
[installation.md → Database choices](../../../docs/usage/installation.md);
follow them, then hand the resulting URL to `setup.sh --local
--postgres own`.

**macOS**

```bash
brew install postgresql@18
brew services start postgresql@18
echo 'export PATH="/opt/homebrew/opt/postgresql@18/bin:$PATH"' >> ~/.zshrc
```

**Ubuntu**

```bash
sudo apt-get install -y postgresql
sudo systemctl enable --now postgresql
```

Verify: `psql --version` reads 18 (older majors are untested; if the Ubuntu
package is older, use the bundled Docker path or the PostgreSQL project's apt
repository); a cluster is running (`pg_lsclusters` on Ubuntu,
`brew services list` on macOS).

## 6. Flutter 3.44.x + Chrome

Needed by: C only. Pins matter — use **3.44.x**; CI pins 3.44.0.

**Both platforms — pinned archive (recommended)**

Download from the [release archive](https://docs.flutter.dev/release/archive)
(the direct forms below were verified live). The archive is checked against
the sha256 pinned here **before** it is unpacked; on a mismatch the `&&` chain
stops (non-zero exit), nothing is extracted and your shell rc file is left
alone:

```bash
# macOS Apple silicon. Another target: swap the URL and the hash on the lines below —
#   macOS Intel  .../stable/macos/flutter_macos_3.44.0-stable.zip
#                2e9efb651b333b201487c2ac213f910c717338483ca24cbf497a067b05f048e5
#   Ubuntu x64   .../stable/linux/flutter_linux_3.44.0-stable.tar.xz
#                e1ec95e6c550458a34de93580cb85dac24da0e9bedb9bb42811f050ac5a0c7d5
#                (check with sha256sum -c, unpack with tar -xJf instead of unzip -q,
#                 and append the PATH line to ~/.bashrc instead of ~/.zshrc)
FLUTTER_SHA256=0d4d1d3f379d18aff392291be6871d63bf090a9e81078d4c4bc84e6208603dca
curl -fsSL -o /tmp/flutter.zip \
  "https://storage.googleapis.com/flutter_infra_release/releases/stable/macos/flutter_macos_arm64_3.44.0-stable.zip"
echo "$FLUTTER_SHA256  /tmp/flutter.zip" | shasum -a 256 -c - \
  && mkdir -p ~/development && cd ~/development && unzip -q /tmp/flutter.zip \
  && echo 'export PATH="$PATH:$HOME/development/flutter/bin"' >> ~/.zshrc
```

For another Flutter release, change the version in the URL and the hash
together; the hashes are in the release feeds
(`.../flutter_infra_release/releases/releases_{linux,macos}.json`, field
`sha256` of the `archive` you download). CI compares them with this file.

**Shortcut installs** — `brew install --cask flutter` (macOS) or
`snap install flutter --classic` (Ubuntu) install the *latest*
stable: acceptable only if `flutter --version` afterwards reports
3.44.x; otherwise fall back to the pinned archive.

**Chrome** (web target): `brew install --cask google-chrome` (macOS);
on Ubuntu, Google's [chrome deb](https://www.google.com/chrome/) or
`snap install chromium`.

**Optional targets** — say they are optional and move on unless the
user asks: iOS needs full Xcode (macOS only, App Store) plus its
command-line components; Android needs Android Studio + SDK. This
repo's own record shows web is the primary development target and a
machine without the Android SDK is a known, accepted state.

Verify: `flutter --version` (3.44.x), then `flutter doctor` — resolve
what it flags for the targets the user actually chose (a ✗ under
Android/iOS tooling is fine for web-only), and `flutter devices`
lists Chrome.

## 7. InsideOut's own components

Not system packages — these are the repo's scripts, run from the repo
root after the modules above are green. Full commands and the
path-specific decisions are [SKILL.md steps 2–5](../SKILL.md#step-2--get-the-repository);
the module view:

| Component | Installed by | Result |
| --- | --- | --- |
| CLI + MCP binaries | `./scripts/install.sh` | `insideout`, `insideout-mcp` in `~/.local/bin`, PATH appended once |
| `.env` | `./scripts/env.sh init` or `setup.sh` | the only file holding real values; never printed, never committed |
| Host key store | `python3 scripts/keys.py init` + `fill` | `hosts/<host>/keys/` — one `.key` per mapped kind, never absent |
| Database schema | `./scripts/dev.sh -C server go run ./cmd/insideout migrate` (bundled path runs it via `setup.sh`) | embedded SQL in `server/db/migrations/`, inside the `insideout` schema only |
| MCP registration | the user's agent MCP config (the `insideout-mcp` entry from Step 4 of the skill) | standard `mcpServers` entry; the agent lists the `insideout` tools after a reload |

Verify: `insideout whoami` — `not signed in` is the
finished-install signal on the hosted path; on a local server,
`curl -s http://127.0.0.1:8080/healthz` answers once the server runs.
