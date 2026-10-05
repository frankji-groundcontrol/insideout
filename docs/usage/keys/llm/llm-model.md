# llm-model

`INSIDEOUT_LLM_MODEL` — the model id the coach requests. Config
companion of [llm-api-key](llm-api-key.md).

## 1. Where the `.key` file is

`hosts/<host>/keys/llm/.secrets/llm-model.key` (manifest
`llm/provider.yaml`).

## 2. How it is propagated

`.env` → `dev.sh` export → every model request body; deployed value in
the provider's variable store. Catalog default:
`claude-sonnet-4-20250514`.

## 3. How to set it

Exactly the id your endpoint serves (relay aliases included). Discover
before assuming an id works — a 404/400 on the provider ping is the
signal of a wrong id.

## 4. How to test it

The provider ping with this id → 200 (rung 2 of the
[llm-api-key](llm-api-key.md) ladder); a real coach reply proves it
app-level.

## 5. How it is used

Names the model in every coach call; offline mode ignores it.

## 6. Installation guide

This file is the detail — [SETENV.md](../../../SETENV.md) Step 3
links here. Contract row in [environment.md](../../environment.md).
