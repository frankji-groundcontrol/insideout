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

From your model provider's console. **Empty is a valid choice**: it
selects the offline template-reply coach, which is the default local
dev experience. One key per provider account; rotation is a plain
paste-over in `.env` + provider store.

## 4. How to test it

Boot the server and watch the log: the
`INSIDEOUT_LLM_API_KEY not set — using offline template-reply coach`
notice must **not** appear; then one coach reply in a real
conversation proves the key and the model id together.

## 5. How it is used

Authenticates the coach's model calls (clarify/draft/critique/
finalize turns). Without it the coach answers from templates — honest,
offline, and never a crash.

## 6. Installation guide

[SETENV.md](../../../SETENV.md) Step 3 — "a choice, not a gap".
