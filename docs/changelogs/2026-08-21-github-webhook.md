# 2026-08-21 — GitHub webhook live (Stage B core)

Plan: [roadmap-parity-and-github](../plans/2026-08-21-roadmap-parity-and-github.md).
Deliveries to `https://insideout.yalotein.net/api/v1/hooks/github` now
verify and sync.

## What changed

- `POST /api/v1/hooks/github` (no user auth): HMAC-SHA256 verification
  of `X-Hub-Signature-256` against `INSIDEOUT_GH_WEBHOOK_SECRET`
  (constant-time; empty secret → 503). `ping` acknowledges; `push` and
  `pull_request` resolve the repo's projects and re-run the existing
  per-project commit sync; other events are ignored with 200.
- Migration `20260821120000_github_webhook_repo_lookup.sql`: SECURITY
  DEFINER `insideout._projects_by_repo(repo_url) → (project, owner)` —
  the webhook has no RLS identity, so a DEFINER helper resolves targets
  and every subsequent read/write runs as that project's owner through
  the normal user-scoped store methods.
- `handleSyncGithub` refactored onto a shared `syncGithubProject` core
  used by both the user route and the webhook.
- Config/env: `INSIDEOUT_GH_WEBHOOK_SECRET` (Go config + `.env.example`
  + Railway). The future `INSIDEOUT_GH_PRIVATE_KEY` will accept a
  `\n`-escaped PEM or a `_FILE` path.

## Verification

- Unit: signature accept/reject (wrong secret, tampered body, wrong
  scheme, malformed hex), payload parsing; full `go test ./...` green.
- Live: migration applied by Railway boot-migrate as `insideout_owner`
  (18/18 — first production use of the documented owner-URL path).
  Direct and via the domain: ping → `{"ok":true}`; bad signature → 401;
  signed push → resolved 2 projects (both scratch projects had the repo
  bound) and synced 62 backlog commits; re-delivery → `commits: 0`
  (cursor idempotent).

## Gotchas recorded

- **nginx caches the server's internal IP**: redeploying `server`
  changes its Railway internal address, but the `app` service's nginx
  resolves upstreams only at config load, so `/api` via the domain
  504s until `app` restarts. After any `server` deploy, restart `app`
  (`railway redeploy --service app`). Verification via `/api/v1/me`
  (expect 401), not `/healthz` on the app host (the SPA answers 200).
- The first sync after a long gap can exceed nginx's ~60 s proxy
  timeout (backlog walk); GitHub retries deliveries, so the eventual
  state converges.

## Remaining Stage B

- Install the GitHub App on `frankji-groundcontrol/insideout` (user
  action) so real deliveries flow.
- `insideout.yaml` matching guide loader + installation tokens
  (private-key JWT) to advance leaf-node evidence, replacing the
  timeline-only pull.
