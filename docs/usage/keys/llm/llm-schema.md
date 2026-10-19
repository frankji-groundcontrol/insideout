# llm-schema

`INSIDEOUT_LLM_SCHEMA` selects the wire protocol used for provider-backed coach
calls: `messages` or `responses`. Blank means `messages`.

## 1. Where the `.key` file is

`hosts/<host>/keys/llm/.secrets/llm-schema.key` (manifest
`llm/provider.yaml`).

## 2. How it is propagated

The value moves through `.env` and `dev.sh` into the coach streamer. The
server appends `/messages` and sends `x-api-key` for `messages`; it appends
`/responses` and sends `Authorization: Bearer` for `responses`.

## 3. How to set it

1. Choose `messages` for an Anthropic Messages endpoint, or `responses` for an
   OpenAI-compatible Responses endpoint. Match the provider documentation and
   the base URL; do not infer one from the model name.
2. Run `./scripts/env.sh edit`, choose `INSIDEOUT_LLM_SCHEMA`, and enter exactly
   `messages` or `responses`.
3. Initialize/fill the host key store:

   ```bash
   INSIDEOUT_KEYS_HOST='<host>' python3 scripts/keys.py init
   INSIDEOUT_KEYS_HOST='<host>' python3 scripts/keys.py fill
   ```

## 4. How to test it

`insideout setup --check --llm` is a legacy compatibility probe for a schema
named `openai`; both current `messages` and `responses` values report a
skipped live probe. Finish with one real coach turn for either lane.

**Expected result:** the selected endpoint accepts the matching header and
request shape, and the SSE stream reaches `message_end`.

**If it fails:** 404/400 errors usually indicate that the schema and base URL
belong to different provider protocols. Correct them as a pair, restart the
server, and repeat the app-level turn. A skipped `messages` probe is expected;
it is not a failure or a successful live test.

The server also checks that a reply is whole: it takes a turn as complete only
after the stream's closing event (`message_stop` on `messages`;
`response.completed` or `response.incomplete` on `responses`). A gateway or
proxy that cuts or never sends that event makes every turn fail with a
transient provider error ("stream ended before message_stop" or "before
response.completed"), and one that rejects the `parallel_tool_calls` field on
`responses` answers 400, shown as a provider-configuration error. Both are
settings or gateway faults, not model faults: point the base URL at the
provider's own endpoint or one that relays the closing event, and repeat the
turn.

The automated placement check is:

```bash
cd scripts && python3 -m unittest discover -s tests/keys -t . -p 'test_llm-schema.py' -v
```

## 5. How it is used

It chooses the request codec, endpoint suffix, and authentication header for
every provider-backed coach call. Offline mode ignores it.

## 6. Installation guide

Follow the [coach setup tutorial](../../key-setup-tutorial.md#4-choose-the-coach-mode),
the [coach testing tutorial](../../key-testing-tutorial.md#5-prove-the-coach-configuration),
and the [own-server installation path](../../installation.md#your-own-server).
