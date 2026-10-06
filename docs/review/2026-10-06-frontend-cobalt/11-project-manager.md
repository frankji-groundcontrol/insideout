# 11 · Project Manager

**Lens:** Docs and records health, release process, plan/changelog/issue hygiene
**Score:** **6 / 10**
**Bottom line:** The stranger path from clone to running frontend now holds, the design docs and tokens agree on most of the lock, and CI genuinely guards the client — but the two most-seen surfaces of the product (the signed-in navigation dock and the first-touch auth modal) render visibly broken at a common laptop viewport, the written design law and the code have drifted apart in a half-dozen small places, an unexplained pubspec.lock downgrade is sitting uncommitted, and the incoming neo-brutalist pivot has no plan row, no design-qa thread, and no token seam prepared for it.

Reviewed branch `main` at `f185e94`; a skeptic pass re-checked every Critical/High finding — corrections marked [Correction]. Screenshot evidence lives in `./shots/`.

---

## Scope

The PM pass covered the surfaces this lens owns: whether a stranger can follow the README from clone to a running frontend; whether DESIGN.md's written lock matches `cobalt.dart` and the shipped pages; what prior reviews already filed (so nothing is re-filed); CI and worktree hygiene; how the current design and its records complicate the incoming neo-brutalist pivot; and the release gate for the hosted client. Code-level correctness inside widgets was left to the code reviewers; rendering defects are filed here only because they ship through the release process this role guards.

## What is good

- The README quick start is finally coherent end-to-end: `env.sh init` → `dev.sh migrate` → `dev.sh run` → `cd client && flutter run -d chrome` (`README.md:79-99`), the client's default API base is the local server (`client/lib/session/session.dart:7-10`), and invitation-only signup is documented with its bootstrap path (`README.md:132-137`). This is a real recovery from review 1, where the quick start dead-ended three separate ways.
- DESIGN.md and the token file agree on the spine: all seven locked hexes appear verbatim in `client/lib/theme/cobalt.dart:62-70`, the button lock "14px 24px" matches `client/lib/theme/cobalt_theme.dart:67`, and the type lock (Display 48–58 / Title 26+ / Body 17.5 / Meta 12) matches `client/lib/theme/cobalt_theme.dart:129-145`. This agreement is what the pivot will lean on.
- CI covers the client properly: pinned Flutter 3.44.0 in lockstep with the shipped image (`.github/workflows/ci.yml:83-86`, `client/Dockerfile:1-3`), plus analyze, the full test suite, and a release web build (`ci.yml:88-95`).
- The records culture this repo demands is mostly real: two prior whole-tree reviews with skeptic passes and reproduction tables, a design-qa log that preserves verbatim user quotes and their resolutions, and a plan board that already reflects the f185e94 simulation fixes (`docs/plans/README.md:12-13`).
- The design-qa acceptance trail for the current language is intact (`docs/design-qa/2026-09-17-cobalt-layouts.md` → lock screens in `docs/design-system/cobalt/`), so the pivot can be recorded the same way.

## Findings, worst first

### pm-dock-mid-viewport — High

The ProductDock renders at the vertical center of the viewport on every signed-in page, occluding content — including the destructive delete-workspace row — instead of sitting at the bottom edge the design law requires. Anyone who signs in at 1440×900 gets primary navigation floating mid-screen on dashboard, workspace, and roadmap; on the workspace page it partially hides the delete-workspace action behind a cobalt pill, and it clips card text (the dashboard chapter body is unreadable behind the dock in shot 04).

**Evidence:** `client/lib/app.dart:135`

