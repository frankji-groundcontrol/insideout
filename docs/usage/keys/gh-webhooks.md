# GitHub webhook secret

`INSIDEOUT_GH_WEBHOOK_SECRET` — the HMAC shared secret GitHub signs
every webhook delivery with.

## 1. Where the `.key` file is

`hosts/<host>/keys/github/.secrets/gh-webhook-secret.key` (manifest
`github/webhooks.yaml`).

## 2. How it is propagated

`.env` → `dev.sh` export → the server's `/api/v1/hooks/github`
receiver. The **same string** must also be set in the GitHub App's
webhook configuration — it is one secret with two homes by design.

## 3. How to set it

Self-minted: `openssl rand -hex 32`, then paste it into the GitHub
App's webhook secret field and into `.env` (never into anything
tracked). One per deployment; local-dev receivers that never get real
deliveries can mint freely.

## 4. How to test it

The GitHub App's delivery history: deliveries must arrive signed
(`X-Hub-Signature-256`) and be answered 200; a mismatched secret
shows as 401/403 on the receiver side with unsigned rejections logged.

## 5. How it is used

The receiver HMAC-verifies every delivery before parsing — an
unsigned or mismatched body never reaches the evidence pipeline.
Redeliveries are idempotent, so verification is the only gate.

## 6. Installation guide

The guidance-system chapter of
[cli-mcp-and-guidance.md](../cli-mcp-and-guidance.md) (step 4,
installing the GitHub App).
