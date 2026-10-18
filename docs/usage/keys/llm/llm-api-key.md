# LLM provider key

`INSIDEOUT_LLM_API_KEY` authenticates the PRD coach with the selected model
provider. The key, base URL, model, and wire schema are one configuration
quartet.

## 1. Where the `.key` file is

`hosts/<host>/keys/llm/.secrets/llm-api-key.key` (manifest
`llm/provider.yaml`).

## 2. How it is propagated

The value moves from `.env` through `dev.sh` into the coach streamer. The
provider key is never sent to the browser or printed by setup. The companion
references are [llm-base-url](llm-base-url.md),
[llm-model](llm-model.md), and [llm-schema](llm-schema.md).

## 3. How to set it

1. Create a key in the model provider's console with only the permissions and
   spending limit needed for this environment. Keep the provider's key format
   intact; do not put it in a shell argument.
2. Run `./scripts/env.sh edit`, choose `INSIDEOUT_LLM_API_KEY`, and enter the
   key at the hidden prompt. Set the three companions in the same session.
3. Initialize/fill the host key store:

   ```bash
   INSIDEOUT_KEYS_HOST='<host>' python3 scripts/keys.py init
   INSIDEOUT_KEYS_HOST='<host>' python3 scripts/keys.py fill
   ```

An empty value is a supported offline lane: the server uses the template coach
and does not call a provider. Do not create a key only to silence the offline
notice.

## 4. How to test it

Start with the status-only checks:

```bash
insideout setup --check
insideout setup --check --llm
```

The first command runs the normal API/database/GitHub checks and includes the
LLM key-shape check. The `--llm` request performs an authenticated `GET
{INSIDEOUT_LLM_BASE_URL}/models` only for the legacy `openai` schema. The
current `messages` and `responses` schemas report a skipped live probe; that
is expected and is not a coach proof. Finish with one real coach turn through
the app or the provider leg of `cd server && ./scripts/smoke.sh` (the smoke
flow uses a disposable database account and prints only status markers).

**Expected result:** the server does not log the offline-coach notice, the
provider status check is accepted where supported, and the coach SSE stream
ends with `message_end`.

**If it fails:** distinguish a rejected key (401/403) from a base URL or schema
mismatch (404/400). Confirm the model is served at that endpoint, include any
required `/v1` segment in the base URL, and rerun through `scripts/dev.sh` so
the edited environment is loaded. Never paste a provider response body into a
report.

The automated placement check is:

```bash
cd scripts && python3 -m unittest discover -s tests/keys -t . -p 'test_llm-api-key.py' -v
```

## 5. How it is used

The key authenticates clarify, draft, critique, and finalize model calls. If it
is empty, those turns use the explicit offline template implementation.

## 6. Installation guide

Follow the [coach setup tutorial](../../key-setup-tutorial.md#4-choose-the-coach-mode),
the [coach testing tutorial](../../key-testing-tutorial.md#5-prove-the-coach-configuration),
and the [health-check section](../../installation.md#re-verify-any-time).
