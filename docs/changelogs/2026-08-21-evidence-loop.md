# 2026-08-21 — Evidence loop closed: guide-matched deliveries land on leaves

Plan: [roadmap-parity-and-github](../plans/2026-08-21-roadmap-parity-and-github.md).
The webhook no longer just pulls commits into timelines — deliveries are
matched against the repo's committed `insideout.yaml` and evidence lands
on roadmap leaves.

## What changed

- Migration `20260821130000_roadmap_evidence.sql`: `roadmap_evidence`
  table (node, kind, detail, source URL), FORCE RLS with
  member-select / owner-or-admin-insert policies, and a
  `(node_id, kind, detail)` unique index so GitHub redeliveries are
  idempotent.
- `internal/github/appauth.go`: GitHub App JWT (RS256 from the `.pem`,
  `\n`-escaped env form or `_FILE`), installation access tokens with
  per-installation caching, and `FetchGuideFile` (raw contents download;
  unauthenticated fallback for public repos).
- `internal/guide/parse.go`: `insideout.yaml` v1 parser + matcher
  (branch exact/`prefix/*`, exact labels, path prefixes), leaf-filtered.
- Webhook `push`/`pull_request` handling: parse installation id, branch,
  touched paths / PR labels; load the guide (installation token when
  configured, anon otherwise); match per repo-bound project; append
  evidence as the resolved project owner. PR events record
  `review`/`implementation` kinds.
- `GET /api/v1/roadmap/{nid}/evidence` lists a node's evidence (newest
  first).
- Railway now carries `INSIDEOUT_GH_APP_ID` + `INSIDEOUT_GH_PRIVATE_KEY`
  alongside the webhook secret.

## Verification

- Unit: guide parse/match (prefix rules, leaf filter, rejections), JWT
  and PEM loading; full `go test ./...`, `go vet`, `gofmt` green.
- Migration applied by Railway boot-migrate as owner (19/19).
- **Live loop, all through the domain**: scratch project built by the
  agent planner → repo bound → `insideout guide` generated, matchers
  (`main`, `server/`) set on a leaf, and committed as this repo's own
  dogfood `insideout.yaml` → signed push delivery (ref main, commits
  touching `server/`) returned `evidence: 1` →
  `GET /roadmap/{leaf}/evidence` shows the activity row with the commit
  URL → **redelivery left the table at exactly one row** (dedupe works).

## Caveats

- The installation-token path is built and configured; the live test
  used the public-repo unauthenticated fallback. The app was installed
  on the repo right after (install ping answered 200), so deliveries
  now carry an installation id and the token path is active.
- This repo's committed guide references the scratch project's node ids
  — regenerate from the real project (`insideout guide <pid>`) when one
  exists.
