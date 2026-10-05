# gh-app-id

`INSIDEOUT_GH_APP_ID` — the numeric id of the GitHub App that powers
the evidence loop.

## 1. Where the `.key` file is

`hosts/<host>/keys/github/.secrets/gh-app-id.key` (manifest
`github/app.yaml`; siblings
[gh-app-private-key.md](gh-app-private-key.md) and
[gh-client-secret.md](gh-client-secret.md)).

## 2. How it is propagated

`.env` → `dev.sh` export → the GitHub client at installation-token
time; deployed value in the provider's variable store.

## 3. How to set it

GitHub → Settings → Developer settings → GitHub Apps (the InsideOut
evidence app) — the numeric id on the about page. It travels with the
App's private key: one identifies, the other authenticates.

## 4. How to test it

Bind a repo (`insideout repo set`) and `insideout sync <project>` — a
successful guide fetch for a private repo proves id + key mint an
installation token together.

## 5. How it is used

Names the App when minting installation tokens (private-repo guide
fetch, evidence sync). Alone it authenticates nothing.

## 6. Installation guide

This file is the detail. The consumer walkthrough is
[cli-mcp-and-guidance.md](../../cli-mcp-and-guidance.md); contract row
in [environment.md](../../environment.md).
