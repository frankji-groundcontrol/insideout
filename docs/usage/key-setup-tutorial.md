# Tutorial: set up every InsideOut key

This tutorial takes a new operator from an empty checkout to a configured
InsideOut environment. It keeps real values in the root `.env`, keeps the
host key store synchronized, and links each setting to its detailed reference.
The companion [key testing tutorial](key-testing-tutorial.md) proves the
settings work.

Do not print `.env`, `.env.local`, DSNs, passwords, tokens, signed URLs, or
provider identifiers. Use `env.sh` and the setup prompts; they redact output.

## Before you start

Use macOS or Ubuntu with the toolchain required by the
[installation guide](installation.md) and the
[installation skill](../../skills/insideout-setup/SKILL.md). Decide whether
this machine will use the hosted API or run its own PostgreSQL-backed server.
Hosted CLI users do not configure server keys. This tutorial applies when you
run the server, operate a deployment, or maintain the host key store.

## 1. Create the environment safely

From the repository root, create or update the root environment file through
the interactive writer:

```bash
./scripts/env.sh init
# or choose one setting at a time:
./scripts/env.sh edit
./scripts/env.sh check server
```

The only unconditional server requirements are `DATABASE_URL` and
`INSIDEOUT_JWT_SECRET`. A commented setting in `.env.example` has a working
default or an intentional empty lane; `env.sh check` reports the difference
without showing its value.

## 2. Configure the database and authentication foundation

Choose one database route in [Step 1 of SETENV](../SETENV.md#step-1-database_url-required):

- [Runtime database DSN](keys/database/database-app-dsn.md) and
  [migration-owner DSN](keys/database/database-owner-dsn.md) for an existing
  PostgreSQL instance.
- [Bundled app password](keys/database/postgres-app-password.md) and
  [bundled owner password](keys/database/postgres-owner-password.md) when the
  compose volume is new.

For a guided local server, use:

```bash
insideout setup --local --postgres bundled
# or provide an existing DSN on hidden stdin:
insideout setup --local --postgres own
```

The bundled roles and passwords are initialized with a fresh database volume.
Changing a compose password after the volume exists does not change the
PostgreSQL role; rotate the role and both DSNs together or recreate the
disposable volume. A fresh own-URL database may need the two roles described
in [the local setup guide](installation.md#your-own-server):
`insideout_owner` is `LOGIN NOSUPERUSER BYPASSRLS` and owns the database and
schema; `insideout_app` is `LOGIN NOSUPERUSER NOBYPASSRLS`. An existing owner
without BYPASSRLS fails during the RLS migrations; repair that role before
retrying setup.

Set [the JWT secret](keys/auth/jwt-secret.md) with `env.sh init` or let guided
setup generate it. Generate a distinct value for each environment.

## 3. Add runtime and compose settings only when needed

The defaults are valid for the common local path. Change a setting with
`./scripts/env.sh edit`, then follow its reference and verification step:

- Runtime: [address](keys/app/addr.md), [cookie security](keys/app/cookie-secure.md),
  [CORS origins](keys/app/cors-origins.md), [development CORS](keys/app/dev-cors.md),
  [access TTL](keys/app/access-ttl.md), and [refresh TTL](keys/app/refresh-ttl.md).
- Compose: [server port](keys/compose/server-port.md),
  [PostgreSQL port](keys/compose/postgres-port.md),
  [client port](keys/compose/client-port.md), and
  [superuser bootstrap password](keys/compose/postgres-superuser-password.md).

Do not place database passwords in command arguments. The compose password
settings apply to a fresh volume; the DSNs must carry matching role passwords.

## 4. Choose the coach mode

An empty LLM key is a supported offline lane. For a provider-backed coach,
configure all six related settings together:

- [API key](keys/llm/llm-api-key.md)
- [base URL](keys/llm/llm-base-url.md)
- [model](keys/llm/llm-model.md)
- [wire schema](keys/llm/llm-schema.md)
- [maximum concurrency](keys/llm/ai-max-concurrent.md)
- [queue wait](keys/llm/ai-queue-wait.md)

The base URL must match the provider adapter's path convention. Follow the
per-key reference rather than guessing whether a `/v1` segment belongs in the
value. A shape check is not a successful coach turn; use the testing tutorial.

## 5. Configure mail when auth email must leave the server

Mail dev mode is valid for local development and logs a safe development
status. For delivery, configure the [mail API key](keys/mail/mail-api-key.md),
[verified sender](keys/mail/mail-from.md), and [public URL](keys/mail/public-url.md).
The optional [development body switch](keys/mail/mail-dev-body.md) is for
local debugging only. The sender domain and public URL must agree before a
real auth link can work.

## 6. Configure GitHub evidence access

For the evidence loop, store the GitHub App credentials together:

- [App ID](keys/github/gh-app-id.md)
- [App private key](keys/github/gh-app-private-key.md)
- [private-key file pointer](keys/github/gh-private-key-file.md) when using
  the file form
- [webhook secret](keys/github/gh-webhook-secret.md)
- [optional GitHub token](keys/github/github-token.md)

[The OAuth client secret](keys/github/gh-client-secret.md) is retained for a
future invite-link consumer and has placement-only verification today.

After the App is installed, bind a project and follow the
[CLI guidance guide](cli-mcp-and-guidance.md). The testing tutorial separates
manual token sync from signed App webhook delivery so each credential is proved
by the consumer that actually reads it.

## 7. Record the host key store

Once `.env` is valid, initialize and fill the host-specific placeholders:

```bash
export KEY_HOST='your-host-name'
INSIDEOUT_KEYS_HOST="$KEY_HOST" python3 scripts/keys.py init
INSIDEOUT_KEYS_HOST="$KEY_HOST" python3 scripts/keys.py fill
INSIDEOUT_KEYS_HOST="$KEY_HOST" python3 scripts/keys.py audit
```

The key files are ignored, mode `0600`, and never printed. The store is a
recovery copy; `.env` remains the live consumer. Read the
[key reference index](keys/README.md) before rotating or moving a value.

## 8. Finish the server setup

Run the guided setup or the normal development wrapper in one terminal, then
use a second terminal for the checks. Point the CLI at the local API so it
does not continue probing the hosted service:

```bash
./scripts/dev.sh -C server go run ./cmd/insideout
```

In a second terminal:

```bash
export INSIDEOUT_API=http://127.0.0.1:8080/api/v1
./scripts/env.sh check server --db
insideout setup --check
```

The final server check separates API reachability, database connectivity,
GitHub token validity, and LLM key shape. It does not replace the live probes
in the [testing tutorial](key-testing-tutorial.md).

## If a step fails

Start with `./scripts/env.sh check server`; it names the missing or malformed
setting without printing it. Then open the referenced key document and use its
failure-specific remedy. A successful process boot with a warning about an
unusable key is still a failed setup; prove the capability the key enables.
