# Two ramp steps can fail both grounds

## What was learned

A second cobalt that survives the dark ground can still fail to separate
from the primary, and the navy that separates from the primary disappears
on the dark ground. A two-tone transparent mark has to clear three
checks: each color against the light ground, each color against the dark
ground, and the two colors against each other.

## Evidence

Relative-luminance contrast of the Cobalt tokens
([`cobalt.dart`](../../client/lib/theme/cobalt.dart)):

| Pair | Contrast |
| --- | --- |
| `#1E2A5A` on `#0B0D12` | 1.42 |
| `#1E2A5A` on `#F7F8FA` | 12.88 |
| `#1D4ED8` on `#0B0D12` | 2.90 |
| `#1D4ED8` on `#F7F8FA` | 6.31 |
| `#3B5BFF` on `#1D4ED8` | 1.32 |
| `#3B5BFF` on `#1E2A5A` | 2.69 |

The second icon round uses only `#3B5BFF` and `#1D4ED8`
([changelog](../changelogs/2026-10-06-app-icon-candidates.md)). Sampled
pixels on `weave.png`, `through.png`, `rift.png`, and `drop.png` are
those two colors. They stay visible on both grounds. They do not
separate from each other.

## Scope

Transparent marks, app icons, and any graphic that sits on an unknown
wallpaper. Not a rule for body text on a solid cobalt button, where the
pair is cobalt on white.

## When to apply it again

Before calling a two-tone mark readable, compute the three contrasts.
If the shades are adjacent ramp steps, do not rely on the boundary
between them to carry the idea.
