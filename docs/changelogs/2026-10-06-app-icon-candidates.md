# 2026-10-06 — App icon candidates

Two rounds of transparent app-icon marks, not a chosen icon. Nothing in the
Flutter client, server, or token contract changed. Not deployed.

## What changed

Drew two rounds of SVG masters and 1024×1024 PNGs under
[icon-candidates/](../design-system/cobalt/icon-candidates/).

The first round is `breakout`, `pierce`, `shift`, and `seam`: one
square, shifted. Colors are the cobalt ramp (`#3B5BFF`, `#1D4ED8`,
`#1E2A5A`) plus ink `#0B0D12` on the seam. A hard offset slab sits
only under a solid block.

The user called that too plain. The second round is the pick set:
`weave` (two rings, one crossing over and one under), `through` (a
frame with a brighter square filling the window and stepping past the
corner), `rift` (a square pulled apart on the diagonal), and `drop`
(the base stepped down by exactly one stroke). Second-round colors are
`#3B5BFF` and `#1D4ED8` only, so both shapes stay visible on a light
ground and a dark one. Corners stay square.

The
[cobalt lock index](../design-system/cobalt/README.md) points at the
folder. The user's comments and the still-open pick are in
[design-qa](../design-qa/2026-10-06-app-icon.md).

## Verification

`rsvg-convert` wrote each PNG. Pillow reports mode RGBA and a corner
pixel alpha of 0 on every file, including the second round
([lesson](../learning/2026-10-06-a-transparent-mark-needs-a-zero-alpha-corner.md)).
Relative luminance of the second-round pair is 1.32:1, so the shades
do not separate; each shade does clear both grounds
([lesson](../learning/2026-10-06-two-ramp-steps-can-fail-both-grounds.md)).
No app-icon asset was replaced, and no Flutter test was run.

## Operator

Nothing to migrate or deploy. The install is filed and blocked on a
pick
([issue](../issues/2026-10-06-app-icon-not-installed.md),
[plan](../plans/2026-10-06-app-icon/2026-10-06-app-icon.md)).

*Superseded 2026-10-06: the pick landed the same day — the user named
`breakout` the logo and it was installed on every platform
([record](2026-10-06-app-icon-breakout.md)).*
