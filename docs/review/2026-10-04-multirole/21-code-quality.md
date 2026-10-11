# 21 · Code Quality

**Lens:** file-size rule and its tracking, dead code, duplicate stacks, logging
discipline, error-swallowing, copy-paste helpers, stale comments, test quality
(wiring vs behavior), lint coverage.
**Score:** **7.2 / 10**
**Bottom line:** The code itself is disciplined — structured logging
everywhere it matters, a centralized HTTP-call table with shape-pinning tests,
a correct conditional-import SSE pair, and a green `flutter analyze` + 57/57
client tests. The debt is structural bookkeeping: six files breach the ~350
line rule but only two carry the dated issue the contract requires, one
Flutter test greps source text instead of testing behavior, and one dead Go
helper plus a stale-assignment test bug show that no staticcheck config
exists to catch this class. None of it breaks the documented path today; all
of it compounds.

Reviewed against main@877ad03; High/Critical findings re-checked by an
independent skeptic — verdicts in [01-findings-register.md](01-findings-register.md).

---
## Scope

All of `server/internal/` (33 packages), `server/cmd/insideout/`,
`client/lib/` (66 Dart files), `client/test/` (21 files), `scripts/` (10
files), `client/analysis_options.yaml`, and lint configuration anywhere in
the tree. Ran `wc -l` over every Go/Dart/py/sh file; ran
`staticcheck ./...` (v0.8.1) in `server/`; ran `flutter analyze` and
`flutter test` in `client/`; traced imports for every suspected dead or
duplicate stack (Ink & Seal remnants, Nuxt leftovers, `sse_transport`
variants, l10n tables).

## What is good

- **Structured logging is the house style in Go.** `slog.NewJSONHandler` is
  installed once at `server/cmd/insideout/main.go:31` and injected
  (`*slog.Logger`) into server, coach, mail, and seed; every error path logs
  with key/value pairs (`server/internal/agent/coach.go:94` etc.).
  `fmt.Println` appears only in CLI output paths (`cli_auth.go`,
  `cli_admin.go`), which is exactly where it belongs. No `print()` or
  `debugPrint` exists anywhere in `client/lib`.
- **The SSE transport pair is the right pattern, not copy-paste.**
  `client/lib/api/sse_transport_web.dart` (fetch + ReadableStream) and
  `sse_transport_stub.dart` (9-line `UnsupportedError`) are a conditional-import
  pair with a comment at `sse_transport_web.dart:8` documenting why dio's XHR
  adapter cannot stream (BUG-014). The two Anthropic/Responses stream parsers
  in `server/internal/agent/` parse genuinely different wire formats.
- **One authority for the client API contract.** `client/lib/api/requests.dart`
  builds every `ApiCall` (method/path/body), and `client/test/requests_test.dart`
  pins those shapes — including the BUG-015 lesson (wrong-method bug caught by
  httptest pinning in Go, mirrored here). The l10n catalogs have a key-parity
  test (`client/test/l10n_test.dart:14`), and `leftover_scan_test.dart`
  mechanically guards the Cobalt restyle against Ink & Seal token regressions.
- **Error handling is deliberate where it matters.** The coach's best-effort
  persistence uses `detachedContext` (`server/internal/agent/coach.go:60`) so
  cleanup writes survive request cancellation; the Dart session hydrate has a
  real fallback ladder (`client/lib/session/session.dart:49-63`). Bare
  `catch (_)` sites are few (6) and each is a defensible best-effort path.
- **Verification is green.** `go build ./...`, `go vet ./...` pass;
  `flutter analyze` reports 0 errors/warnings (20 info lints);
  `flutter test` passes 57/57.

## Findings

