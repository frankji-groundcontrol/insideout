# The app's nginx follows its API upstream

`client/nginx.conf` looked the API upstream up once, when nginx loaded its
configuration. After a `server` deploy the public `/api/` returned 504 until
`app` was restarted (the "restart app" ritual in HANDOFF), and nginx would not
start at all while the upstream name did not resolve
([BUG-027](../../issues/2026-10-10-bug-027-nginx-resolved-the-api-upstream-once.md),
review items REL-07 and REL-18).

## What changed

- **`client/nginx.conf`:** `resolver ${NGINX_LOCAL_RESOLVERS} valid=10s;` and
  `set $api_upstream ${NGINX_API_UPSTREAM}; proxy_pass $api_upstream;`. A
  variable makes nginx resolve per request; an answer is cached for 10 s.
- **`client/Dockerfile`:** `ENV NGINX_ENTRYPOINT_LOCAL_RESOLVERS=1`, which makes
  the image entrypoint export the container's nameservers (from
  `/etc/resolv.conf`, IPv6 in brackets) as `NGINX_LOCAL_RESOLVERS` before it
  renders the template.
- **`scripts/check_nginx_reresolve.py`:** a real-nginx check with a stub DNS
  server and two upstreams (red before the fix, six of six after).

No new variable for operators: `NGINX_API_UPSTREAM` is unchanged
(`scheme://host:port`, no path), and compose and Railway already set it that way.

## How it was verified

- nginx 1.27.5, built from source into a scratch directory (the pinned image
  ends on 1.27.5). Red: nginx refuses to start with `host not found in upstream`.
  Green: starts without DNS and answers 502; finds the upstream when the name
  resolves; passes `Host`, `Authorization`, `X-Forwarded-Proto` and the query
  string; follows the name to a new address after the old upstream is shut; a
  chunked response arrives chunk by chunk.
- Mutations: no `resolver` fails four checks, no `Host` header fails one;
  `proxy_buffering on` is not caught, because nginx 1.27.5 forwards a slow,
  small stream as it arrives either way
  ([learning](../../learning/2026-10-10-a-check-that-passes-under-its-mutation-guards-nothing.md)).
- The entrypoint script and the Dockerfile's `COPY` of it were read at tag
  1.27.5 of `nginxinc/docker-nginx-unprivileged`; they were not run.

## What an operator needs to do

Not run against the real image or Railway's DNS (no container runtime here), and
not deployed. Deploy `app` (`railway up --service app`), then check
`/api/v1/me` returns 401 through the domain; deploy `server` alone and check it
again within 15 seconds without restarting `app`. Keep the HANDOFF restart note
until that has been seen.
