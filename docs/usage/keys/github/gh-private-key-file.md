# gh-private-key-file

`INSIDEOUT_GH_PRIVATE_KEY_FILE` — the path pointer at the canonical
App PEM. Config companion of
[gh-app-private-key](gh-app-private-key.md) (`kind: path`).

## 1. Where the `.key` file is

`hosts/<host>/keys/github/.secrets/gh-private-key-file.key` (manifest
`github/app.yaml`) — this `.key` holds a **path**, and the path it
holds points at `gh-app-private-key.key` in the same `.secrets/`.

## 2. How it is propagated

`.env` → `dev.sh` export → `github.LoadPrivateKey`, which prefers the
`_FILE` variant when set and reads the target **raw** (no `\n`
unescaping — that is the env var's lane). Setting it is why the store
keeps the canonical multi-line PEM.

## 3. How to set it

The absolute path of the PEM `.key` in this host's store. Leave the
inline `INSIDEOUT_GH_PRIVATE_KEY` unset when the pointer is used — one
lane at a time.

## 4. How to test it

`insideout sync <project>` on a bound **private** repo (token mint
exercises the PEM end-to-end); the per-key test additionally asserts
the pointed-at file exists.

## 5. How it is used

Routes the App's authenticating credential from the store into the
server without an env-file bridge — the fix for the escaped-PEM
defect (2026-10-05).

## 6. Installation guide

This file is the detail. Contract row in
[environment.md](../../environment.md).