**CQ-01 · Medium · modularity-rule tracking.** `docs/issues/README.md` (index)
— four files over the ~350-line budget have no dated modularity issue.
Quote (AGENTS.md:71-74): "Keep implementation modular: files under ~350
lines, single responsibility … A chunky or mixed-responsibility file that
can't be split now gets a dated modularity issue in docs/issues/". Measured:
`server/internal/agent/coach.go` 410 (HTTP handler + SSE + persistence +
rate-limit/circuit in one struct), `server/internal/agent/anthropic.go` 406,
`client/lib/api/models.dart` 472, `server/internal/store/roadmap_test.go`
494. Only `apiclient.go` (478) and `client.dart` (529) are tracked
(2026-09-03, 2026-09-18 issues with bounded fix prompts). Failure scenario: a
developer needs the coach split prompt, finds none, and either splits blind
or adds a seventh responsibility; the rule degrades to "tracked when
convenient". Fix: write four dated issues in `docs/issues/` mirroring the two
existing line-budget records (target structure + bounded fix prompt), or land
the splits directly. diff_size: 4 small files (issues only).

**CQ-02 · Medium · tests that test wiring.** `client/test/feature_wiring_test.dart:16`
— `expect(settings.contains('updateWorkspace'), isTrue);` — the whole file is
35 `String.contains` assertions over source code read from disk
(`read('lib/features/workspace/settings_page.dart')`). It proves a substring
exists, not that any screen calls anything: a rename with zero behavior
change goes red; deleting the call and leaving the word in a comment goes
green. It also hard-codes incidental spelling (`'parentId: node.id'`,
`'coach.noConversation'`), so refactors pay cost with no signal. Fix: replace
with behavioral coverage (widget tests driving the real pages, as
`versions_page_test.dart` already does) or delete it and rely on
`requests_test.dart` + widget tests. diff_size: unknown (rewrite) or −53
lines (delete).

**CQ-03 · Low · dead code + no staticcheck gate.** `server/internal/api/cors.go:43`
— `func splitCORSOrigins(raw string) []string {` — verified unused
(staticcheck U1000; config parses CORS itself at
`server/internal/config/config.go:89` via `splitCSV`). Nothing catches this:
there is no `.golangci.yml`, no `staticcheck.conf`, no Makefile, and no CI
(L1). Failure scenario: a reader assumes this is the CORS parsing path and
"fixes" the wrong function; the next dead helper ships the same way. Fix:
delete the 13-line function and add a minimal staticcheck (or golangci-lint)
config plus a documented run command so U1000/SA4006 cannot recur.
diff_size: −13 lines + config file.

**CQ-04 · Low · dead assignment in a DB-gated test.** `server/internal/store/roadmap_test.go:400`
— `idx := byID(nodes)` — staticcheck SA4006: this value is never used; line
424 re-assigns `idx = byID(nodes)` before the first read. Harmless at runtime
but it means the replace-attribution test computes an index it discards, and
it demonstrates test files get no lint attention either (494 lines, itself
over budget — see CQ-01). Fix: delete the line-400 assignment. diff_size:
1 line.

