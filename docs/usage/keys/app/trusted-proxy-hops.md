# trusted-proxy-hops

`INSIDEOUT_TRUSTED_PROXY_HOPS` — how many reverse proxies in front of the
server append to `X-Forwarded-For`. Blank means the server default `0` (key the
per-client throttles on the TCP peer); the shipped compose stack defaults it to
`1`.

## 1. Where the `.key` file is

`hosts/<host>/keys/app/.secrets/trusted-proxy-hops.key` (manifest
`app/runtime.yaml`). The file is one value with mode `0600`.

## 2. How it is propagated

The value moves from the root `.env` through `scripts/dev.sh` into
`config.Load`. Under compose, `docker-compose.yml` passes
`${INSIDEOUT_TRUSTED_PROXY_HOPS:-1}` to the server. A hosted server reads it
from the platform's service variables.

## 3. How to set it

1. Count the proxies in front of the server that append the client address
   to `X-Forwarded-For`: `0` for a bare server, `1` for the compose stack or
   the hosted deployment, plus one for every extra appending proxy.
2. Run `./scripts/env.sh edit`, choose `INSIDEOUT_TRUSTED_PROXY_HOPS`, and
   enter the integer.
3. Initialize/fill the host key store:

   ```bash
   INSIDEOUT_KEYS_HOST='<host>' python3 scripts/keys.py init
   INSIDEOUT_KEYS_HOST='<host>' python3 scripts/keys.py fill
   ```

With hops at 1 or more, bind the published client port (`CLIENT_PORT`) to
`127.0.0.1` or firewall it, unless nothing else can reach it.

## 4. How to test it

Run the shape test, then restart the server:

```bash
cd scripts && python3 -m unittest discover -s tests/keys -t . -p 'test_trusted-proxy-hops.py' -v
```

**Expected result:** the test passes and the server starts. A non-integer or
negative value fails startup. With hops at `0`, a request that carries
`X-Forwarded-For` logs one warning that the header is ignored.

**If it fails:** recount the appending proxies and restart the server after
editing.

## 5. How it is used

The auth throttle and the resend cooldown key on the address `N` hops from the
right of `X-Forwarded-For`, `N` being this value; a shorter chain or an entry
that is not an IP address falls back to the TCP peer.

## 6. Installation guide

The [deployment guide](../../deployment.md#reverse-proxy--same-origin-expectation)
explains the client-IP trust chain for compose and the hosted service. The
variable's row is in the [environment reference](../../environment.md).
