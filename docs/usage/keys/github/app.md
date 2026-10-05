# GitHub App credentials (id + private key + client secret)

`INSIDEOUT_GH_APP_ID` / `INSIDEOUT_GH_PRIVATE_KEY` /
`INSIDEOUT_GH_CLIENT_SECRET` — the GitHub App that powers the evidence
loop (repo binding, guide fetch, webhook authentication).

## 1. Where the `.key` files are

`hosts/<host>/keys/github/.secrets/{gh-app-id,gh-app-private-key,
gh-client-secret}.key` (manifest `github/app.yaml`). The private-key
file keeps the **whole multi-line PEM** — the one key that is not a
single line.

## 2. How it is propagated

The PEM deliberately bypasses the single-line env bridge: leave
`INSIDEOUT_GH_PRIVATE_KEY` unset and point
`INSIDEOUT_GH_PRIVATE_KEY_FILE` straight at the `.key` file. The id
and client secret ride `.env` → `dev.sh` normally; deployed values in
the provider's variable store.

## 3. How to set them

GitHub → Settings → Developer settings → GitHub Apps (the InsideOut
evidence app): the numeric **App id** from the about page, the
**private key** via "Generate a private key" (download the PEM, keep
it multi-line in the `.key` file), the **client secret** from the same
page. Regenerating the private key invalidates the old one — rotate
all consumers at once.

## 4. How to test them

Bind a repo (`insideout repo set`) and `insideout sync <project>` — a
successful guide fetch for a private repo proves the App id + key can
mint an installation token. Webhook delivery health (signed, 200)
proves the pair end-to-end with the webhook secret.

## 5. How they are used

App id + private key authenticate as the App to mint installation
tokens (private-repo guide fetch, evidence sync); the client secret
has **no server reader yet** — kept because it is the App's OAuth
credential for the future invite-link flow.

## 6. Installation guide

This file is the detail. The guidance-system walkthrough that consumes
these keys is [cli-mcp-and-guidance.md](../../cli-mcp-and-guidance.md);
env contract rows in [environment.md](../../environment.md).
