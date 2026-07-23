# BUG-006: `pnpm install --frozen-lockfile` fails in a fresh Docker build with `[ERR_PNPM_IGNORED_BUILDS]`

**Found**: 2026-07-20, during the InsideOut rewrite (P7), building `app/Dockerfile` inside `docker compose build`.

**Symptom**: `[ERR_PNPM_IGNORED_BUILDS] Ignored build scripts: @parcel/watcher@2.5.6, esbuild@0.27.2, esbuild@0.28.1` followed by `pnpm install --frozen-lockfile` exiting non-zero, failing the Docker build. This did not reproduce locally — `pnpm install` on the host machine completed without complaint.

**Root cause**: newer pnpm versions default-deny running a dependency's native install/postinstall scripts unless the package is explicitly allow-listed (a supply-chain-security feature) — interactively resolved via `pnpm approve-builds`, which isn't usable in a non-interactive Docker build. The host machine didn't hit this because a prior local `pnpm install` had already run `pnpm approve-builds` once, which pnpm v10 records as an `allowBuilds` map in `pnpm-workspace.yaml` (not, as first assumed, an `onlyBuiltDependencies` key under `package.json`'s `"pnpm"` field — that key is a no-op on this pnpm version). `app/Dockerfile`'s layer-caching step copied only `package.json` and `pnpm-lock.yaml` before `RUN pnpm install --frozen-lockfile`, so `pnpm-workspace.yaml` — the file that actually held the allow-list — never made it into the build context, and a fresh container saw no approval at all.

**Fix**: added `pnpm-workspace.yaml` to the `COPY` step that precedes `pnpm install` in `app/Dockerfile`:
```dockerfile
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile
```
Removed the ineffective `"pnpm": {"onlyBuiltDependencies": [...]}` block from `package.json` since it did nothing on this pnpm version. Verified with a full `docker compose build` — the `app` image now builds cleanly with no ignored-builds error.

**Why it matters**: pnpm's per-package build allow-list can live in more than one place depending on version/config; check `pnpm-workspace.yaml` first, not just `package.json`. And for any Docker build using a layer-caching `COPY <manifest files> ./` step before install, every file pnpm actually reads during install (lockfile, workspace config, `.npmrc`) must be in that `COPY` line — not just the ones that seem obviously relevant.
