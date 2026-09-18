# 2026-09-17 — Cobalt layouts accepted as references

After six layout ideas, the user said:

> Looks better, take them as references and proceed to open-design

## Mapping

| Ref | File | Surface |
| --- | --- | --- |
| A | `REFERENCE-A-typewall.jpg` | Ideas |
| B | `REFERENCE-B-bands.jpg` | Roadmap |
| C | `REFERENCE-C-stack.jpg` | Commit |
| D | `REFERENCE-D-split.jpg` | PRD + Coach |
| E | `REFERENCE-E-dock.jpg` | Chrome |
| F | `REFERENCE-F-index.jpg` | Home |

Cobalt shade scale + light/dark stay. OpenDesign built
`insideout-cobalt.html` from that map (light + dark, dock chrome,
English-only copy). Live-walked: capture an idea, request review,
human Commit, add a Now item, toggle theme. Flutter waits.

## Follow-up (same day)

> First of all, I do not like one card, small texts, a lot of blank style, fix them. Enlarge the texts, make it clear, let user have focus

The floating single card and 15px type are rejected. Prototype
revised: full-bleed surfaces, 20px reading type, large titles, one
focal block per screen, no empty field as decoration.

## Follow-up (same day)

> I still prefer the previous design, just adjust layout and font size

The density rewrite (no cards, type-rows, full-bleed) is reverted.
Restore the A–F card / poster / stack / band grammar. Only enlarge
type (body 17.5) and tighten unused padding.

## Follow-up (same day)

> send to open design as reference, make design language and page prototypes

The accepted card screens are pinned as `LOCK-home-light.jpg` …
`LOCK-roadmap.jpg`. Language lives in `DESIGN_LANGUAGE.md` and
`cobalt-language.html`. Page prototypes stay in `insideout-cobalt.html`.

## Follow-up (2026-09-18)

> yes, use this one for flutter, give me the goal prompt

Cobalt card lock is the Flutter direction. Goal prompt:
[`docs/plans/2026-09-18-cobalt-flutter-goal.md`](../plans/2026-09-18-cobalt-flutter-goal.md).
In-repo lock screens:
[`docs/design-system/cobalt/`](../design-system/cobalt/README.md).
Implementation plan:
[`docs/plans/2026-09-18-cobalt-flutter.md`](../plans/2026-09-18-cobalt-flutter.md).