```
extendBody: true, (client/lib/app.dart:71) with bottomNavigationBar: signedIn ? const ProductDock() : null, (client/lib/app.dart:135); the dock's build returns `child: Center(` inside a SafeArea (client/lib/chrome/product_dock.dart:28) and `static const height = 88.0;` (product_dock.dart:15) is never referenced anywhere. Confirmed in ./shots/04-dashboard-light.png (dock at y≈440/900), ./shots/06-workspace-dark.png (covering 「删除这个工作区」), ./shots/10-roadmap-dark.png (bisecting the empty state).
```

**Fix:** Bound the bar: wrap the Center in `SizedBox(height: ProductDock.height)` (or drop the Center and set `extendBody: false` with body bottom padding), then re-shoot the three pages at 1440×900 to confirm the dock sits at the bottom edge.

**Verdict:** CONFIRMED — the skeptic pass independently reproduced the geometry: `product_dock.dart:28` Center + `:26` SafeArea inside the bottomNavigationBar slot (`app.dart:135`) with extendBody (`app.dart:71`) centers the pill at viewport mid-height, and `:15` height=88.0 is a dead constant with zero references. Corroborated across roles as the same root cause (dock-pill-mid-screen-root-cause, a semantic duplicate kept for traceability).

### pm-authdoor-header-collision — High

The auth modal's brand header (cobalt dot + "InsideOut" + greeting) renders as siblings ABOVE the modal card over a 54% dim barrier, so the wordmark collides with the landing page's hero subtitle and its two CTA buttons. Every first-time visitor who clicks 登录 on the landing page gets a modal whose header fights the page behind it — two competing "InsideOut" wordmarks and CTA labels visible through the barrier; at zh-CN the hero subtitle shows through the wordmark, making both unreadable.

**Evidence:** `client/lib/features/auth/auth_door.dart:50`

```
child: const ColoredBox(color: Colors.black54), (auth_door.dart:38); decoration: const BoxDecoration(color: Cobalt.hue500, shape: BoxShape.circle), (:54); child: Text('InsideOut', style: theme.textTheme.headlineSmall), (:62) — Column siblings above the card, which only starts at auth_door.dart:72. ./shots/02-login-modal.png shows the dot at y≈215 and the wordmark at y≈280 overlapping 「开始打磨」/「登录」.
```

**Fix:** Move the dot, wordmark, and greeting inside the modal card (single surface, matching DESIGN.md's "rounded cards on a field"), or make the barrier fully opaque. Then re-shoot 02/03.

**Verdict:** CONFIRMED — `auth_door.dart:38` `ColoredBox(color: Colors.black54)` scrims the full page, the brand header is a Column sibling above the `:72` card, and `landing_page.dart:50,181` stack the layers the header collides with.

### pm-pubspec-lock-downgrade — Medium

The worktree carries an uncommitted `pubspec.lock` that DOWNGRADES six transitive packages (intl 0.20.3→0.20.2, matcher 0.12.20→0.12.19, meta 1.19.0→1.18.0, test_api, vector_math 2.4.2→2.2.0), consistent with a `pub get` from an older local SDK than CI's pinned 3.44.0. If this lockfile rides along with the next docs commit, the repo certifies older transitives than the toolchain CI builds with; the next dev on 3.44.x flips it back, producing lockfile ping-pong and unreviewed dependency drift on every unrelated change.

**Evidence:** `client/pubspec.lock:215`

```
git diff client/pubspec.lock shows `-      version: "0.20.3"` → `+      version: "0.20.2"` (hunk @@ -209,10), `-      version: "1.19.0"` → `+      version: "1.18.0"` (@@ -305,10); working tree now has client/pubspec.lock:215 `    version: "0.20.2"`, client/pubspec.lock:311 `    version: "1.18.0"`, client/pubspec.lock:508 `    version: "2.2.0"` — all DOWNGRADES.
```

**Fix:** Restore it (`git checkout -- client/pubspec.lock`) unless the downgrade is deliberate and explained, and root-cause the local SDK mismatch (see pm-flutter-version-unpinned).

### pm-flutter-version-unpinned — Medium

No user-facing doc pins the Flutter version: local-development says only "Flutter 3", `client/README.md` omits a version entirely, while CI and the shipped image pin 3.44.0 — and the pubspec.lock downgrade above is the demonstrated bite of that gap. A stranger (or the maintainer's other machine) with Flutter 3.3x/3.5x runs `flutter pub get`, silently churns the lockfile exactly as the current dirty diff shows, and either commits the downgrade or debugs resolution errors.

**Evidence:** `docs/usage/local-development.md:11`

```
- **Flutter 3** (for `client/`; see [`client/README.md`](../../client/README.md)); — client/README.md has no version or Prerequisites section at all; client/Dockerfile:1-3 `# Pin to 3.44.0 (Dart 3.12.0). ... FROM ghcr.io/cirruslabs/flutter:3.44.0 AS build`; ci.yml:85 `flutter-version: 3.44.0`.
```

**Fix:** Add a Prerequisites line to `client/README.md` ("Flutter 3.44.x — the version CI and client/Dockerfile pin") and tighten `docs/usage/local-development.md:11` to match.

### pm-home-numeral-off-lock — Medium

The dashboard chapter numeral is an ad-hoc 120px, more than double the Display band (48–58) that DESIGN.md assigns to "page names and the Home numeral". The written design law and the shipped UI disagree on the product's home screen; any contributor enforcing DESIGN.md "fixes" the numeral, any designer trusting the screenshot amends the doc — drift compounds.

**Evidence:** `client/lib/features/dashboard/dashboard_page.dart:204`

```
style: theme.textTheme.displayLarge?.copyWith(color: palette.brand, fontSize: 120, height: 0.9), — vs DESIGN.md:69 `Body **17.5**. Titles **26+**. Display **48–58**. Meta **12**.` and DESIGN.md:96 `- **Display** (700, 48–58, Ideas wall up to 320): page names and the Home numeral.` (visible as the oversized "01" in ./shots/04-dashboard-light.png).
```

**Fix:** Pick one authority in a single commit: either lock 120 as the Home numeral exception in DESIGN.md, or bring `dashboard_page.dart:204` into the 48–58 band.

### pm-rogue-radius-token — Medium

The token file defines `radiusControl = 12.0`, a non-pill control radius that contradicts DESIGN.md's lock (`rounded.control: "9999px"`) and is used exactly once — a rogue token that also complicates the zero-radius pivot. A contributor reading "control" applies a 12px radius to an actual control and breaks the pill lock; during the pivot, three radius tokens must be reconciled instead of one semantic decision.

**Evidence:** `client/lib/theme/cobalt.dart:72`

```
static const radiusControl = 12.0; — vs DESIGN.md:28 `  control: "9999px"`; the sole use is client/lib/features/landing/step_peek.dart:148 `borderRadius: BorderRadius.circular(Cobalt.radiusControl),` while real controls use radiusPill (cobalt_theme.dart:29,93,97,101).
```

**Fix:** Delete `radiusControl` or rename it for what it is (peek-tile corner) and realign the DESIGN.md frontmatter; when the pivot lands, collapse all radius tokens to one decree.

### pm-pivot-has-no-records — Medium

The neo-brutalist pivot (the user's stated next design direction) has no plan row, no design-qa thread, and no amendment to the documents that currently codify Cobalt as the brand commitment — the record system the repo requires to precede code doesn't exist yet for this work. The pivot starts in code (it is already being discussed as a design brief), and tokens/pages get restyled with no accepted lock screens, no decision record, and no board tracking — exactly the failure mode the 2026-08-19 "Flutter cutover dropped Ink & Seal" design-qa thread exists to prevent.

**Evidence:** `PRODUCT.md:438`

```
**Visual identity:** the shipped Cobalt design system — lock screens and design tokens in [`docs/design-system/cobalt/`] ... was superseded by the Cobalt restyle merged 2026-09-18 (PRODUCT.md:438-443); docs/design-qa/README.md newest record is 2026-09-18; docs/plans/README.md:10-13 current P1 rows cover simulation bugs and key stores — no pivot row.
```

**Direction:** Record before code: open a dated plan (status board row) plus a design-qa thread with the reference brief and the cobalt-palette variant, and amend DESIGN.md/PRODUCT.md in the same commit that lands the first token change.

**First check:** Confirm the pivot decision verbatim with the user, then check `docs/design-qa/README.md` and `docs/plans/README.md` for the newest rows to mirror their format.

### pm-hardcoded-paint-at-pivot-seam — Medium

The surfaces the pivot touches first carry hardcoded colors and shadow values outside the Cobalt token layer (literal cobalt hex on the landing brand mark, raw black barrier/shadows in auth_door and product_dock). When the zero-radius/border-everything/hard-shadow language lands, these four sites — landing mark, first-touch modal, primary nav — must be found and hand-edited one by one instead of riding one token change, guaranteeing a half-pivoted first screen.

**Evidence:** `client/lib/features/landing/landing_page.dart:59`

```
decoration: const BoxDecoration(color: Color(0xFF3B5BFF), shape: BoxShape.circle), (landing_page.dart:59); child: const ColoredBox(color: Colors.black54) (auth_door.dart:38) and shadowColor: Colors.black.withValues(alpha: 0.35), (auth_door.dart:80); shadowColor: Cobalt.lightInk.withValues(alpha: 0.18), (product_dock.dart:32).
```

**Fix:** Introduce token roles (brandMark, barrier, hardShadow) and migrate these call sites as the pivot's first commit, before any visual change.

### pm-ci-blind-to-rendering — Medium

The client CI proves analyze + unit tests + a compiling web build — nothing observes rendering, so both confirmed Highs (mid-screen dock, colliding modal header) pass CI green and ship. Any layout regression in the Scaffold chrome — the exact class of bug these two findings represent — reaches the hosted bundle undetected until a human looks at a page.

**Evidence:** `.github/workflows/ci.yml:88`

```
- run: flutter analyze --no-fatal-infos / - run: flutter test / - run: flutter build web --release --dart-define=API_BASE=/api/v1 (ci.yml:91-95) — no golden, screenshot, or smoke step exists anywhere in the workflow.
```

**Fix:** Add a minimal visual guard: either golden-file tests for AppScaffold/AuthDoor at 1440×900 or a CI step that serves the release build and screenshots the landing + a signed-in shell via headless Chrome.

### pm-handoff-claims-clean-worktree — Low

HANDOFF's "Worktree to preserve" section still declares "the worktree is clean" while git status shows six dirty entries — the current-worktree warning AGENTS.md tasks HANDOFF with carrying is missing. The next resuming agent is told the tree is clean and either distrusts HANDOFF or stages something unintended — HANDOFF's single job is a correct resume path.

**Evidence:** `docs/HANDOFF.md:126`

```
Both machines and origin were synced at the last [checkpoint] commit; the worktree is clean. (docs/HANDOFF.md:126-127) — vs `git status --short`: ` M client/pubspec.lock`, ` M docs/HANDOFF.md`, three modified docs/plans files, `?? docs/review/2026-10-06-frontend-cobalt/`.
```

**Fix:** Replace the sentence with one naming the in-flight rows (simulation-fix record updates) and the two anomalies (pubspec.lock churn, untracked review folder).

### pm-design-md-unlinked — Low

DESIGN.md — the declared current design law — is not linked from the doc map, the README's learn-more table, or the design-system index; only historical plans and changelogs reference it. A new contributor asked to "follow the design law" finds the lock screens and tokens but not the prose law (radii decree, dock rule, do's/don'ts), and the pivot lands without a discoverable document to amend.

**Evidence:** `docs/index.md:44`

```
docs/index.md:44-54 links design-qa/ and design-system/ (incl. cobalt/) but never DESIGN.md; README.md:141-151's "Learn more" table has no design row; `grep -rl DESIGN.md docs/` returns only plans/, changelogs/, and one design-qa history file.
```

**Fix:** Add a DESIGN.md row to README's learn-more table and a pointer line in `docs/index.md` next to design-system/.

### pm-modularity-issues-missing — Low

Three feature files over the ~350-line budget (project_page.dart 548, workspace_page.dart 371, ideas_page.dart 370) lack the dated modularity issue AGENTS.md requires; only client.dart and models.dart have one. The oversize files keep growing untracked (project_page.dart gained ~70 lines past the budget with no target structure on record), and the line-budget rule silently erodes.

**Evidence:** `docs/issues/README.md:103`

```
docs/issues/README.md:103-108 lists only the models.dart and client.dart line-budget issues for the client; AGENTS.md: "A chunky or mixed-responsibility file that can't be split now gets a dated modularity issue in docs/issues/ with the target structure and a bounded fix prompt."
```

**Fix:** File one dated issue per file (or one grouped issue) with the target split and bounded fix prompt, mirroring `docs/issues/2026-10-04-models-dart-line-budget.md`.

## Recommendations

1. Fix the dock geometry first: bound ProductDock with its own unused `ProductDock.height` (or remove the Center/extendBody combination), then re-shoot dashboard, workspace, and roadmap at 1440×900 to confirm the dock sits at the bottom edge and no longer covers the delete-workspace row.
2. Fold the AuthDoor mark, wordmark, and greeting into the modal card (or make the barrier opaque) and re-shoot the login/register captures — the first touch of every visitor.
3. Restore `client/pubspec.lock` to the committed resolution unless the downgrade is deliberate, and pin "Flutter 3.44.x" in `client/README.md` and `docs/usage/local-development.md` so local pub resolution matches CI and the Dockerfile.
4. Open the pivot's dated plan and design-qa thread before touching any token, and amend DESIGN.md + PRODUCT.md's brand commitment in the same commit as the first token change.
5. Reconcile the type and radius locks with the code in one commit each way — either amend DESIGN.md to lock the 120px Home numeral or bring `dashboard_page.dart` into the 48–58 band; delete or rename `radiusControl`.
6. Migrate the four hardcoded color/shadow sites (landing mark, auth barrier and shadow, dock shadow) into Cobalt token roles as the pivot's first commit.
7. Add a minimal visual guard to the client CI job — golden tests for the shell and AuthDoor, or a serve-and-screenshot smoke — so chrome-level layout regressions stop shipping green.
8. Close out the records: correct HANDOFF's "worktree is clean" line, file the three missing modularity issues, link DESIGN.md from the doc map and README, and finish `docs/review/2026-10-06-frontend-cobalt/` with its index README per `docs/review/README.md` convention.

**Release gate:** ship no client build until the dock and AuthDoor fixes land, the web bundle is rebuilt from that tree (superseding the stale hosted bundle), and a signed-in page load is verified on the deployed domain.

## What this role would do first

Fix the dock geometry in `client/lib/chrome/product_dock.dart` and `client/lib/app.dart` — bind the bar with the already-defined `ProductDock.height` — and fold the AuthDoor dot/wordmark/greeting into the modal card in `client/lib/features/auth/auth_door.dart`; then rebuild the web bundle from that tree and run the signed-in hosted page-load gate. No client release before that.
