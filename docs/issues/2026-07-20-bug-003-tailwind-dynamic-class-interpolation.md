# BUG-003: Template-literal-constructed Tailwind classes never get generated (JIT can't see them)

**Found**: 2026-07-20, during the InsideOut rewrite (P6), while codemoding `src/pages/projects/[id].vue`'s update-kind toggle buttons onto the semantic design tokens — caught during self-review before it ever ran, but worth recording since the pattern is easy to reintroduce.

**Symptom (would have been)**: the active/selected progress-blocker-note toggle button would render with no background/text color at all in production, because the utility classes needed were never emitted into the compiled CSS.

**Root cause**: the first draft computed the class dynamically — `` `bg-status-${tone}-bg text-status-${tone}-fg` `` — where `tone` is a runtime value (`'success' | 'danger' | 'neutral'`). Tailwind's content scanner does static text scanning of source files for whole class-name tokens; it does not execute JavaScript, so it can never see the fully-interpolated class names (`bg-status-success-bg`, etc.) — only the un-interpolated template literal itself appears in the source. None of those classes get generated.

**Fix**: replaced the interpolated template literal with a static lookup object (`Record<Kind, string>`) whose values are complete, literal class-name strings for every possible case, so each full class name appears verbatim somewhere in the source for the scanner to find. See `kindActiveClasses` in `src/pages/projects/[id].vue`.

**Why it matters**: this is a general Tailwind JIT pitfall, not specific to this file — any `:class="\`...-${dynamicValue}\`"` construction should be treated as suspect and rewritten as a static per-case lookup. Worth a quick grep (`` `[a-z-]*\$\{ `` in `.vue` templates/scripts) before adding new dynamic-tone UI.
