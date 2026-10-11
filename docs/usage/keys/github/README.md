# Keys — github module

Per-key reference for the `github/` module of the key store. Each
file documents one key kind with the six parts (where the `.key` file
is, propagation, setting, testing, use, and the installation-guide
pointer). The module table, shared flow, and per-host state live in
the [keys README](../README.md).

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
