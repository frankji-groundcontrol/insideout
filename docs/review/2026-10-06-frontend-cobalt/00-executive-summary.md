# Executive summary — 2026-10-06 frontend cobalt review

- Repo: `insideout` — branch `main` @ `f185e94`
- Date: 2026-10-06
- Reviewers: 10 role agents + skeptic clusters + 1 completeness critic
- Findings: 114 (7 Critical / 22 High / 50 Medium / 32 Low)
- Verdicts on High/Critical: 28 confirmed, 1 downgraded, 0 refuted
- **Mean score: 5.22 / 10**

*Corrected 2026-10-11: the printed breakdown sums to 111, not 114 — it counts
only the reviewer findings as filed; 114 includes the critic's +3 rows (the
register header after the one downgrade reads 7C/21H/52M/34L = 114). See
[README](README.md).*

## Verdict

The engineering core is healthy — session management, API wiring, feature
breadth, and module structure all score 6.5–7.2. The product fails where the
user sees it: navigation chrome geometry, contrast, and the complete absence
of the adopted Cobalt-Neo design mechanics. Every Critical is in the visual
layer; every one has a known, small fix. Nothing requires re-architecture.

## Scores

| Role | Score | One sentence |
|---|---|---|
| feature-implementation | 7.2 | The advertised loop exists end-to-end; the last mile (raw enums, silent write failures) is missing |
| code-quality | 6.5 | Clean, modular code with repo-wide formatting drift and no gate |
| testing | 6.5 | Real tests exist, but the harness hides exactly the bugs users hit |
| fullstack | 6.5 | Sound client/server wiring; six write actions can fail without telling anyone |
| project-manager | 6.0 | Not shippable while the two most-seen defects stand |
| reliability-ops | 5.5 | No timeouts anywhere; a flaky network can log the user out |
| a11y-visual | 4.5 | Body type is fine; primary chrome fails contrast across the board |
| first-run-user | 4.0 | First thing a new user sees is a dock floating at screen center |
| responsive | 3.0 | One layout mechanism breaks every real viewport |
| design-fidelity | 2.5 | Zero of four signature Cobalt-Neo mechanics implemented |

## Top ten by user impact

| # | Finding | Where | Sev | Fix size |
|---|---|---|---|---|
| 1 | Dock pill renders at viewport center (root cause + all pages) | `product_dock.dart:26-28`, `app.dart:135` | C×2 | **S** — `Center` → `Align(bottomCenter, heightFactor: 1.0)`, ~3 lines |
| 2 | Top-bar icons invisible over navy (1.42:1) | `app.dart:90` | C | **S** — surfaceRaised pill behind the icon row, ~10 lines |
| 3 | Radius decree inverted (12/20/28/999 vs 0px) | `cobalt.dart:72-75` | C | **S** tokens / **M** sweep of hardcoded radii |
| 4 | No cast-shadow system | `cobalt_theme.dart:54-61` | C | **M** — 2 tokens + 6 call sites |
| 5 | No press-the-block motion | `ink_motion.dart:4` | C | **M** — motion tokens + shared pressable wrapper |
| 6 | Dock unfinished state blue-on-blue (1.32:1) | `product_dock.dart:166` | C | **S** — amber warn token + swap, ~5 lines |
| 7 | Every stroke token fails non-text contrast | `cobalt.dart` (stroke ramp) | H | **S** — 3 hex values + WCAG script check |
| 8 | Light mode is the default | `appearance.dart:36` | H | **S** — 1 line + DESIGN.md note |
| 9 | Hydrate wipes tokens on transport error | `session.dart:58` | H | **S** — clear only on 401/403, ~4 lines |
| 10 | Six write actions fail silently | ideas/workspace/project pages | H | **M** — guarded-action helper + 6 wraps |

## What each role would say at the door

- **code-quality:** "Run the dart format sweep and gate it in CI first —
  55/68 files drift; it is mechanical and unblocks every later diff."
- **testing:** "Add one 1440×900 LandingPage + AuthDoor zh-CN golden —
  the cheapest test, and it would have caught the modal collision."
- **reliability-ops:** "Two sets of timeouts: Dio 10s/30s plus hydrate 15s,
  and a 75s idle timer on the coach stream."
- **fullstack:** "Wrap the six write actions in a guarded-action helper:
  catch → localized SnackBar, success → reload."
- **feature-implementation:** "Fix the advertised loop's first screen: raw
  enum as idea status, and capture/convert failing silently."
- **project-manager:** "Do not release until the mid-viewport dock and the
  AuthDoor collision are fixed, repackaged, and the signed-in hosted gate
  passes."
- **first-run-user:** "Bind the dock height with a SizedBox and switch
  Center → Align; verify at 1440×900 and 390×844."
- **design-fidelity:** "Draft the Cobalt-Neo token table first — gray ramps,
  border widths, cast shadows, press motion, semantic policy, dark-first —
  every code change is blocked on token values."
- **a11y-visual:** "Retune the strokes and dock warn/selected states and
  prove them with the WCAG script; that clears the widest failure surface."
- **responsive:** "One line — `Center` → `Align(bottomCenter)` in
  product_dock.dart:28 — then re-shoot every page."

## Effort

| Bucket | Items | Estimate |
|---|---|---|
| Wave 1 — one-line & token-value fixes | 8 | ~1 day |
| Wave 2 — dock geometry & clearance | 6 | ~2 days |
| Wave 3 — Cobalt-Neo token system + DESIGN.md | 7 | ~3 days |
| Wave 4 — write reliability & stream timeouts | 4 | ~2 days |
| Wave 5 — non-color a11y channels | 2 | ~1 day |
| Wave 6 — test harness, goldens, format gate | 4 | ~2 days |
| Wave 7 — bundle & URL hygiene | 2 | ~0.5 day |
| **Total** | **33** | **~11–12 engineer-days** |

## What not to do yet

- Do not redesign below 390px width or touch the server; this review covers
  the web client's visual and interaction layer.
- Do not refactor the theme architecture beyond adding tokens — the token
  file structure itself was not found wanting.
- Do not fix the raw-enum status display by hiding the status — localize it.
- Do not chase performance beyond the 41MB font strip; no other hotspot was
  measured.
- Do not act on score-vs-findings mismatches (a11y 4.5, feat 7.2, PM 6.0) —
  they are calibration notes for the next review, not work items.
