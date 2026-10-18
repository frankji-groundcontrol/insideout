---
name: insideout-setup
description: Guide a beginner and their coding agent through a from-scratch InsideOut installation on macOS or Ubuntu — install every required module (base tools, Go, Flutter, Docker/PostgreSQL), choose hosted vs own server, configure .env and the host key store, install the CLI/MCP binaries, verify, and write the installation record. Use when a user asks to set up, install, or onboard InsideOut on a new machine or new environment, or when insideout/go/flutter commands fail with missing-toolchain or missing-environment errors. Other platforms (Windows, containers without Docker) are out of scope — say so and stop.
---

# InsideOut setup — bare machine to verified install

You are the installing agent. Your user is likely a beginner: explain
each step in one line before running it, never run a command they
cannot see the purpose of, and never leave them holding a secret.

Supported platforms: **macOS** (Homebrew) and **Ubuntu** (apt). On
anything else — Windows, WSL untested by this repo, Alpine, NixOS —
stop, say it is unsupported, and offer the manual docs instead.

## Non-negotiables (apply at every step)

- **Never read `.env` or `.env.local`, never print a secret.** Values
  go into `.env` via the scripts or into `.key` files via `keys.py`;
  both are built to hide what they write. If a command would echo a
  credential, it is the wrong command.
- **Ask exactly one question before installing**: which destination
  (table below). Everything else is decided by that answer.
- **System changes are announced**: name the package and why before
  `brew install` / `apt-get install` / touching shell rc files.
- **Non-interactive over prompts**: pass flags (`setup.sh --local
  --postgres bundled`) instead of driving interactive screens; scripts
  refuse without a TTY on purpose so you can drive them.
- **Idempotent re-runs are safe**: `install.sh`, `env.sh init`,
  `keys.py init` can be run again; prefer re-running over improvising.
- Sign-in, invites, and admin promotion need **the human's email and
  password** — you never invent either. Passwords, codes, and magic links are
  entered through hidden prompts. `insideout setup` saves the resulting
  credentials to its `0600` client config; the plain `insideout login` verb
  prints only its access token for an explicit caller to capture.

## Step 0 — platform and destination

1. Detect the platform: `uname -s` (`Darwin` → macOS, `Linux` →
   confirm Ubuntu via `/etc/os-release`) and `uname -m` (arm64/aarch64
   vs x86_64 — module downloads differ). Unsupported → stop.
2. Ask the one question:

| Destination | Gives the user | Needs |
| --- | --- | --- |
| **A. Hosted API** (default) | CLI + MCP driving the hosted service | git, Go, python3, curl — no database, no env vars |
| **B. Own server** | A PostgreSQL-backed server on this machine | A + Docker (or their own Postgres 14+) + `.env` + migrations |
| **C. Full development** | B + the Flutter client (web/iOS/Android) | B + Flutter 3.44.x + Chrome (and Xcode/Android Studio for those targets) |

If the user cannot choose, pick A — it is reversible (B/C build on the
same clone) and needs no secrets.

## Step 1 — install the modules

Follow [references/modules.md](references/modules.md). Each module
states what it is for, which destinations need it, exact macOS and
Ubuntu commands, and the verify command that proves it worked. Install
only the modules the chosen destination needs; check-then-install
(check the verify command first — skip a module that already passes).
Every module must end green before you move on.

## Step 2 — get the repository

```bash
git clone <repository-url> insideout && cd insideout
```

(Use the URL the user gives you; they may already have a checkout —
use it.) The scripts are run from the repo root throughout.

## Step 3 — the chosen path

Once `./scripts/install.sh` has built the binaries, the front door is
`insideout setup` (interactive on a terminal; flags for you, the
agent). The scripts below remain the pre-binary fallback. If a `v*`
release already exists, `./scripts/bootstrap.sh` (or the
`curl | bash` form in
[installation.md](../../docs/usage/installation.md)) installs
prebuilt checksum-verified binaries instead — same Step 2 checkout
still needed for `--local` (migrations, compose, `.env.example`).

**A. Hosted API** — from [installation.md](../../docs/usage/installation.md):

```bash
./scripts/install.sh          # builds both binaries into ~/.local/bin, PATH handled
insideout setup --hosted      # confirms the hosted path (agent form)
# or: insideout setup         # interactive form on a terminal: sign-in + save
insideout whoami              # "authentication required" before sign-in = finished install
```

Do not set `INSIDEOUT_API` on this path. Sign-in (password, emailed
code, or magic link) saves the token to a 0600 config file the CLI
and MCP server read; environment variables still win where set.

**B. Own server** — decide bundled Docker Postgres vs a URL the human
already has (ask if unclear):

```bash
insideout setup --local --postgres bundled    # Docker on 127.0.0.1:5442; writes .env, migrates
# or: insideout setup --local --postgres own  # URL arrives on stdin, never echoed
export INSIDEOUT_API=http://127.0.0.1:8080/api/v1   # or the CLI keeps calling hosted
./scripts/dev.sh -C server go run ./cmd/insideout   # start the server
./scripts/dev.sh -C server go run ./cmd/insideout seed   # disposable local DB only
./scripts/dev.sh -C server go run ./cmd/insideout promote-admin demo@insideout.local   # first local admin
insideout setup   # sign in and save the local token through hidden prompts
```

