# 2026-11-21 — Close the gaps the review 5 remediation left open

The owner asked to close every gap the
[review 5 remediation](../2026-11-21-rv5-remediation/2026-11-21-rv5-remediation.md)
left open. They are listed in its
[changelog](../../changelogs/2026-11-21-rv5-remediation/index.md). The four
gaps share no file, so four clusters run in parallel from `main` at `cecb0cb`.
The coordinator reviews each diff and runs the full gates once on the
integrated branch `rv5/close-gaps`. That branch goes to `main` in one PR with
`[skip ci]` (CI is disabled in the repository).

## Checklist

- [ ] api-errors: every remaining store-error ladder in `server/internal/api`
      uses `writeStoreError`
      ([record](../../issues/2026-11-21-store-error-ladders.md)); the three
      long API handlers are split.
- [ ] long-funcs: the five long store, setup and LLM functions are split; the
      function guard's list is empty
      ([record](../../issues/2026-11-21-long-functions.md)).
- [ ] key-manifests: one shared manifest set under `hosts/_manifests/keys/`;
      the drift between the manifests and `.env.example` is settled
      ([record](../../issues/2026-11-21-key-manifests-one-set.md)).
- [ ] parity-nested: the parity walk covers nested request fields and matches
      client calls by receiver type
      ([practice](../../practices/2026-11-21-surface-parity-guards.md)).
- [ ] Integrated gates green; PR merged; records closed.
