# LLM provider key

`INSIDEOUT_LLM_API_KEY` — the PRD coach's model key.

## 1. Where the `.key` file is

`hosts/<host>/keys/llm/.secrets/llm-api-key.key` (manifest
`llm/provider.yaml`).

## 2. How it is propagated

`.env` → `dev.sh` export → the coach at conversation time; the
deployed value rides the provider's variable store. Companion config
(not credentials, so `.env.example` material): `INSIDEOUT_LLM_BASE_URL`
(any `/v1` suffix is yours to add), `INSIDEOUT_LLM_MODEL`,
`INSIDEOUT_LLM_SCHEMA` (`messages` | `responses`).

## 3. How to set it

The key comes from your model provider's console — but it is one of
**four values that must agree**: `INSIDEOUT_LLM_API_KEY` +
`INSIDEOUT_LLM_BASE_URL` + `INSIDEOUT_LLM_MODEL` +
`INSIDEOUT_LLM_SCHEMA` (`messages` | `responses`, i.e. Anthropic- vs
OpenAI-wire). A relay/proxy base URL is fine; put any `/v1` suffix on
it yourself. Set all four in the same `.env` (they are catalog config,
not store credentials — only the key is). **Empty key is a valid
choice**: it selects the offline template-reply coach, the default
local dev experience. Rotation is a plain paste-over in `.env` +
provider store.

## 4. How to test it

Three rungs, each proving more:

1. **Key is read** — boot the server: the
   `INSIDEOUT_LLM_API_KEY not set — using offline template-reply
   coach` notice must **not** appear.
2. **Key + endpoint + format + model agree** — an authenticated
   provider ping, status-only (never print the key or body): POST
   `$BASE_URL/v1/responses` with `Authorization: Bearer` for schema
   `responses` (or `$BASE_URL/v1/messages` with `x-api-key` for
   `messages`) and the real model id → expect 200.
3. **App-level** — one coach reply in a real conversation.

Caveat learned the hard way: a relay behind a tunnel (ngrok and
friends) transiently refuses connections — a `000`/connection-failure
on rung 2 means retry before blaming the key.

## 5. How it is used

Authenticates the coach's model calls (clarify/draft/critique/
finalize turns). Without it the coach answers from templates — honest,
offline, and never a crash.

## 6. Installation guide

This file is the detail — [SETENV.md](../../../SETENV.md) Step 3
("a choice, not a gap") links here. Contract rows for the key and its
`INSIDEOUT_LLM_*` companions are in
[environment.md](../../environment.md).