Pre-binary fallback (identical flags): `./scripts/setup.sh --local
--postgres bundled|own [--no-start]`. A **fresh** own-URL database
needs the two NOSUPERUSER roles first (`insideout_owner` with
BYPASSRLS owning the database, `insideout_app` with CONNECT) —
provision per `docker/postgres-init/001-create-app-role.sh`
([issue](../../docs/issues/2026-10-07-own-url-fresh-db-bootstrap.md)).

Registration is invitation-only on a fresh database; locally-only, the
human may prefer `signup-mode open` (see installation.md).

**C. Full development** — B, plus:

```bash
./scripts/env.sh init         # interactive .env setup; safe to re-run
./scripts/env.sh check        # validates; never prints values
cd client && flutter run -d chrome --web-port=5173 --dart-define=API_BASE=http://127.0.0.1:8080/api/v1
```

Keep `--web-port=5173`: the server builds invite, magic-link and reset
links from `INSIDEOUT_PUBLIC_URL` (default `http://localhost:5173`), and
`flutter run` otherwise picks a random port, so those links would point
at a port nothing listens on.

`dev.sh` gates every launch on `env.sh check` — when it refuses, fix
what the check names, do not bypass it.

## Step 4 — register MCP with the user's agent

```bash
./scripts/mcp-config.sh       # hints + JSON; -q for JSON only
```

Merge the `insideout` entry where the user's agent reads MCP config
(Claude Code `.mcp.json`, Cursor `~/.cursor/mcp.json`, others: the
standard `mcpServers` shape). Reload MCP servers afterwards. Omit the
token until sign-in exists; the `auth_*` tools can sign in later.

## Step 5 — the host key store

Never-absent rule: every mapped kind gets a `.key` file — blank means
"default in effect", missing means broken.

```bash
export KEY_HOST='your-host-name'
INSIDEOUT_KEYS_HOST="$KEY_HOST" python3 scripts/keys.py init    # placeholders for every kind
./scripts/env.sh init                                          # real values into .env (human supplies)
INSIDEOUT_KEYS_HOST="$KEY_HOST" python3 scripts/keys.py fill    # env values into EMPTY .key files
INSIDEOUT_KEYS_HOST="$KEY_HOST" python3 scripts/keys.py audit   # report only; never prints values
```

Per-key detail (where / set / tested / used):
[docs/usage/key-setup-tutorial.md](../../docs/usage/key-setup-tutorial.md),
[docs/usage/key-testing-tutorial.md](../../docs/usage/key-testing-tutorial.md),
and the detailed [key index](../../docs/usage/keys/README.md); the full
sequence with failure signals:
[practice record](../../docs/practices/2026-10-05-setting-up-a-key.md).
On the hosted path this step is optional — no local secrets are
needed.

## Step 6 — verification gates

Run the gates for what was installed; all must pass before you claim
success. Report failures honestly with the command output.

| Gate | Command | Where |
| --- | --- | --- |
| Health probes (API / DB / GitHub / LLM shape) | `insideout setup --check` (add `--llm` for the legacy `openai` compatibility probe; finish current provider keys with the testing tutorial) | any shell, once the binary exists |
| Environment valid | `./scripts/env.sh check` | repo root, always |
| CLI installed | `insideout whoami` (401/authentication-required is a **pass**) | any shell |
| Unit tests | `go test ./...` | `server/` |
| DB integration | `./scripts/dev.sh -C server go test ./internal/store/... -run TestAuthz -v` | repo root, needs migrated DB |
| Live smoke | `./server/scripts/smoke.sh` | repo root, needs curl+jq + DB |
| Client | `flutter analyze --no-fatal-infos && flutter test` | `client/` |
| Env + key suites | `python3 test_env_catalog.py && python3 test_env_writes.py` then `python3 -m unittest discover -s tests/keys -t .` | `scripts/` |

## Step 7 — record the installation

The install is not done until it is written down:

1. **Host record**: create `hosts/<hostname>/` — `README.md` (one
   paragraph: what this machine is for), `host.yml` (name, os,
   checkout path — metadata only, no secrets, no addresses; copy the
   shape of [an existing host](../../hosts/franks-macbook-pro.local/host.yml)),
   and the `keys/` store from Step 5. Add the host to the
   [hosts table](../../hosts/README.md).
2. **Changelog**: if you created or changed repo files (a new host
   record counts), add a dated entry under
   [docs/changelogs/](../../docs/changelogs/README.md) — what was
   installed, versions, decisions taken, what failed or was skipped.
   Stage files individually; never `git add .`.
3. **Report to the user**: modules installed with versions, the
   destination chosen, verification results, and what only the human
   can do next (sign in, supply real credentials, decide signup mode).

## References

- [references/modules.md](references/modules.md) — per-module install
  steps, macOS + Ubuntu.
- [docs/usage/installation.md](../../docs/usage/installation.md) — the
  canonical install guide (the "For an agent" section is the contract
  this skill implements).
- [docs/usage/local-development.md](../../docs/usage/local-development.md)
  — day-to-day commands after setup.
- [docs/SETENV.md](../../docs/SETENV.md) — the `.env` operating manual.
- [docs/usage/environment.md](../../docs/usage/environment.md) — every
  variable, default, and consumer.
