# BUG-027 — the app's nginx resolved the API upstream once, at start

## Symptom

After every `server` deploy on Railway the public `/api/` answered 504 until
someone restarted the `app` service. [HANDOFF](../HANDOFF.md) carried it as a
hand ritual: "After a `server` deploy, restart `app` … its nginx resolves the
server's internal address only at startup".

Both multirole reviews filed it (REL-07 on 2026-10-04: "every server redeploy
ships a guaranteed public API outage until a human restarts `app`"; REL-18 on
the second pass: the dated issue that REL-07 asked for still did not exist).

A second symptom turned up while fixing it: when the upstream name does not
resolve at the moment nginx starts (the `server` service is not up yet), nginx
refuses to start at all: `[emerg] host not found in upstream`.

## Root cause

`client/nginx.conf` had `proxy_pass ${NGINX_API_UPSTREAM}/api/;`. After the
image entrypoint's `envsubst` that is a plain host name, and nginx looks a plain
host name up once, while it loads the configuration, and keeps the address for
the life of the process. A redeployed `server` gets a new private address;
nginx keeps dialing the old one.

FS-06 (2026-10-04) had made the upstream portable with `envsubst` but left the
literal `proxy_pass`, and its promised issue record was never written.

## Fix

| Step | Commit |
| --- | --- |
| Red test: `scripts/check_nginx_reresolve.py` renders the template like the entrypoint, runs it on a real nginx beside a stub DNS server and two upstreams, and fails on the committed template with `host not found in upstream` | `2797afd` |
| Fix: `resolver ${NGINX_LOCAL_RESOLVERS} valid=10s;`, `set $api_upstream ${NGINX_API_UPSTREAM};` and `proxy_pass $api_upstream;`; the Dockerfile sets `NGINX_ENTRYPOINT_LOCAL_RESOLVERS=1` so the entrypoint exports the container's nameservers as `NGINX_LOCAL_RESOLVERS` | `fd8a144` |

A variable in `proxy_pass` makes nginx resolve per request. `NGINX_API_UPSTREAM`
stays `scheme://host:port` with no path, which is what both Railway and compose
already set; with a variable and no URI nginx forwards the request URI as the
client sent it.

## Verification

- `python3 scripts/check_nginx_reresolve.py` on nginx 1.27.5 (the line the
  pinned `nginxinc/nginx-unprivileged:1.27-alpine` image ends on), built from
  source into a scratch directory. Six checks: starts while the name does not
  resolve; answers 502 rather than hanging; finds the upstream once the name
  resolves; passes `Host`, `Authorization` and `X-Forwarded-Proto` and the query
  string; follows the name to a new address after the old upstream is gone; and
  a chunked response arrives chunk by chunk. Red 0 of 1 before (nginx does not
  start), 6 of 6 after.
- Mutations: deleting the `resolver` line fails four checks; deleting
  `proxy_set_header Host $host;` fails the header check. `proxy_buffering on` is
  **not** caught: nginx 1.27.5 forwards a slow, small stream as it arrives with
  buffering on as well, so the last check guards the request path through the
  variable, not that directive
  ([learning](../learning/2026-10-10-a-check-that-passes-under-its-mutation-guards-nothing.md)).
- The entrypoint, read from source rather than run: the unprivileged image's
  Dockerfile at tag 1.27.5 has `COPY 15-local-resolvers.envsh
  /docker-entrypoint.d`; the script exports `NGINX_LOCAL_RESOLVERS` from the
  `nameserver` lines of `/etc/resolv.conf` when `NGINX_ENTRYPOINT_LOCAL_RESOLVERS`
  is non-empty, and wraps IPv6 addresses in brackets.
- The server issues no redirects (`http.Redirect` and `"Location"` are absent
  from non-test code), so losing `proxy_redirect default`, which a variable
  `proxy_pass` turns off, changes nothing.

## Not verified, and why

- **The real image and Railway's DNS.** There is no container runtime here, so
  the Dockerfile and the entrypoint's output were not run, and Railway's
  private network (IPv6) was not reached.
- **The old behaviour against a changing system resolver.** Changing what the
  system resolver answers needs root. The red run shows the load-time lookup
  directly (nginx will not start for an unresolvable name); that the address then
  stays fixed is nginx's documented behaviour and the HANDOFF history.
- **Not deployed.** It takes effect on the next `railway up --service app`.

## Owner check after the deploy

1. `curl -s -o /dev/null -w '%{http_code}\n' https://<app-domain>/api/v1/me`
   returns `401` (not `/healthz`, which the SPA answers).
2. Deploy `server` alone, wait for it to be healthy, and repeat the request
   within about 15 seconds without restarting `app`: still `401`, not `504`.
3. Then drop the restart ritual from HANDOFF. Until step 2 has been seen, keep it.

## Status

**Fixed in the repository 2026-10-10**; live only after the owner deploys `app`.
Re-run `scripts/check_nginx_reresolve.py` when the nginx base image is bumped.
