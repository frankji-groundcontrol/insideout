# Practice: Discover provider models before assuming a model id works

**Date**: 2026-07-21

## Trigger

Configuring or changing `AI_MODEL`, pointing `ANTHROPIC_BASE_URL` at a new endpoint (proxy, gateway, or the provider directly), or debugging AI-path errors that don't reproduce elsewhere. Proxies and gateways serve **different model sets** than the upstream provider — a model id that is valid at Anthropic may not exist behind a given gateway, and vice versa.

## Sequence / guardrail

1. List what the endpoint actually serves:
   ```bash
   curl -s "$ANTHROPIC_BASE_URL/v1/models" \
     -H "x-api-key: $ANTHROPIC_AUTH_TOKEN" \
     -H "anthropic-version: 2023-06-01" | jq '.data[].id'
   ```
2. Pick a model id from that list and smoke-test one real request against the same endpoint the app will use:
   ```bash
   curl -s "$ANTHROPIC_BASE_URL/v1/messages" \
     -H "x-api-key: $ANTHROPIC_AUTH_TOKEN" \
     -H "anthropic-version: 2023-06-01" \
     -H 'Content-Type: application/json' \
     -d '{"model":"<id from step 1>","max_tokens":64,"messages":[{"role":"user","content":"ping"}]}'
   ```
   Inspect the raw response shape too — the same model-discovery `curl` work during BUG-009 revealed that this account's model emits `thinking` content blocks, which is what broke the old client.
3. Only then set `AI_MODEL` (default `claude-sonnet-4-20250514` in `server/internal/config/config.go`) and exercise the app path.

## Verification

Step 2 returns a 200 with a real `content` block from the exact base URL + token + model triple the server is configured with. Never validate against a different endpoint than the app uses.

## Failure signals

- `model_not_found: Model "..." is not supported...` — the id exists upstream but not on this gateway (or is misspelled). The rewritten client (`server/internal/agent/anthropic.go`) now surfaces this provider message; the old langchaingo path reduced it to a generic `no response`.
- Errors or empty responses **only in the app path** while ad-hoc requests to a different endpoint work — you validated against the wrong endpoint.
- Responses parse but content is empty — the model's response shape (e.g. extended thinking) differs from what the client expects; compare against the raw smoke-test output.

## Related

- [BUG-009: langchaingo removed; model discovery via direct curl](../issues/2026-07-21-bug-009-langchaingo-removed.md)
- [Live-exercise streaming endpoints](2026-07-21-live-exercise-streaming-endpoints.md)
- [Learning records](../learning/README.md)
