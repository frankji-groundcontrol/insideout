# llm-base-url

`INSIDEOUT_LLM_BASE_URL` is the provider API root used by the coach. It is a
config companion of [llm-api-key](llm-api-key.md).

## 1. Where the `.key` file is

`hosts/<host>/keys/llm/.secrets/llm-base-url.key` (manifest
`llm/provider.yaml`). Blank means the catalog default
`https://api.anthropic.com/v1`.

## 2. How it is propagated

The value moves through `.env` and `dev.sh` into the coach HTTP client. The
server appends `/messages` for the `messages` schema or `/responses` for the
`responses` schema. It never inserts `/v1` for you.

## 3. How to set it

1. Copy the API root from the provider or relay documentation. Include the
   `/v1` segment when that provider requires it; do not append `/messages`,
   `/responses`, or `/models` yourself.
2. Run `./scripts/env.sh edit`, choose `INSIDEOUT_LLM_BASE_URL`, and enter the
   absolute `http://` or `https://` URL.
3. Initialize/fill the host key store:

   ```bash
   INSIDEOUT_KEYS_HOST='<host>' python3 scripts/keys.py init
   INSIDEOUT_KEYS_HOST='<host>' python3 scripts/keys.py fill
   ```

## 4. How to test it

`insideout setup --check --llm` is a legacy compatibility probe for a schema
named `openai`; current `messages` and `responses` configurations report a
skipped live probe. Use one real coach turn for this setting instead. The
turn proves the configured base plus `/messages` or `/responses` path works.

**Expected result:** the provider responds at the exact base-plus-path that the
configured schema selects, and the coach stream ends with `message_end`.

**If it fails:** a connection failure means the URL is unreachable; a 404
usually means a missing or duplicated `/v1` or the wrong schema path. Correct
the base only, then repeat the coach turn. A relay or tunnel can transiently
refuse a connection, so retry before rotating a valid key.

The automated placement check is:

```bash
cd scripts && python3 -m unittest discover -s tests/keys -t . -p 'test_llm-base-url.py' -v
```

## 5. How it is used

Every provider-backed coach turn targets this base URL. Offline mode ignores
it because no network request is made.

## 6. Installation guide

Follow the [coach setup tutorial](../../key-setup-tutorial.md#4-choose-the-coach-mode),
the [coach testing tutorial](../../key-testing-tutorial.md#5-prove-the-coach-configuration),
and the [own-server installation path](../../installation.md#your-own-server).
