# 2026-08-21 — Custom domain insideout.yalotein.net

The product now serves from one domain instead of two Railway hostnames.

## What changed

- `insideout.yalotein.net` (CNAME → Railway, DNS-only on Cloudflare)
  is attached to the `app` service. Railway verified ownership (TXT
  `_railway-verify.insideout`) and issued a valid certificate.
- The web app, the same-origin API proxy (`/api/v1/…` → `server`), and
  the future GitHub webhook
  (`https://insideout.yalotein.net/api/v1/hooks/github`) all live
  behind it.

## Verification

- `https://insideout.yalotein.net/` → 200 over valid TLS.
- `/api/v1/me` → 401 (proxy routes to the Go server's auth).
- Flutter assets served with the icon-only FontManifest (current web
  build).

The old `*.up.railway.app` hostnames keep working; the domain is the
canonical URL going forward (HANDOFF examples updated).