**CQ-05 · Low · stale Nuxt comments in live code.** `server/internal/api/cookies.go:63`
— "refreshTokenFromRequest prefers the httpOnly refresh cookie (Nuxt) and"
— plus `server/internal/api/middleware.go:125` ("without going through the
Nuxt proxy"), `server/internal/config/config.go:31` ("Nuxt same-origin needs
none"), `server/internal/api/auth.go:246` ("so Nuxt can keep assigning the
JSON to UserProfile"), `client/lib/api/errors.dart:32` ("Codes Nuxt treats as
a Coach send backoff"), `client/lib/theme/ink_motion.dart:3` ("from the
Nuxt Ink & Seal landing / AuthDoor"), `.env.example:89` ("the browser talks
same-origin through the Nuxt proxy"). The Nuxt tree was deleted 2026-08-18
(docs/architecture/frontend.md:8). Failure scenario: a new maintainer
believes a Nuxt consumer still exists and preserves wire shapes, cookie
semantics, or error codes for a client that will never call them. Fix:
mechanical comment sweep rewriting each to name the current consumers
(Flutter web same-origin, CLI/MCP bearer, curl/tests with DEV_CORS=1).
diff_size: ~12 comment lines.

**CQ-06 · Low · BuildContext across async gaps.** `client/lib/features/prd/prd_page.dart:60`
— `context.read<LoopFocus>().rememberWorkspace(loaded.workspaceId);` after
three awaits with no `mounted` guard (analyzer `use_build_context_synchronously`,
8 sites total: prd_page ×3, revisions_page, versions_page, project_page,
settings_page, workspace_page). Failure scenario: user navigates away while
`getPrd`/`conversationForPrd`/`listMessages` are in flight → the page is
unmounted → `context.read` on a deactivated element throws inside the catch
that then calls `setState` on a dead widget. Low probability, real crash.
Fix: capture providers before the awaits or add `if (!mounted) return;`
after each gap. diff_size: ~8 one-line guards.

## Lead verification

| Lead | Verdict | Evidence |
|---|---|---|
| L4 (Ink & Seal leftovers) | **Refuted (code half)** — no dead Ink & Seal stack in live code | `grep -r "Ink & Seal\|InkSeal\|inkSealTheme"` over `client/lib`/`server` hits only docs and `ink_motion.dart`'s provenance comment; `ink_motion.dart` has 4 live imports (landing ×3, auth_door) and is guarded by `leftover_scan_test.dart`; other "ink" matches are `magic_link_page.dart` and the `lightInk` color token in `cobalt.dart`. The PRODUCT.md brand mismatch is a docs finding, not dead code. |
| L6 (stale Nuxt references) | **Confirmed + extended** | Lead listed `.env.example`, architecture docs, INIT.md, README, `env_catalog.py`, two client tests. Additionally found 4 live Go comment sites (`config.go:31`, `auth.go:246`, `cookies.go:63`, `middleware.go:125`) and 3 live Dart sites (`errors.dart:32`, `ink_motion.dart:3`, `assembly_diagram.dart:6`) the lead did not list. |
| L7 (size-rule breaches) | **Confirmed + extended** | `wc -l`: six files >350 — `client.dart` 529, `roadmap_test.go` 494, `apiclient.go` 478, `models.dart` 472, `coach.go` 410, `anthropic.go` 406. Issues index (`docs/issues/README.md:69-74`) tracks only `client.dart` and `apiclient.go`; the other four have no dated issue (CQ-01). |
| L13 (flutter analyze/test) | **Ran — confirmed green** | `flutter analyze`: 0 errors, 0 warnings, 20 info lints (8 `use_build_context_synchronously`, 7 `use_null_aware_elements`, 4 `prefer_initializing_formals`, 1 `use_function_type_syntax_for_parameters`). `flutter test`: 57/57 pass (plans board said 54 on 2026-09-18 — tests were added since). |

## Recommendations

1. Write the four missing dated modularity issues for `coach.go`,
   `anthropic.go`, `models.dart`, and `roadmap_test.go` — or land the
   `coach.go` split first, since it mixes HTTP, SSE, and persistence in one
   410-line struct.
2. Replace `feature_wiring_test.dart` with behavioral widget tests or delete
   it; substring greps over source files protect nothing.
3. Delete `splitCORSOrigins` and add a `staticcheck`/`golangci-lint` config
   with a documented run command (and hook it into CI when CI exists).
4. Run the mechanical Nuxt-comment sweep across the 11 live sites (Go, Dart,
   `.env.example`); one commit, no behavior change.
5. Guard the 8 `use_build_context_synchronously` sites with `mounted` checks
   or pre-await provider capture.
6. Land the two already-tracked splits (`apiclient.go`, `client.dart`) at
   their next structural touch, exactly per their bounded fix prompts.

## What this role would do first

Two mechanical cleanups, one commit each, both independently mergeable.
First: delete the dead `splitCORSOrigins` helper and the dead `idx`
assignment, and add a minimal staticcheck config — it converts a whole class
of rot (U1000/SA4006) from "reviewer with spare time" into a one-command
check, and it is the prerequisite for recommendation 3 sticking. Second: the
Nuxt-comment sweep — eleven comment sites across Go, Dart, and
`.env.example` describing a client deleted seven weeks ago; a single
sed-plus-reread pass with `go build`/`flutter analyze` as the only
verification. Both are pure-debt removals that touch no behavior, need no
product decision, and clear the ground so the structural items (CQ-01, CQ-02)
can be argued on their merits instead of drowning in trivia.
