# gh-app-id

`INSIDEOUT_GH_APP_ID` is the numeric id of the GitHub App that signs
installation-token requests for private guide and evidence fetches.

## 1. Where the `.key` file is

`hosts/<host>/keys/github/.secrets/gh-app-id.key` (manifest
`github/app.yaml`; the matching credentials are
[gh-app-private-key](gh-app-private-key.md) and
[gh-private-key-file](gh-private-key-file.md)).

## 2. How it is propagated

The value moves from `.env` through `dev.sh` into `config.Load`. The server
uses it only when a signed GitHub App delivery carries an installation id. It
does not authenticate the user-triggered `insideout sync` command; that route
uses `GITHUB_TOKEN` when one is configured, or GitHub's public API lane.

## 3. How to set it

1. In the GitHub App settings page, copy the numeric **App ID**. Do not copy
   the client id or installation id.
2. Run `./scripts/env.sh edit`, choose `INSIDEOUT_GH_APP_ID`, and enter the
   number at the hidden prompt. The value is never printed.
3. Initialize and fill the host store from the repository root:

   ```bash
   INSIDEOUT_KEYS_HOST='<host>' python3 scripts/keys.py init
   INSIDEOUT_KEYS_HOST='<host>' python3 scripts/keys.py fill
   ```

Keep the App ID paired with the private key from the same GitHub App. A key
rotation is incomplete until both values and the deployment are updated.
If this deployment does not use GitHub App evidence, leave the value blank;
the initialized empty placeholder is the supported disabled lane.

## 4. How to test it

Install the App on a private test repository, configure a signed webhook, and
link that repository to a project. Redeliver a `push` or `pull_request` event
from the GitHub App delivery page. The server must answer HTTP 200; when the
repository contains a matching `insideout.yaml` and project leaf, the response
reports a non-zero `evidence` count. That evidence fetch is the path that uses
the App ID and PEM together. A `ping` delivery tests the webhook secret only.

When the App integration is disabled, run
`INSIDEOUT_KEYS_HOST='<host>' python3 scripts/keys.py audit` instead; `key=empty`
is the expected result and there is no live App delivery to prove.

**Expected result:** the enabled lane acknowledges the delivery and attaches
private guide evidence when the guide and event match. The disabled lane keeps
an empty `0600` placeholder and skips the provider walk.

**If it fails:** a zero evidence count can mean there is no matching guide or
leaf. A server warning about an unusable private key, or an installation-token
error in the delivery log, means the App ID and PEM are mismatched, the App is
not installed on the repository, or its permissions do not include the needed
repository contents. Correct the GitHub App configuration, then redeliver the
same event without putting any credential in a shell argument.

The automated placement check is
`cd scripts && python3 -m unittest discover -s tests/keys -t . -p 'test_gh-app-id.py' -v`.

## 5. How it is used

The id becomes the `iss` claim of the short-lived App JWT used to mint an
installation access token. It has no meaning without the matching RSA PEM.

## 6. Installation guide

Use the [guided GitHub setup](../../key-setup-tutorial.md#6-configure-github-evidence-access),
the [GitHub verification tutorial](../../key-testing-tutorial.md#7-prove-github-access),
and the [own-server installation path](../../installation.md#your-own-server).
The recurring checkpoint is [installation health checks](../../installation.md#re-verify-any-time).
