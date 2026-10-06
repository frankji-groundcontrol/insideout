# Completeness critic — 2026-10-06 frontend cobalt review

Scope: 111 findings + register (7C/21H/51M/32L; 28 confirmed / 1 downgraded / 0
refuted), 10 role scores, skeptic verdicts. Branch `main` @ f185e94. All probes
below are my own read-only checks; the only file written is this report.

## 1. The uncovered dimension: security/privacy

The fan-out brief advertises a dedicated `security` role ("what to rotate today
and what not to expose past localhost"); it was never assigned. Security is
also a repo non-negotiable — AGENTS.md's Privacy section (tokens, PII,
redaction). Total security coverage across 111 findings: exactly one Low
(rel-web-token-storage). Performance got no dedicated pass either (one
incidental Low), and i18n/docs accuracy got only incidental coverage — but
security is the only advertised dimension with no role at all.

Probe (read-only, ~10 min):

- `grep print/debugPrint lib/` → 0 hits; no analytics/telemetry SDK in
  pubspec.yaml. The client never logs tokens. Clean.
- Tokens ride only in `Authorization` headers (client.dart:39,513,596); the
  presence-stream URL carries a random per-load clientId, never a token.
- Unfound issue: reset/magic-link/invite secrets sit in URL query strings
  (router.dart:28,30,31) and persist in browser history and access logs. No
  role reviewed this. Filed below (Low).
- Runner-up gap — performance: the probe found 41 MB of dead CJK fonts shipped
  in the web bundle (see New findings).

## 2. Docs/router claims vs client code

Six concrete claims checked; five hold, one contradicts and is new.

| Claim | Verdict |
|---|---|
| README.md:87 / client/README.md:16 — `flutter run -d chrome` with no dart-define | HOLDS — session.dart:7-10 defaults API_BASE to `http://127.0.0.1:8080/api/v1` |
| client/README.md:30-32 — zh-CN default; translate/theme icons top-right; dock order Capture/Ideas/Version/Commit/Roadmap/Evidence | HOLDS — l10n.dart:4 `defaultLocale = 'zh-CN'`; app.dart:100-109 both IconButtons; product_dock.dart:13 enum order matches |
| DESIGN.md:91-92 — faces load via web/index.html, first frame gated in web_fonts_web.dart | HOLDS — index.html:24 loads all four families; main.dart awaits loadWebFonts() before boot completes |
| DESIGN.md frontmatter hex values (cobalt 50/200/500/700/950, grounds, inks) | HOLDS — all eight match cobalt.dart:62-70 exactly |
| CLAUDE.md key commands (dev.sh preflights env.sh; env.sh init/edit/check/propagate; port 8080) | HOLDS — dev.sh:54 runs `env.sh check`; propagate exists; server default `:8080` |
| DESIGN.md:41 — chip-shade `textColor: "{colors.cobalt-700}"` | **CONTRADICTS code** — cobalt.dart:120 ships `statusNeutralFg: Color(0xFF3A3F4A)` (gray), used by locked/pending/neutral chips (cobalt_chip.dart:30-32). Not filed: design-md-contract-stale names only the radius row. Filed below. |

Already-filed contradictions re-confirmed: local-development.md:11 "Flutter 3"
vs CI/Dockerfile 3.44.0 (pm-flutter-version-unpinned); DESIGN.md rounded lock
vs `radiusControl = 12.0` (pm-rogue-radius-token, design-md-contract-stale).

## 3. Scores that disagree with their own findings

Cross-role anchor: every role with ≥1 Critical scored ≤4.0 except one; roles
with 2+ Highs scored ≤6.5.

- **a11y-visual 4.5 — worst mismatch.** Its own findings are 2 Critical + 3
  High (topbar icons 1.42:1; dock unfinished 1.32:1; every stroke 1.40–2.26:1;
  minimap color-only; admin red 2.72:1) — a severity mix worse than
  first-run-user (4.0; 1C+3H) and responsive (3.0; 1C+2H), yet it scores above
  both. The 4.5 rationale ("sound body-text foundation") measures the type
  system while its own findings say the *primary navigation chrome* fails.
  The score describes a different product than the findings list.
- **feature-implementation 7.2.** Highest of all ten while carrying 2
  confirmed Highs in the core loop (raw enum as the default idea-card body;
  capture/convert fail silently) — above code-quality 6.5, which has zero
  Highs. Its own bottom line is a "last mile" defect list, not a 7.2 profile.
- **project-manager 6.0.** Its two Highs are the two most-seen confirmed
  defects in the review (mid-viewport dock, colliding auth header), plus the
  most Mediums of any role (7). first-run-user scored 4.0 for essentially the
  same headline failures; 6.0 is ~2 points generous.

## 4. Verdict spot-checks (3 re-derived from code)

- **dock-unfinished-blue-on-blue** — product_dock.dart:166-178 paints icon,
  11px label, and 1.5px border with `statusWarnFg`; light theme statusWarnFg =
  #1D4ED8 on dock brand #3B5BFF: L 0.107 vs 0.156 → **1.32:1** (register
  1.32:1). Verdict agrees.
- **rel-hydrate-wipes-tokens** — session.dart:45-65: inner try `_me()` →
  catch → `_refresh()` + retry; outer catch (:58-60) `user = null; await
  clear();` with no status discrimination, so a transport error deletes stored
  tokens; `clear()` → token_store delete(). Verdict agrees.
- **admin-hardcoded-red-dark-unreadable** — admin_section.dart:167 literal
  `Color(0xFFB3261E)` vs dark surfaceRaised #141820: L 0.111 vs 0.0091 →
  **2.72:1** (register 2.72:1); the existing token dark fgDanger hue200
  (#C7D2FE) computes ~13:1. Verdict agrees.

No verdict-vs-measurement mismatch in the three sampled; the one DOWNGRADED
verdict (dock-dead-clicks-evidence-version) also matches the code read
(`stay()` navigates when a project is known; product_dock.dart:91-96).

## 5. New findings

Three new findings filed by this pass (full detail in the handoff JSON):

1. **critic-web-bundle-ships-41mb-unused-fonts (Medium)** — pubspec.yaml:66
   bundles both CJK variable fonts (24M + 17M) into every web build; web never
   loads them (native_fonts.dart:12 `if (kIsWeb) return;`; index.html:24 uses
   the Google Fonts CDN), yet they sit in the shipped artifact
   (build/web/assets/assets/fonts/ = 41M) and its own comment says web "must
   never download" them. No performance role existed to catch this.
2. **critic-auth-secrets-in-url-query (Low)** — router.dart:28,30,31 put
   invite/reset/magic-link secrets in URL query strings; they persist in
   browser history. Consume then replace the history entry.
3. **critic-design-md-chipshade-textcolor (Low)** — DESIGN.md:41 prescribes
   cobalt-700 chip text; the shipped neutral-chip token is #3A3F4A
   (cobalt.dart:120). Pick one authority during the DESIGN.md refresh.
