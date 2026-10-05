# gh-client-secret

`INSIDEOUT_GH_CLIENT_SECRET` — the GitHub App's OAuth client secret.

## 1. Where the `.key` file is

`hosts/<host>/keys/github/.secrets/gh-client-secret.key` (manifest
`github/app.yaml`; siblings
[gh-app-id.md](gh-app-id.md) and
[gh-app-private-key.md](gh-app-private-key.md)).

## 2. How it is propagated

`.env` → `dev.sh` export — but **no server reader consumes it yet**;
the store keeps it so the credential is not lost.

## 3. How to set it

From the GitHub App settings page (the client secret shown once per
generate). Regenerate + re-store when the future consumer lands.

## 4. How to test it

Nothing to exercise today — presence in the scoped `keys.py audit`
(`key=filled`) is the only check. When the invite-link web flow is
implemented, its OAuth callback becomes the probe.

## 5. How it is used

Reserved for the future GitHub-invite-link flow (web OAuth login).
Deliberately tracked now: it is the App's credential, losing it would
force a regenerate at GitHub.

## 6. Installation guide

This file is the detail. Contract row in
[environment.md](../../environment.md).
