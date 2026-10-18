# Keys — github module

Per-key reference for the `github/` module of the key store. Each
file documents one key kind with the six parts (where the `.key` file
is, propagation, setting, testing, use, and the installation-guide
pointer). The module table, shared flow, and per-host state live in
the [keys README](../README.md).

## Guided GitHub flow

Use the [setup tutorial](../../key-setup-tutorial.md#6-configure-github-evidence-access)
after the [own-server installation path](../../installation.md#your-own-server).
The [testing tutorial](../../key-testing-tutorial.md#7-prove-github-access)
defines the capability proof for each credential. App ID + PEM credentials are
used when a signed GitHub App delivery fetches a private `insideout.yaml`;
`insideout sync` uses the optional `GITHUB_TOKEN` (or GitHub's unauthenticated
public API path) and is not an App-key test. The OAuth client secret is retained
for a future consumer and is placement-only today. All App and webhook kinds
are optional: a blank placeholder means the GitHub App integration is disabled;
a nonblank value must pass its shape and capability check.

- [gh-app-id.md](gh-app-id.md) — `INSIDEOUT_GH_APP_ID`
  (manifest `github/app.yaml`)
- [gh-app-private-key.md](gh-app-private-key.md) —
  `INSIDEOUT_GH_PRIVATE_KEY` (`github/app.yaml`)
- [gh-client-secret.md](gh-client-secret.md) —
  `INSIDEOUT_GH_CLIENT_SECRET` (`github/app.yaml`)
- [gh-private-key-file.md](gh-private-key-file.md) —
  `INSIDEOUT_GH_PRIVATE_KEY_FILE` (kind: path) (`github/app.yaml`)
- [gh-webhook-secret.md](gh-webhook-secret.md) —
  `INSIDEOUT_GH_WEBHOOK_SECRET` (`github/webhooks.yaml`)
- [github-token.md](github-token.md) — `GITHUB_TOKEN` (optional)
  (`github/tokens.yaml`)
