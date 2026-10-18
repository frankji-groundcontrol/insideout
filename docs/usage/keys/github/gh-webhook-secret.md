# gh-webhook-secret

`INSIDEOUT_GH_WEBHOOK_SECRET` is the HMAC-SHA256 shared secret for GitHub App
deliveries.

## 1. Where the `.key` file is

`hosts/<host>/keys/github/.secrets/gh-webhook-secret.key` (manifest
`github/webhooks.yaml`).

## 2. How it is propagated

The same value has two homes: `.env` for the server's
`POST /api/v1/hooks/github` receiver, and the GitHub App webhook settings. The
receiver checks `X-Hub-Signature-256` before it parses a delivery.

## 3. How to set it

1. Generate a random value in a password manager or the GitHub App's secret
   generator. Do not use a command that prints the value into a shared shell
   transcript.
2. In the GitHub App settings, set the webhook URL to the deployment's public
   origin followed by `/api/v1/hooks/github`, then paste the generated value
   into the **Secret** field.
3. Run `./scripts/env.sh edit`, set `INSIDEOUT_GH_WEBHOOK_SECRET`, and then
   initialize/fill the host key store as described in the module README.

Use one secret per deployment. Rotating it requires updating GitHub and the
server before the next delivery. If this deployment does not receive GitHub
App deliveries, leave the value blank; the initialized empty file is the
supported disabled lane.

## 4. How to test it

Use **Redeliver** on a GitHub App `ping` delivery. GitHub signs the request;
the receiver must answer HTTP 200 with an `ok` response. As a negative check,
an unsigned request to the same endpoint must answer HTTP 401; that negative
check proves the gate is enabled but cannot replace a signed delivery.
When webhooks are disabled, run
`INSIDEOUT_KEYS_HOST='<host>' python3 scripts/keys.py audit` and accept
`key=empty` for this optional kind instead.

**Expected result:** the signed ping is 200. For a `push`, `pull_request`,
`deployment`, or `deployment_status` delivery linked to a project, the 200
response also reports the matched repository and any evidence count.
With the integration disabled, `key=empty` is expected and no signed delivery
is attempted.

**If it fails:** check the endpoint path, deployment reachability, and that the
GitHub secret and `.env` value are byte-for-byte equal. HTTP 401 means a
signature mismatch; HTTP 503 means the server has no webhook secret configured.
Do not paste either value into a command or an issue.

The automated placement check is:

```bash
cd scripts && python3 -m unittest discover -s tests/keys -t . -p 'test_gh-webhook-secret.py' -v
```

## 5. How it is used

The receiver rejects unsigned or mismatched bodies before any repository or
database work. Valid deliveries are acknowledged and then feed the evidence
loop.

## 6. Installation guide

Follow the [guided GitHub setup](../../key-setup-tutorial.md#6-configure-github-evidence-access),
the [GitHub verification tutorial](../../key-testing-tutorial.md#7-prove-github-access),
and the [own-server installation path](../../installation.md#your-own-server).
