## PM review — frontend Cobalt, main @ f185e94

**Bottom line: 6.0/10.** A stranger can finally get from clone to a running
frontend, the design law and the token file agree on the spine, and CI truly
guards the client — but the two surfaces every user sees first are visibly
broken at a common laptop viewport, the law and the code have drifted in a
half-dozen quiet places, an unexplained dependency downgrade is sitting in
the worktree, and the neo-brutalist pivot the user wants next has no plan,
no record, and no prepared seam in the tokens.

### Can a stranger follow the README from clone to a running frontend?

Yes — and this is a real recovery from review 1, where the quick start
dead-ended three separate ways. Today's README (README.md:79-99) walks
`env.sh init` → `dev.sh migrate` → `dev.sh run` → `cd client && flutter run -d chrome`,
states plainly that `dev.sh` is not optional, and documents the
invitation-only default with its bootstrap (`promote-admin`, mint invites)
at README.md:132-137. The client defaults its API base to the local server
(`client/lib/session/session.dart:7-10`), so the README's bare `flutter run`
command actually works; the CORS trap review 1 filed (FS-03) is closed because
`dev.sh` now exports `INSIDEOUT_CORS_ORIGINS=localhost` for client runs
(docs/usage/local-development.md:124-127). Two gaps remain: no user-facing
document pins the Flutter version ("Flutter 3" is the entire guidance,
local-development.md:11, while CI and `client/Dockerfile` pin 3.44.0), and
the worktree's dirty `client/pubspec.lock` — which *downgrades* intl, matcher,
meta, test_api, and vector_math — is exactly what a version-unpinned
`flutter pub get` produces. That lockfile must be restored or deliberately
re-committed, and the version must be written down where strangers look.

### Does DESIGN.md's lock match cobalt.dart and the pages?

Mostly — and the agreement is worth saying precisely, because it is what the
pivot will lean on. All seven locked hexes appear verbatim in
`client/lib/theme/cobalt.dart:62-70`; the button lock ("14px 24px", cobalt
fill) matches `cobalt_theme.dart:62-69`; the type lock (Display 48–58, Title
26+, Body 17.5, Meta 12) matches `cobalt_theme.dart:129-145`; cards are 20px
radius with a cobalt-200 hairline; chips are shade pills, not chops
(`client/lib/theme/cobalt_chip.dart`). The drift is at the edges, and it is
all one-directional (code exceeds or contradicts the law): the dashboard
Home numeral is an ad-hoc 120px (`dashboard_page.dart:204`) against the
48–58 Display band DESIGN.md assigns to "page names and the Home numeral";
`cobalt.dart` defines `radiusControl = 12.0` against the `control: 9999px`
pill decree (used once, in `step_peek.dart:148`, where it is not a control);
and the dock — which DESIGN.md mandates as "bottom dock… no Linear-style left
rail" — currently renders at the *vertical center* of the viewport on every
signed-in page, because `AppScaffold` combines `extendBody: true` with a
`bottomNavigationBar` whose child `Center` expands into the full-height slot
(app.dart:71,135; product_dock.dart:28), while the bounding constant
`ProductDock.height = 88.0` sits unused two screens up from where it is
needed. The screenshots in `docs/review/2026-10-06-frontend-cobalt/shots/`
confirm it on dashboard (chapter-card body clipped), workspace (the dock
covers 「删除这个工作区」), and roadmap (empty state bisected). The auth modal
has the complementary problem: its cobalt dot, "InsideOut" wordmark, and
greeting are Column *siblings above* the card (`auth_door.dart:50-70`), so
over the `black54` barrier they collide with the landing hero and both CTAs
— shot 02 shows the wordmark sitting on top of 「开始打磨」. DESIGN.md's "cards
on a field" law is one surface; the modal draws two.

### What did prior reviews already file?

This review must not re-file, and does not: USER-01 (invitation-only
undocumented — now in README), FS-02/FS-03 (invite deep link, CORS — closed
in the remediation waves), USER-02/USER-10 (token refresh — see the
`api_token_refresh_test` the client now ships), FEAT-01/FEAT-02 (quiet flag
and board — review-2 wave 3 code), the raw-English-error cluster (the
`localizedError` mapper exists; residual call sites belong to the code
reviewers), the band-widget duplication (review-2 code-quality), the
client.dart/models.dart line-budget issues (filed 2026-09-18 / 2026-10-04),
and review-2's PM features-left ledger (Branch/Merge, Decision Log,
per-project authority — explicitly end-state and parked by design). What is
*new* here: the two chrome-level rendering Highs (PM-01, PM-02), the
lock/code drift pair (PM-05, PM-06), the lockfile churn (PM-03), the
unpinned toolchain (PM-04), the missing pivot records (PM-07), the hardcoded
paint at the pivot's first surfaces (PM-08), CI's blind spot for layout
(PM-09), and three small record-hygiene items (PM-10, PM-11, PM-12) —
HANDOFF's stale "worktree is clean", DESIGN.md's absence from every index,
and the three over-budget feature files with no modularity issue.

### CI and worktree hygiene

The client CI job (ci.yml:72-95) is genuinely good: pinned Flutter 3.44.0 in
lockstep with the shipped image, analyze with infos as the accepted baseline,
the full test suite, and a release web build. Its blind spot is total for
this bug class: both confirmed Highs compile, analyze, and unit-test green.
A golden or headless-screenshot smoke on the shell and AuthDoor is the
cheapest insurance the pivot will ever buy. The worktree itself is a mixed
picture: the docs/plans and HANDOFF modifications are the honest in-flight
record of f185e94 and should be committed as their own checkpoint, but
`client/pubspec.lock` does not belong in that commit, and HANDOFF's
"worktree is clean" sentence (still present at line 126) is exactly the
warning the file exists to carry. The untracked
`docs/review/2026-10-06-frontend-cobalt/` folder also needs its index README
per `docs/review/README.md` convention.

### How the current design complicates the pivot

The incoming language is the current one's photographic negative: radius
impossible by decree versus a 999px pill law; 2px borders on everything
versus hairline-only cards; hard offset shadows versus soft elevation; a
top-nav h-16 shell versus the bottom dock that is currently misrendering;
a four-step background ladder versus blur shadows. Two structural facts
matter for sequencing. First, the good news: the Cobalt token class is a
single 28-field seam, so a pivot can land as token + theme changes first and
page sweeps second — but only after the hardcoded paint at landing_page,
auth_door, and product_dock is pulled into tokens, or the first screen will
be half-pivoted. Second, the record obligation: PRODUCT.md:438-443 codifies
Cobalt as the brand commitment and the design-qa log's newest entry is
2026-09-18; the 2026-08-19 thread ("an infra migration does not mean visual
change") is the precedent for what happens when a visual change proceeds
without a recorded decision. The pivot needs its dated plan, its board row,
its design-qa thread with the cobalt-palette variant of the brief, and its
DESIGN.md amendment — before the first radius is zeroed. And it should
inherit a fixed dock and a single-surface auth modal first, or it will be
debugging layout on two fires at once

**Release gate:** ship no client build until the dock and AuthDoor fixes land, the web bundle is rebuilt from that tree (`railway up --service app`, superseding the stale :5174 bundle), and a signed-in page load is verified on the deployed domain.
