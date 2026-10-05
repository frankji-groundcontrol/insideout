# public-url

`INSIDEOUT_PUBLIC_URL` — the browser origin of the app as the world
reaches it; the base of every emailed link. Config companion of
[mail-api-key](mail-api-key.md).

## 1. Where the `.key` file is

`hosts/<host>/keys/mail/.secrets/public-url.key` (manifest
`mail/resend.yaml`).

## 2. How it is propagated

`.env` → `dev.sh` export → link construction in auth mails; deployed
value in the provider's variable store. Catalog default
(`http://localhost:5173`) is dev-only: the server **fail-fasts if a
mail key is set while this stays at the default**, so emailed links
can never point at localhost.

## 3. How to set it

The deployment's public origin (scheme + host, no trailing path).
Local servers that never send real mail can leave the default.

## 4. How to test it

Request a magic link and read the link's host in a real inbox — it
must be this origin, not localhost.

## 5. How it is used

Prefixes magic-link/reset URLs; also the reference origin for cookie
and CORS decisions alongside the dedicated toggles.

## 6. Installation guide

This file is the detail. Contract row in
[environment.md](../../environment.md).
