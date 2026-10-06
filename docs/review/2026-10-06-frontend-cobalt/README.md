# Multirole review — frontend & Cobalt design language — 2026-10-06

Branch `main` @ `f185e94`. Scope: the Flutter web client (`client/lib`),
its theme system, and the gap between the shipped UI and the user-supplied
reference app's design language (adopted here as **Cobalt-Neo**: 0px radius,
2px solid borders, 4px/4px blur-free cast shadows, 100ms press-the-block
motion, dark-first, neutral dark ramp, existing blue hue family kept).
10 role agents + skeptic clusters + 1 completeness critic; every High/Critical
carries a skeptic verdict, and the top findings were reproduced by execution,
not just reading.

> **Bottom line:** the product works end-to-end and the architecture is sound
> (scores 6.5–7.2 on engineering roles), but the visual layer is unfinished:
> the primary navigation dock renders at viewport center on real screens, the
> top-bar icons are invisible over navy (1.42:1), every stroke token fails
> non-text contrast, and none of the reference design language's four
> signature mechanics (0 radius, cast shadow, border weight, press motion)
> exists in the theme. Mean score **5.22/10**. 114 findings
> (7 Critical, 22 High); 28 High/Critical confirmed, 1 downgraded, 0 refuted.
> Ordered fix path: [40-remediation-plan.md](40-remediation-plan.md) —
> ~11–12 engineer-days, highest-impact repairs first.

## Findings at a glance

114 findings: **7 Critical / 22 High / 50 Medium / 32 Low** (+3 filed by the
critic; register header counts 7C/21H/52M/34L after the one downgrade).
Verdicts on the 29 High/Critical: **28 CONFIRMED, 1 DOWNGRADED
(`dock-dead-clicks-evidence-version` H→M), 0 REFUTED, 0 UNVERIFIED.**

| File | Aspect | Score | Highlights |
|---|---|---|---|
| [10-fullstack.md](10-fullstack.md) | Full-stack wiring | 6.5 | Solid API/session architecture; write actions fail silently in 6 places |
| [11-project-manager.md](11-project-manager.md) | Ship-readiness | 6.0 | Two most-seen defects (mid-viewport dock, auth header collision) block release |
| [12-first-run-user.md](12-first-run-user.md) | First-run experience | 4.0 | Dock floats mid-screen; create-project FAB hidden behind it |
| [20-feature-implementation.md](20-feature-implementation.md) | Feature completeness | 7.2 | Core loop exists; raw enum as idea status, silent capture/convert failures |
| [21-code-quality.md](21-code-quality.md) | Code quality | 6.5 | Clean modules; 55/68 files not dart-format-clean, no CI gate |
| [22-testing.md](22-testing.md) | Test coverage | 6.5 | Harness hides the dock overlap; no golden at real viewport sizes |
| [23-reliability-ops.md](23-reliability-ops.md) | Reliability & ops | 5.5 | No network timeouts anywhere; hydrate wipes tokens on transport error |
| [24-design-fidelity.md](24-design-fidelity.md) | Design-language fidelity | 2.5 | None of the four Cobalt-Neo mechanics implemented; rounded decree inverted |
| [25-a11y-visual.md](25-a11y-visual.md) | Accessibility & visual contrast | 4.5 | Every stroke token 1.40–2.26:1; blue-on-blue dock states |
| [26-responsive.md](26-responsive.md) | Responsive behavior | 3.0 | Root cause of the mid-screen dock; no viewport meta tag |
| [30-completeness-critic.md](30-completeness-critic.md) | Missed dimensions | — | Security never assigned (clean probe, 1 Low filed); 41MB dead fonts in web bundle; 3 score/finding mismatches |

Full detail: [01-findings-register.md](01-findings-register.md).

## The Criticals

1. **Dock pill renders at viewport center, root cause** —
   `client/lib/widgets/product_dock.dart:26-28`: `Center` inside
   `bottomNavigationBar` expands to full body height behind `extendBody`;
   `static const height = 88.0` (:15) has zero references.
2. **Dock renders at viewport center, all pages** — same mechanism; wiring at
   `client/lib/app.dart:135`. Every screenshot shows the pill mid-screen.
3. **Dock unfinished state is blue-on-blue** —
   `client/lib/widgets/product_dock.dart:166`: `statusWarnFg` #1D4ED8 on dock
   brand #3B5BFF = **1.32:1** light, 3.41:1 dark.
4. **No press-the-block motion** — `client/lib/widgets/ink_motion.dart:4`:
   the 100ms press mechanic does not exist; no motion tokens at all.
5. **No cast-shadow system** — `client/lib/theme/cobalt_theme.dart:54-61`:
   zero blur-free 4px/4px cast shadows; no shadow tokens.
6. **Radius decree inverted** — `client/lib/theme/cobalt.dart:72-75` ships
   radius 12/20/28/999 where Cobalt-Neo decrees 0px everywhere.
7. **Top-bar icons invisible over navy** — `client/lib/app.dart:90`:
   translate/theme icons at **1.42:1** over the navy scaffold.

## Bugs reproduced

Reading is not evidence for runtime claims; the top three were executed:

| Bug | Command / probe | Result |
|---|---|---|
| Mid-screen dock root cause | Read `product_dock.dart:28` + `app.dart:71/135`, cross-check shots 04/06/07 | CONFIRMED — `Center` fills full `bottomNavigationBar` height → pill ≈ (720,443) mid-viewport; `height = 88.0` (:15) unreferenced. Companion lead "six hardcoded `bottom: 140` paddings" **REFUTED** by grep |
| Shared `_retryTimer` drops loops | Read `roadmap_page.dart:40-102` | CONFIRMED — `_progressLoop` (:69) and `_presenceLoop` (:92) share `_backoff()` which overwrites `_retryTimer` (:45); `dispose` (:99) cancels only the last → dangling Completer |
| Formatting drift | `dart format --set-exit-if-changed lib` | CONFIRMED — 55/68 files changed; no CI gate exists |

## Methodology

Stages per the multirole-review skill: (1) branch enumeration — `main` is the
product; (2) scout + lead list, cheap leads run by the lead (one refuted lead
— the "six `bottom: 140`" claim — deleted, not forwarded); (3) fan-out to 10
role agents on one schema (12-finding cap, `file:line` + verbatim quote or
drop, 0–10 scale, fix on every finding); (4) dedup, then skeptic clusters of
three told to refute by default — 29 High/Critical verdicts; (5) reproduction
of the top findings by the lead (table above); (6) completeness critic — one
uncovered advertised dimension (security; probed clean, 1 Low filed), three
score-vs-findings mismatches, three new findings; (7) redaction — the
reference app is named only as "the user-supplied reference app"; (8) this
report.

## Reading order

1. [40-remediation-plan.md](40-remediation-plan.md) — the ordered fix path;
   start at Wave 1.
2. [00-executive-summary.md](00-executive-summary.md) — scores, top ten with
   fix sizes, effort buckets.
3. [01-findings-register.md](01-findings-register.md) — all 114 findings with
   verdicts.
4. Role files by interest; [30-completeness-critic.md](30-completeness-critic.md)
   last.

## Related

- Prior whole-tree reviews:
  [2026-10-04 multirole](../2026-10-04-multirole/README.md),
  [2026-10-04 multirole 2](../2026-10-04-multirole-2/README.md).
- Closure records for this review land with the implementation phase:
  plan `docs/plans/2026-10-06-frontend-cobalt-neo/2026-10-06-frontend-cobalt-neo.md`, dated changelog, and
  issue entries linked to register IDs.
