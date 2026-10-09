# A transparent mark needs a zero-alpha corner

## What was learned

A file named like a transparent PNG is not transparent until a corner
pixel has alpha 0. Raster generators often return a JPEG, or paint a
checkerboard into the pixels. Either one is an opaque picture of
transparency.

The same pass showed a second geometry trap: translating a frame or an
open stroke to fake a hard shadow draws a second copy inside the
opening. The slab only reads as a cast when the shape under it is solid.

## Evidence

The first four generations for the InsideOut icon were JPEGs. Two of
them had a gray-and-white checkerboard in the background pixels. The
redrawn SVGs, rasterized with `rsvg-convert`, report RGBA and corner
alpha 0
([changelog](../changelogs/2026-10-06-app-icon-candidates.md)).

An earlier pierce and trace put a navy copy of the stroke 16–32px
down-right. The copy showed up as a second frame inside the hole and
as stray rectangles in the gaps. Dropping that copy, and keeping the
slab only under the solid squares in breakout and shift, removed it.

## Scope

App icons, logos, and any mark that must sit on both a light and a
dark ground. Also any neo-brutalist "4px 4px 0" shadow applied to a
ring, a stroke, or a path with a gap.

## When to apply it again

Before calling a generated icon transparent, sample a corner pixel.
Before offsetting a shape to make a hard shadow, check whether the
shape has a hole. If it does, the offset will paint inside that hole.
