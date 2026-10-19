# Learning Records

Distilled, reusable lessons extracted from the bug book ([../issues/](../issues/)).
Each record is one focused file: what was learned, the evidence, the scope,
and when to apply it again.

| Date | Record | Source |
|------|--------|--------|
| 2026-10-19 | [One owner per shared resource, and a brief checked against file ownership](2026-10-19-one-owner-per-shared-resource-in-a-multi-worker-wave.md) | [review-4 remediation](../changelogs/2026-10-18-rv4-remediation/index.md) |
| 2026-10-19 | [CanvasKit ignores CSS font faces, so self-hosting fonts is a loader setting](2026-10-19-canvaskit-ignores-css-font-faces.md) | [wave 3, client assets](../changelogs/2026-10-18-rv4-remediation/w3-client-assets.md) |
| 2026-10-19 | [A test on a global row takes the named lock and puts the row back](2026-10-19-a-test-on-a-global-row-takes-the-named-lock.md) | [wave 3, test hardening](../changelogs/2026-10-18-rv4-remediation/w3-test-hardening.md) |
| 2026-10-19 | [A new exported client method needs a row in the escape sweep](2026-10-19-a-new-client-method-needs-a-row-in-the-escape-sweep.md) | [wave 3, first-run copy](../changelogs/2026-10-18-rv4-remediation/w3-first-run-copy.md) |
| 2026-10-19 | [When the server takes over a write, remove the client's copy in the same wave](2026-10-19-when-the-server-takes-over-a-write-remove-the-clients-copy.md) | [wave 3, client apply](../changelogs/2026-10-18-rv4-remediation/w3-client-apply.md) |
| 2026-10-19 | [A handler that never reads its body never sees the client leave](2026-10-19-a-handler-that-never-reads-its-body-never-sees-the-client-leave.md) | [wave 3, context and deadlines](../changelogs/2026-10-18-rv4-remediation/w3-context-deadlines.md) |
| 2026-10-19 | [A release script fails open in three quiet ways](2026-10-19-a-release-script-fails-open-in-three-quiet-ways.md) | [wave 3, release and supply chain](../changelogs/2026-10-18-rv4-remediation/w3-release-supply.md) |
| 2026-10-18 | [A route policy keys on the mux pattern and treats the zero value as denied](2026-10-18-a-route-policy-keys-on-the-pattern-and-fails-closed.md) | [wave 2, agent scope](../changelogs/2026-10-18-rv4-remediation/w2-agent-scope.md) |
| 2026-10-18 | [A column grant turns "RLS hid the row" into "permission denied"](2026-10-18-a-column-grant-turns-zero-rows-into-permission-denied.md) | [wave 2, RLS](../changelogs/2026-10-18-rv4-remediation/w2-rls.md) |
| 2026-10-18 | [A wall-clock test turns red by date, and a push-only CI never sees it](2026-10-18-a-wall-clock-test-needs-a-scheduled-run.md) | [multirole review 4](../changelogs/2026-10-18-multirole-review-4.md) |
| 2026-10-12 | [Key test failures must never render secret operands](2026-10-12-key-tests-must-not-render-secrets.md) | [guided key workflow](../changelogs/2026-10-12-guided-key-workflow.md) |
| 2026-10-11 | [A truncated gate snapshot can mislabel its sections](2026-10-11-truncated-gate-snapshots-mislabel-sections.md) | [docs-order final audit D](../changelogs/2026-10-11-docs-order-marksman.md) |
| 2026-10-11 | [A duplicate source entry masquerades as a rewrite](2026-10-11-a-duplicate-source-entry-masquerades-as-a-rewrite.md) | [docs-order C2 review-fix](../changelogs/2026-10-11-docs-order-marksman.md) |
| 2026-10-11 | [Rerun the gate after a workflow's last repair round](2026-10-11-rerun-gates-after-a-workflows-last-repair.md) | [docs-order C1](../changelogs/2026-10-11-docs-order-marksman.md) |
| 2026-10-11 | [A verbatim quote is evidence, not a link](2026-10-11-a-verbatim-quote-is-evidence-not-a-link.md) | [docs-order link surgery](../plans/2026-10-11-docs-marksman-order.md) |
| 2026-10-11 | [Purge by an enumerated keep-set, never by a date rule](2026-10-11-purge-by-keep-set-not-by-date.md) | [the legacy purge](../changelogs/2026-10-11-owner-deploy-and-mail-truth.md) |
| 2026-10-11 | [A send-scoped API key's 401 looks exactly like a dead key](2026-10-11-a-send-scoped-keys-401-looks-like-a-dead-key.md) | [the re-key and mailed-flow pass](../changelogs/2026-10-11-owner-deploy-and-mail-truth.md) |
| 2026-10-11 | [A key "synced everywhere" can still be dead everywhere](2026-10-11-synced-everywhere-is-not-liveness.md) | [owner gap closure](../changelogs/2026-10-11-owner-deploy-and-mail-truth.md) |
| 2026-10-11 | [`CREATE OR REPLACE` with a changed signature creates an overload](2026-10-11-create-or-replace-with-a-new-signature-overloads.md) | [review-3 remediation](../changelogs/2026-10-11-rv3-remediation/index.md) |
| 2026-10-10 | [A skeptic confirms the finding, not the fix](2026-10-10-a-skeptic-confirms-the-finding-not-the-fix.md) | [multirole review 3](../changelogs/2026-10-10-multirole-review-3.md) |
| 2026-10-10 | [A frozen orchestration is recoverable if the state is on disk](2026-10-10-resumable-orchestration.md) | [review-register sweep](../changelogs/2026-10-10-close-open-gaps/review-sweep-server.md) |
| 2026-10-10 | [zh Material strings cost the web bundle ~110 KB gzip](2026-10-10-locale-tables-cost-110kb-gzip.md) | [review-register sweep, client](../changelogs/2026-10-10-close-open-gaps/review-sweep-client.md) |
| 2026-10-10 | [A VM test cannot show what the web's key map does](2026-10-10-vm-tests-do-not-run-the-web-key-map.md) | [BUG-028](../issues/2026-10-10-bug-028-enter-did-not-activate-focused-blocks-on-the-web.md) |
| 2026-10-10 | [A check that still passes when you break the thing guards nothing](2026-10-10-a-check-that-passes-under-its-mutation-guards-nothing.md) | [BUG-027](../issues/2026-10-10-bug-027-nginx-resolved-the-api-upstream-once.md) |
| 2026-10-10 | [A restyle leaves its strings behind; make a red prove what it claims](2026-10-10-a-restyle-leaves-its-strings-behind.md) | [BUG-026](../issues/2026-10-10-bug-026-committed-versions-hid-what-they-recorded.md) |
| 2026-10-10 | [A fake copies the page's misreading of the contract](2026-10-10-a-fake-copies-the-pages-misreading.md) | [BUG-022](../issues/2026-10-10-bug-022-build-mvp-wrote-nothing.md), [BUG-030](../issues/2026-10-10-bug-030-audience-view-rendered-a-blank-page.md) |
| 2026-10-10 | [Test the width you ship for, and assert rectangles](2026-10-10-test-the-width-you-ship-for.md) | [BUG-025](../issues/2026-10-10-bug-025-phone-layouts-and-copy-found-by-the-web-walk.md) |
| 2026-10-10 | [A path nobody can run hides a missing write](2026-10-10-a-path-nobody-can-run-hides-a-missing-write.md) | [setup roles and the bundled owner DSN](../changelogs/2026-10-10-close-open-gaps/setup-roles-and-bundled-owner.md) |
| 2026-10-10 | [Measure a deferred optimization before shipping it](2026-10-10-measure-before-rewriting-a-query.md) | [backend fixes](../changelogs/2026-10-10-close-open-gaps/backend-fixes.md) |
| 2026-10-10 | [Run a setup doc on its own platform in CI](2026-10-10-run-a-doc-on-its-platform-in-ci.md) | [CI gaps and the Ubuntu install skill](../changelogs/2026-10-10-close-open-gaps/env-ci-and-skill.md) |
| 2026-10-10 | [Script-style tests are silently skipped by `unittest`](2026-10-10-script-style-tests-are-skipped-by-unittest.md) | [CI gaps and the Ubuntu install skill](../changelogs/2026-10-10-close-open-gaps/env-ci-and-skill.md) |
| 2026-10-10 | [A failing step hides the gate behind it](2026-10-10-a-failing-step-hides-the-gate-behind-it.md) | [CI and host fixes](../changelogs/2026-10-10-close-open-gaps/ci-and-hosts.md) |
| 2026-10-09 | [Pinning `HOME` isolates a config path on macOS only](2026-10-09-pinning-home-isolates-config-on-macos-only.md) | [BUG-021](../issues/2026-10-09-bug-021-go-tests-share-one-config-file-on-linux.md) |
| 2026-10-09 | [A lockfile diff confined to SDK-pinned packages is toolchain drift](2026-10-09-lockfile-diff-in-sdk-pins-is-toolchain-drift.md) | [host sync](../changelogs/2026-10-09-host-sync.md) |
| 2026-10-08 | [Replacing a platform widget drops the affordances no test asserts](2026-10-08-custom-primitives-drop-unasserted-affordances.md) | [block focus ring](../changelogs/2026-10-08-block-focus-ring.md) |
| 2026-10-08 | [An agent's self-report is not evidence — of completion, or of status](2026-10-08-agent-self-report-is-not-evidence.md) | [frontend block craft](../plans/2026-10-07-frontend-block-craft/2026-10-07-frontend-block-craft.md) |
| 2026-10-07 | [A signed-off token layer hides an unported component layer](2026-10-07-token-signoff-hides-unported-component-layer.md) | [frontend block craft](../plans/2026-10-07-frontend-block-craft/2026-10-07-frontend-block-craft.md) |
| 2026-10-07 | [A documented curl|bash URL is a release asset contract](2026-10-07-curlbash-url-is-a-release-asset.md) | [release pipeline](../changelogs/2026-10-07-release-pipeline.md) |
| 2026-10-07 | [The own-URL path's hidden precondition: provisioned roles with owner BYPASSRLS](2026-10-07-own-url-role-provisioning-precondition.md) | [setup TUI slice 2](../changelogs/2026-10-07-setup-tui-slice-2.md) |
| 2026-10-07 | [Parallel slots on a dirty worktree can revert unrelated uncommitted lines](2026-10-07-parallel-slots-dirty-worktree.md) | [setup TUI slice 1](../changelogs/2026-10-07-setup-tui-slice-1.md) |
| 2026-10-07 | [A workflow gate must use absolute paths: world.run's cwd is not guaranteed to be the repo root](2026-10-07-workflow-gates-need-absolute-paths.md) | [setup TUI workflow](../changelogs/2026-10-07-setup-tui-parallel-workflows.md) |
| 2026-10-07 | [x/term ReadPassword already consumes the trailing newline](2026-10-07-xterm-readpassword-consumes-newline.md) | [setup TUI workflow](../changelogs/2026-10-07-setup-tui-parallel-workflows.md) |
| 2026-10-07 | [go mod tidy silently strips newly added but not-yet-imported deps](2026-10-07-go-mod-tidy-strips-unused-new-deps.md) | [setup TUI plan deps step](../plans/2026-10-07-setup-tui.md) |
| 2026-10-06 | [Relative links are depth-sensitive: resolve them mechanically](2026-10-06-relative-links-are-depth-sensitive.md) | [install-skill changelog](../changelogs/2026-10-06-install-skill.md) |
| 2026-10-06 | [A localhost API default makes a finished install unusable](2026-10-06-localhost-default-makes-install-unusable.md) | [install hosted default](../changelogs/2026-10-06-install-hosted-default.md) |
| 2026-10-06 | [Two ramp steps can fail both grounds](2026-10-06-two-ramp-steps-can-fail-both-grounds.md) | [app-icon candidates](../changelogs/2026-10-06-app-icon-candidates.md) |
| 2026-10-06 | [A transparent mark needs a zero-alpha corner](2026-10-06-a-transparent-mark-needs-a-zero-alpha-corner.md) | [app-icon candidates](../changelogs/2026-10-06-app-icon-candidates.md) |
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
