# Learning Records

Distilled, reusable lessons extracted from the bug book ([../issues/](../issues/)).
Each record is one focused file: what was learned, the evidence, the scope,
and when to apply it again.

| Date | Record | Source |
|------|--------|--------|
| 2026-10-07 | [The own-URL path's hidden precondition: provisioned roles with owner BYPASSRLS](2026-10-07-own-url-role-provisioning-precondition.md) | [setup TUI slice 2](../changelogs/2026-10-07-setup-tui-slice-2.md) |
| 2026-10-07 | [Parallel slots on a dirty worktree can revert unrelated uncommitted lines](2026-10-07-parallel-slots-dirty-worktree.md) | [setup TUI slice 1](../changelogs/2026-10-07-setup-tui-slice-1.md) |
| 2026-10-07 | [A workflow gate must use absolute paths: world.run's cwd is not guaranteed to be the repo root](2026-10-07-workflow-gates-need-absolute-paths.md) | [setup TUI workflow](../changelogs/2026-10-07-setup-tui-parallel-workflows.md) |
| 2026-10-07 | [x/term ReadPassword already consumes the trailing newline](2026-10-07-xterm-readpassword-consumes-newline.md) | [setup TUI workflow](../changelogs/2026-10-07-setup-tui-parallel-workflows.md) |
| 2026-10-07 | [go mod tidy silently strips newly added but not-yet-imported deps](2026-10-07-go-mod-tidy-strips-unused-new-deps.md) | [setup TUI plan deps step](../plans/2026-10-07-setup-tui.md) |
| 2026-10-06 | [Relative links are depth-sensitive: resolve them mechanically](2026-10-06-relative-links-are-depth-sensitive.md) | [install-skill changelog](../changelogs/2026-10-06-install-skill.md) |
| 2026-10-06 | [A localhost API default makes a finished install unusable](2026-10-06-localhost-default-makes-install-unusable.md) | [install hosted default](../changelogs/2026-10-06-install-hosted-default.md) |
| 2026-10-06 | [A statement timestamp cannot order a batch](2026-10-06-a-statement-timestamp-cannot-order-a-batch.md) | [BUG-019](../issues/2026-10-06-bug-019-applied-proposal-reorders-items.md) |
| 2026-10-06 | [A deadline must bound the await, not just the loop around it](2026-10-06-a-deadline-must-bound-the-await.md) | [BUG-016](../issues/2026-10-06-bug-016-web-boot-gate-hangs-on-font-load.md) |
| 2026-10-06 | [A blank placeholder is only legal if metadata says so — declare the lane first](2026-10-06-blank-lane-legality-lives-in-metadata.md) | [config-companion-keys changelog](../changelogs/2026-10-06-config-companion-keys.md) |
| 2026-10-05 | [A boot-time warning is not a pass — grep for silent degradations](2026-10-05-boot-warning-is-not-a-pass.md) | [buddy-host-keys changelog](../changelogs/2026-10-05-buddy-host-keys.md) |
| 2026-10-05 | [Skip-green and phantom-done](2026-10-05-skip-green-and-phantom-done.md) | [planning-flow-remediation plan](../plans/2026-10-05-planning-flow-remediation/2026-10-05-planning-flow-remediation.md) |
| 2026-10-05 | [Port a reference scheme structurally, not by approximation](2026-10-05-port-reference-schemes-structurally.md) | [buddy-host-keys changelog](../changelogs/2026-10-05-buddy-host-keys.md) |
| 2026-10-04 | [Go test census: `=== RUN` lines double-count subtests](2026-10-04-go-test-census-counts-subtests.md) | [multirole review critic §4](../review/2026-10-04-multirole/30-completeness-critic.md) |
| 2026-07-21 | [RLS + row locking: EvalPlanQual silently drops rows](2026-07-21-rls-row-locking-evalplanqual.md) | [BUG-007](../issues/2026-07-20-bug-007-rls-against-real-postgres.md) |
| 2026-07-21 | [SECURITY DEFINER, same-owner FORCE RLS, and policy merging](2026-07-21-security-definer-same-owner-rls.md) | [BUG-007](../issues/2026-07-20-bug-007-rls-against-real-postgres.md) |
| 2026-07-21 | [pgx simple protocol: jsonb parameters need string + explicit cast](2026-07-21-pgx-simple-protocol-jsonb.md) | [BUG-007](../issues/2026-07-20-bug-007-rls-against-real-postgres.md) |
| 2026-07-21 | [Unmaintained LLM abstraction libraries: own a small direct client](2026-07-21-unmaintained-llm-abstraction-libraries.md) | [BUG-009](../issues/2026-07-21-bug-009-langchaingo-removed.md) |
| 2026-07-21 | [ResponseWriter wrapping drops http.Flusher](2026-07-21-response-writer-wrapping-drops-flusher.md) | [BUG-010](../issues/2026-07-21-bug-010-sse-flusher-swallowed-by-logging-middleware.md) |
| 2026-07-26 | [Nuxt dev server: IPv4 clients get a bare 426 (IPv6/IPv4 bind split)](2026-07-26-nuxt-dev-ipv6-426.md) | [collab T10 verification](../changelogs/2026-07-26-roadmap-canvas-collab-attribution.md) |
| 2026-07-26 | [Nuxt routing: a `pages/x/[id].vue` without `<NuxtPage />` silently swallows `pages/x/[id]/*` children](2026-07-26-nuxt-dynamic-route-parent-shadowing.md) | [collab T9 changelog](../changelogs/2026-07-26-roadmap-canvas-workstream-d.md) |
| 2026-08-23 | [dio on Flutter web does not stream SSE — fetch + ReadableStream does](2026-08-23-dio-web-does-not-stream-sse.md) | [BUG-014](../issues/2026-08-23-bug-014-dio-web-sse-never-streams.md) |
| 2026-08-23 | [Browser font-readiness signals race the stylesheet parse](2026-08-23-browser-font-readiness-signals-race-stylesheet-parse.md) | [font-flash changelog](../changelogs/2026-08-23-web-font-flash.md) |
| 2026-08-25 | [uuid.Nil does not mean NULL — nullable FK columns reject the zero UUID](2026-08-25-uuid-nil-violates-nullable-fk.md) | [invitation-only changelog](../changelogs/2026-08-25-invitation-only-signup.md) |
| 2026-08-24 | [Gated integration tests must derive unique keys per run — rows persist](2026-08-24-gated-integration-tests-rerun-collisions.md) (+ settings restore follow-up, 08-25)(2026-08-24-gated-integration-tests-rerun-collisions.md) | [email-auth changelog](../changelogs/2026-08-24-email-auth.md) |
| 2026-08-24 | [Interactive CLI prompts must share one buffered stdin reader](2026-08-24-cli-prompts-share-one-stdin-reader.md) | [auth-parity changelog](../changelogs/2026-08-24-auth-parity-cli-mcp.md) |
| 2026-08-24 | [This machine's sandbox can block exec of freshly built binaries](2026-08-24-sandbox-blocks-new-binary-exec.md) | [auth-parity changelog](../changelogs/2026-08-24-auth-parity-cli-mcp.md) |
