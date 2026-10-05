# ai-max-concurrent

`INSIDEOUT_AI_MAX_CONCURRENT` — parallel coach model requests.
Config companion: blank = default `4`; the file exists so the kind is
always findable.

## 1. Where the `.key` file is

`hosts/<host>/keys/llm/.secrets/ai-max-concurrent.key` (manifest
`llm/coach.yaml`).

## 2. How it is propagated

`.env` → `dev.sh` export → the coach's semaphore construction;
deployed value in the provider's variable store.

## 3. How to set it

A positive integer (`4`, `8`). Blank = `4`; zero/negative or
non-numeric silently falls back to the default.

## 4. How to test it

`scripts/tests/keys/llm/test_ai-max-concurrent.py` (exists + positive
integer when non-empty). Live: fire N parallel coach calls and watch
at most N in flight in server logs.

## 5. How it is used

Caps concurrent provider calls so one burst of sessions cannot
exhaust the [llm-api-key](llm-api-key.md) rate limit.

## 6. Installation guide

This file is the detail — [SETENV.md](../../../SETENV.md) Step 3
links here. Contract row in [environment.md](../../environment.md).
