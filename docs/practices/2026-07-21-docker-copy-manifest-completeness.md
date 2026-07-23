# Practice: Docker layer-cache COPY must include every manifest the installer reads

**Date**: 2026-07-21

## Trigger

Writing or editing any Dockerfile that uses the layer-cache pattern: `COPY <manifests>` → `RUN <install>` → `COPY . .`. Also when upgrading a package manager (pnpm major versions move where config lives) or adding installer config files.

## Sequence / guardrail

The first `COPY` must contain **every file the package manager actually reads during install**, not just the obviously relevant ones:

- **pnpm** (`app/Dockerfile`): `package.json` + `pnpm-lock.yaml` + `pnpm-workspace.yaml` (which holds pnpm v10's `allowBuilds` map) — plus `.npmrc` if one exists.
  ```dockerfile
  COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
  RUN pnpm install --frozen-lockfile
  ```
- **Go** (`server/Dockerfile`): `go.mod` + `go.sum`.
  ```dockerfile
  COPY go.mod go.sum ./
  RUN go mod download
  ```

Before finishing: list the installer's config surface (lockfile, workspace file, rc files) and diff it against the `COPY` line.

## Verification

Build from a clean context — the host's caches and prior interactive approvals must not be able to rescue the build:

```bash
docker compose build --no-cache
```

A build that only ever ran with warm layers has not verified this practice.

## Failure signals

- Builds pass on the host machine but fail in a fresh container (the host had state — e.g. a prior `pnpm approve-builds` — that the build context lacks).
- `[ERR_PNPM_IGNORED_BUILDS] Ignored build scripts: ...` from `pnpm install --frozen-lockfile`.
- Go: `missing go.sum entry` during `go mod download`.
- Any install-step error that disappears when you `COPY . .` before installing (which also destroys the layer cache — fix the manifest list instead).

## Related

- [BUG-006: pnpm ignored build scripts in fresh Docker build](../issues/2026-07-20-bug-006-pnpm-ignored-build-scripts.md)
- [BUG-004: compose nested interpolation](../issues/2026-07-20-bug-004-compose-nested-interpolation.md)
- [Learning records](../learning/README.md)
