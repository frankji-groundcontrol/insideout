# gh-app-private-key

`INSIDEOUT_GH_PRIVATE_KEY` — the GitHub App's private-key PEM; the
authenticating half of the App credentials.

## 1. Where the `.key` file is

`hosts/<host>/keys/github/.secrets/gh-app-private-key.key` (manifest
`github/app.yaml`; siblings
[gh-app-id.md](gh-app-id.md) and
[gh-client-secret.md](gh-client-secret.md)). The one key file that is
**not** a single line — it keeps the whole multi-line PEM.

## 2. How it is propagated

It deliberately bypasses the single-line env bridge: leave
`INSIDEOUT_GH_PRIVATE_KEY` unset and point
`INSIDEOUT_GH_PRIVATE_KEY_FILE` straight at the `.key` file.

## 3. How to set it

GitHub App page → "Generate a private key" — download the PEM and
store it multi-line in the `.key` file. Regenerating invalidates the
old key: rotate every consumer at once (local `.env` pointer +
provider store).

## 4. How to test it

`insideout sync <project>` against a bound **private** repo — the
installation-token mint fails loudly with a bad or missing PEM.

## 5. How it is used

Signs the App's installation-token requests (JWT → installation
token) used for private-repo guide fetch and evidence sync.

## 6. Installation guide

This file is the detail. The consumer walkthrough is
[cli-mcp-and-guidance.md](../../cli-mcp-and-guidance.md); contract
rows (`INSIDEOUT_GH_PRIVATE_KEY` and the `_FILE` variant) in
[environment.md](../../environment.md).
