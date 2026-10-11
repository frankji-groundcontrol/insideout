# Keys — llm module

Per-key reference for the `llm/` module of the key store. Each file
documents one key kind with the six parts (where the `.key` file is,
propagation, setting, testing, use, and the installation-guide
pointer). The module table, shared flow, and per-host state live in
the [keys README](../README.md).

- [llm-api-key.md](llm-api-key.md) — `INSIDEOUT_LLM_API_KEY`
  (manifest `llm/provider.yaml`)
- [llm-base-url.md](llm-base-url.md) — `INSIDEOUT_LLM_BASE_URL`
  (`llm/provider.yaml`)
- [llm-model.md](llm-model.md) — `INSIDEOUT_LLM_MODEL`
  (`llm/provider.yaml`)
- [llm-schema.md](llm-schema.md) — `INSIDEOUT_LLM_SCHEMA`
  (`llm/provider.yaml`)
- [ai-max-concurrent.md](ai-max-concurrent.md) —
  `INSIDEOUT_AI_MAX_CONCURRENT` (`llm/coach.yaml`)
- [ai-queue-wait.md](ai-queue-wait.md) — `INSIDEOUT_AI_QUEUE_WAIT`
  (`llm/coach.yaml`)
