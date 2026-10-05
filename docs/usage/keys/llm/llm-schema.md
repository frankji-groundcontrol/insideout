# llm-schema

`INSIDEOUT_LLM_SCHEMA` — the wire format of the coach's model calls.
Config companion of [llm-api-key](llm-api-key.md).

## 1. Where the `.key` file is

`hosts/<host>/keys/llm/.secrets/llm-schema.key` (manifest
`llm/provider.yaml`).

## 2. How it is propagated

`.env` → `dev.sh` export → the coach's client construction; deployed
value in the provider's variable store.

## 3. How to set it

Exactly one of `messages` (Anthropic wire: `x-api-key` header,
`/v1/messages`) or `responses` (OpenAI wire: `Authorization: Bearer`,
`/v1/responses`). It must match what `llm-base-url` actually serves —
a mismatch shows as 404s on model calls.

## 4. How to test it

The provider ping uses the schema's URL and header; 200 proves the
choice matches the endpoint.

## 5. How it is used

Selects request/response codecs for every coach model call; offline
mode ignores it.

## 6. Installation guide

This file is the detail — [SETENV.md](../../../SETENV.md) Step 3
links here. Contract row in [environment.md](../../environment.md).
