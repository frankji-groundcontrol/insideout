# Tutorial: verify every InsideOut key

This tutorial turns a configured environment into evidence. It uses three
levels: file shape, service reachability, and the real capability enabled by a
key. A shape test alone never proves that a provider, role, or external
delivery works.

Never put credentials in shell arguments or print response bodies that may
contain tokens. Keep `.env` and key values out of logs.

## 1. Run the safe shape and catalog suites

From the repository root:

```bash
python3 scripts/test_env_catalog.py
python3 scripts/test_env_writes.py
(cd scripts && python3 -m unittest discover -s tests/keys -t . -p 'test_*.py' -v)
```

The key suite checks all 31 mapped kinds for placement, mode `0600`, mapped
variable, and type-specific shape. It does not compare values or prove an
external service. A missing host key is an error; an empty key is valid only
when the manifest declares a default or optional lane.

## 2. Prove the environment and database roles

```bash
./scripts/env.sh check server --db
./scripts/dev.sh -C server go test ./internal/store/... -run TestAuthz -v
./scripts/dev.sh -C server go run ./cmd/insideout migrate
```

The first command proves only database endpoint reachability (host and port);
it does not authenticate the DSN. The real-DB
authorization suite proves runtime RLS behavior. The migration command proves
the owner path can apply schema changes. A `401` from `/api/v1/me` proves API
routing and authentication enforcement; it does not prove either database role.

## 3. Prove the server and runtime settings

Use the setting's reference for the exact observable check:

| Setting family | Proof | Expected result |
| --- | --- | --- |
| [address](keys/app/addr.md) | `insideout health` or `curl` the configured `/healthz` | HTTP 200 on the intended bind address |
| [cookie security](keys/app/cookie-secure.md) | inspect the login response in a local browser/HTTP trace | `Secure` follows the setting; plain HTTP local testing needs `0` |
| [CORS](keys/app/cors-origins.md), [dev CORS](keys/app/dev-cors.md) | send a preflight from the intended origin | only the declared origin receives the expected headers |
| [access](keys/app/access-ttl.md) and [refresh TTL](keys/app/refresh-ttl.md) | login, decode expiry metadata without printing the token, and wait/use a short disposable TTL | access expiry and refresh rotation match the configured durations |
| [compose ports](keys/compose/server-port.md) / [Postgres port](keys/compose/postgres-port.md) / [client port](keys/compose/client-port.md) | `docker compose up` followed by health and browser requests | published ports accept the intended service |
| [superuser bootstrap password](keys/compose/postgres-superuser-password.md) | create a fresh disposable volume and run the init path | roles initialize; never test by printing the password |

## 4. Prove authentication and JWT signing

For [the JWT secret](keys/auth/jwt-secret.md), boot the server with the
configured environment, perform a real login against the disposable server,
and make an authenticated request. The token must validate on the same server;
an empty or short secret must fail before boot. The real-DB authorization suite
and the end-to-end smoke flow exercise this path without exposing token text.

## 5. Prove the coach configuration

For the [LLM settings](keys/llm/README.md):

```bash
insideout setup --check
insideout setup --check --llm
```

The first command checks key shape. The `--llm` probe calls `/models` only for the legacy `openai` schema. The
current `messages` and `responses` schemas report shape-only with a skipped
live probe; a skipped probe is not a successful coach test.
Finish with one real coach turn through the SSE endpoint. A real `message_end`
proves the configured key, base URL, model, schema, and response handling work
together. The repository smoke flow performs this provider leg when the key is
configured.

## 6. Prove mail delivery

For [the mail settings](keys/mail/README.md), an auth request returning `200`
only proves the anti-enumeration API contract. It does not prove delivery.
Run this only against an isolated disposable database: the smoke flow creates
throwaway rows, requires `signup_mode=open`, requires `DATABASE_OWNER_URL` to
promote its probe user, and leaves that user promoted to admin for explicit
cleanup or demotion afterward. Do not set `SMOKE_BASE`, because that skips the
provider leg. Use an operator-owned verified recipient:

```bash
export SMOKE_MAIL_TO='verified-recipient@example.test'
./server/scripts/smoke.sh
```

The output must report `status=sent`. Then redeem one delivered invitation,
magic link, or login code. For local mail dev mode, expect a logged development
status instead of delivery; the smoke provider leg is skipped when the mail key
is empty, so observe one local `/auth/magic-link` request in the server log
separately and keep the body switch local.

## 7. Prove GitHub access

Bind a project and run the manual sync path. A public repository is read
anonymously; a private one is read with the GitHub App's installation token, not
with `GITHUB_TOKEN` (rv4-FEAT-04):

```bash
export PROJECT_ID='project-id'
export REPO_URL='https://github.com/owner/repo'
insideout repo set "$PROJECT_ID" "$REPO_URL"
insideout sync "$PROJECT_ID"
```

For the [GitHub App ID and private key](keys/github/README.md), install the App
on a private test repository and trigger a signed App delivery that requires a
private guide fetch. The delivery must answer HTTP 200 and the server log must
not report an unusable private key. For the [webhook secret](keys/github/gh-webhook-secret.md),
send a signed delivery from GitHub and require HTTP 200; a mismatched signature
must be rejected. To prove commit sync on a private repository, install the App
on it and run the sync above: the installation token reads it. The
[optional GitHub token](keys/github/github-token.md) is read only by
`insideout setup --check` and is validated there when present. The stored OAuth
client secret has no live consumer and receives placement-only verification.

## 8. Troubleshoot by evidence

- Shape failure: correct the value through `env.sh edit`; never hand-edit the
  ignored key file as the live source.
- Database reachability failure: verify the DSN role, host, port, and pooler
  option; run the owner migration check separately.
- Provider rejection: distinguish an invalid credential from an endpoint or
  schema mismatch; use status-only probes and never paste response bodies.
- Delivery failure: verify the sender domain, public URL, recipient policy,
  and provider status before changing the key.
- A boot warning about an unusable key is a failure. Re-run the capability
  probe until the warning is absent and the observable action succeeds.

Finish by recording the command, status, and next action in the installation
record. The per-key references under [keys/](keys/README.md) remain the
authoritative details for each setting.
