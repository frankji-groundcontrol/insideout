# A localhost API default makes a finished install unusable

## What was learned

When the product a person just installed is an online service, the
client's default API has to be that service. Defaulting to
`127.0.0.1` makes the first command fail before any account exists,
which reads as a broken install.

The installer has to say which API it will call. A silent hosted
default is how a clean checkout used to write to production without
the operator noticing.

## Evidence

After `./scripts/install.sh` on 2026-10-06, `insideout whoami` with no
`INSIDEOUT_API` dialed `127.0.0.1:8080` and got connection refused.
The public app's `/api/v1/me` answered 401 `authentication required`
in the same session. After the default moved to that hosted API,
`insideout whoami` returned unauthorized instead of a refused
connection
([changelog](../changelogs/2026-10-06-install-hosted-default.md)).

The 2026-10-04 remediation had switched the default to localhost so a
checkout would not touch production quietly
([changelog](../changelogs/2026-10-04-remediation-wave-0-1.md)). The
install script now prints the URL it will use.

## Scope

CLI and MCP clients of a hosted product. Not the server process,
which still requires `DATABASE_URL` on the machine that runs it.

## When to apply it again

Before calling an installer done, run one read command with a clean
environment. If it cannot reach the service a new user is supposed to
use, the default is wrong. Print the target in the installer output.
