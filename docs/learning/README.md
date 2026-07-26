# Learning Records

Distilled, reusable lessons extracted from the bug book ([../issues/](../issues/)).
Each record is one focused file: what was learned, the evidence, the scope,
and when to apply it again.

| Date | Record | Source |
|------|--------|--------|
| 2026-07-21 | [RLS + row locking: EvalPlanQual silently drops rows](2026-07-21-rls-row-locking-evalplanqual.md) | [BUG-007](../issues/2026-07-20-bug-007-rls-against-real-postgres.md) |
| 2026-07-21 | [SECURITY DEFINER, same-owner FORCE RLS, and policy merging](2026-07-21-security-definer-same-owner-rls.md) | [BUG-007](../issues/2026-07-20-bug-007-rls-against-real-postgres.md) |
| 2026-07-21 | [pgx simple protocol: jsonb parameters need string + explicit cast](2026-07-21-pgx-simple-protocol-jsonb.md) | [BUG-007](../issues/2026-07-20-bug-007-rls-against-real-postgres.md) |
| 2026-07-21 | [Unmaintained LLM abstraction libraries: own a small direct client](2026-07-21-unmaintained-llm-abstraction-libraries.md) | [BUG-009](../issues/2026-07-21-bug-009-langchaingo-removed.md) |
| 2026-07-21 | [ResponseWriter wrapping drops http.Flusher](2026-07-21-response-writer-wrapping-drops-flusher.md) | [BUG-010](../issues/2026-07-21-bug-010-sse-flusher-swallowed-by-logging-middleware.md) |
| 2026-07-26 | [Nuxt dev server: IPv4 clients get a bare 426 (IPv6/IPv4 bind split)](2026-07-26-nuxt-dev-ipv6-426.md) | [collab T10 verification](../changelogs/2026-07-26-roadmap-canvas-collab-attribution.md) |
| 2026-07-26 | [Nuxt routing: a `pages/x/[id].vue` without `<NuxtPage />` silently swallows `pages/x/[id]/*` children](2026-07-26-nuxt-dynamic-route-parent-shadowing.md) | [collab T9 changelog](../changelogs/2026-07-26-roadmap-canvas-workstream-d.md) |
