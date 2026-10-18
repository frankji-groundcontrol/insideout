# llm-model

`INSIDEOUT_LLM_MODEL` is the model id sent in every provider-backed coach
request. Blank means `claude-sonnet-4-20250514`.

## 1. Where the `.key` file is

`hosts/<host>/keys/llm/.secrets/llm-model.key` (manifest
`llm/provider.yaml`).

## 2. How it is propagated

The value moves through `.env` and `dev.sh` into each request body. The model
id is not validated by the shape suite against a provider catalog.

## 3. How to set it

1. Choose an exact model id shown by the provider or relay for the configured
   base URL and schema. Keep punctuation and case unchanged.
2. Run `./scripts/env.sh edit`, choose `INSIDEOUT_LLM_MODEL`, and enter the id
   at the hidden prompt.
3. Initialize/fill the host key store:

   ```bash
   INSIDEOUT_KEYS_HOST='<host>' python3 scripts/keys.py init
   INSIDEOUT_KEYS_HOST='<host>' python3 scripts/keys.py fill
   ```

## 4. How to test it

`insideout setup --check --llm` is a legacy compatibility probe and does not
prove that this model id is served by the current `messages` or `responses`
configuration. Complete one real coach turn through the app or the provider
leg of the smoke flow.

**Expected result:** the provider accepts the selected model and the SSE stream
reaches `message_end` with assistant deltas.

**If it fails:** a provider 400/404 naming the model means the id is wrong for
that endpoint, not necessarily that the key is invalid. Select an id from the
provider's model list, verify the schema and base URL, and retry without
printing the request or response body.

The automated placement check is:

```bash
cd scripts && python3 -m unittest discover -s tests/keys -t . -p 'test_llm-model.py' -v
```

## 5. How it is used

The coach sends this id to the provider for every clarify, draft, critique, and
finalize turn. Offline mode ignores it.

## 6. Installation guide

Follow the [coach setup tutorial](../../key-setup-tutorial.md#4-choose-the-coach-mode),
the [coach testing tutorial](../../key-testing-tutorial.md#5-prove-the-coach-configuration),
and the [health-check section](../../installation.md#re-verify-any-time).
