# Raw role outputs

This directory holds the ten role agents' raw, unedited review outputs from
the 2026-10-06 frontend & Cobalt design-language review, preserved exactly as
submitted before the lead's dedup, skeptic verdicts, and editing. The
polished counterparts are the numbered role reports one level up (see the
[review README](../README.md)); where a raw finding was later downgraded,
refuted, or merged, [01-findings-register.md](../01-findings-register.md) is
the authoritative record. The ten files, in role order:

- [fullstack.md](fullstack.md) — full-stack wiring pass: client behavior,
  endpoint coverage, and the env contract behind every named control.
- [project-manager.md](project-manager.md) — ship-readiness verdict (6.0/10),
  from clone-to-run through the two release-blocking defects.
- [first-run-user.md](first-run-user.md) — new-user walkthrough against the
  wave-1 captures; the first three minutes scored 4/10.
- [feature-implementation.md](feature-implementation.md) — every
  frontend-facing claim in README.md and PRODUCT.md checked against the
  implementing code.
- [code-quality.md](code-quality.md) — code-quality pass over all 68 files of
  `client/lib` plus the test suite, against the repo's own modularity rule.
- [testing.md](testing.md) — the client test suite as actually run: the exact
  `flutter analyze` tail and what the harness hides.
- [reliability-ops.md](reliability-ops.md) — how the client degrades when the
  network misbehaves; the silent cases are the worst.
- [design-fidelity.md](design-fidelity.md) — gap analysis against the
  Cobalt-Neo target: no shadow system, inverted radius decree, no press
  motion.
- [a11y-visual.md](a11y-visual.md) — accessibility and contrast pass: WCAG
  ratios computed for every token pair across both themes.
- [responsive.md](responsive.md) — responsive/overflow pass; isolates the
  mid-screen dock's root cause in Scaffold/Center layout mechanics.
