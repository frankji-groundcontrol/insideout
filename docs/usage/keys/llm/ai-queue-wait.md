# ai-queue-wait

`INSIDEOUT_AI_QUEUE_WAIT` — how long a coach request may wait for a
semaphore slot (Go duration). Config companion: blank = default
`15s`; the file exists so the kind is always findable.

## 1. Where the `.key` file is

`hosts/<host>/keys/llm/.secrets/ai-queue-wait.key` (manifest
`llm/coach.yaml`).

## 2. How it is propagated

`.env` → `dev.sh` export → the coach's queue admission; deployed
value in the provider's variable store.

## 3. How to set it

A Go duration string (`15s`, `30s`). Blank = `15s`; garbage silently
falls back to the default.

## 4. How to test it

`scripts/tests/keys/llm/test_ai-queue-wait.py` (exists + duration
shape when non-empty). Live: with a 1-slot semaphore, a second
concurrent call beyond the wait fails fast.

## 5. How it is used

Bounds how long a request queues before giving up, pairing with
[ai-max-concurrent](ai-max-concurrent.md) as backpressure.

## 6. Installation guide

This file is the detail — [SETENV.md](../../../SETENV.md) Step 3
links here. Contract row in [environment.md](../../environment.md).
