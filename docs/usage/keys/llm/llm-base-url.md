# llm-base-url

`INSIDEOUT_LLM_BASE_URL` — the coach's model endpoint. Config
companion of [llm-api-key](llm-api-key.md); the quartet must agree.

## 1. Where the `.key` file is

`hosts/<host>/keys/llm/.secrets/llm-base-url.key` (manifest
`llm/provider.yaml`). Non-secret, but store-backed so the override
survives a `.env` loss.

## 2. How it is propagated

`.env` → `dev.sh` export → the coach's HTTP client; deployed value in
the provider's variable store. The catalog (`.env.example`) documents
the default; this file records the override.

## 3. How to set it

Your provider's or relay's API root, **without** `/v1` — put any `/v1`
on it yourself (the server appends the per-schema path). Relays and
tunnels are fine; a tunnel-backed base intermittently refuses
connections (see part 4).

## 4. How to test it

The provider ping, status-only: `POST $BASE/v1/responses` (or
`/v1/messages`) with the real key and model → 200. A `000` on a relay
behind ngrok means retry, not a wrong URL.

## 5. How it is used

Every coach model call (clarify/draft/critique/finalize) targets it.

## 6. Installation guide

This file is the detail — [SETENV.md](../../../SETENV.md) Step 3
links here. Contract row in [environment.md](../../environment.md).
