# GitHub token (optional PAT)

`GITHUB_TOKEN` — an operator-minted personal access token; **not
needed to run anything**.

## 1. Where the `.key` file is

`hosts/<host>/keys/github/.secrets/github-token.key` (manifest
`github/tokens.yaml`). Empty by default on both hosts — that is the
intended state until rate limits actually bite.

## 2. How it is propagated

Read ad hoc from process env by `server/internal/github/github.go`
when present; no other consumer.

## 3. How to set it

Only if evidence sync hits the unauthenticated rate limit: mint a PAT
with the narrowest scopes that covers the repos you bind
(public-repo read is usually enough), put it in `.env`, fill the
store.

## 4. How to test it

Evidence sync throughput — syncs that previously rate-limited now
complete; GitHub's rate-limit headers in the server logs confirm the
authenticated bucket.

## 5. How it is used

Raises evidence sync above the unauthenticated rate limit. Nothing
else; it is never required for the webhook receiver or the App flow
(those use the App credentials).

## 6. Installation guide

Mentioned in [environment.md](../environment.md); no SETENV step —
optional by design.
