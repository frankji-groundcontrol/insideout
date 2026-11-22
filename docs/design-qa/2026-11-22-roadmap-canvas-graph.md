# 2026-11-22 — Roadmap Canvas: an ordered flow of steps

## Comments

> "Canvas show with a hierarchy, no drop downs. user can click to see the tree
> that node represents. for example, I am the boss, I would like to see global
> and general things being managed by my descendant managers. the work tree can
> be in parallel, and merge somewhere. each flow connected nodes. when boss
> clicked a node, then he/she can see what the manager will see as root."

> "In canvas I want 1. drag to move center view 2. zoom in and out"

> "I do not want to see hierarchy on canvas, only things in order. hierarchy is
> represented when I click into a certain step, namely node"

The owner first said "merge" meant real dependencies, and asked for two levels
of depth. The third comment replaced the two-level hierarchy with one level.

## What was wrong

- **One band:** a built project has one root, so the band Canvas was a
  single 300px column with fold toggles and a minimap.
- **First rework too wide:** the two-level graph ran about 5000px wide, and
  the root opened off screen.
- **Links misdrawn:** links that skipped a column ran under the boxes in
  between, so they read as links into those boxes.

## Resolution

- **Steps:** the Canvas shows the steps under the current node. With one
  root, the steps are its children. With several roots, the roots are the
  steps. Mine shows the caller's topmost owned nodes.
- **Order:** dependencies between any nodes are lifted to the steps that hold
  them. Steps are placed left to right by longest path, with parallel steps
  stacked in one column. Steps that wait on each other share a column under
  a loop marker.
- **Links:** a link that skips a column runs through a lane reserved under
  that column. Links into one step land at separate points on its left edge.
- **Opening a step** shows its own steps, with the breadcrumb leading back
  up. A leaf opens its detail.
- **Pan and zoom:** drag to pan, wheel or pinch to zoom (0.3x to 2x), and
  +/−/fit buttons.
- **Pinned by:**
  [`roadmap_canvas_test.dart`](../../client/test/features/project/roadmap_canvas_test.dart),
  [`roadmap_canvas_view_test.dart`](../../client/test/features/project/roadmap_canvas_view_test.dart),
  [`roadmap_canvas_loop_test.dart`](../../client/test/features/project/roadmap_canvas_loop_test.dart)
  and
  [`roadmap_canvas_route_test.dart`](../../client/test/features/project/roadmap_canvas_route_test.dart).

## Files

- [`client/lib/features/project/roadmap_canvas.dart`](../../client/lib/features/project/roadmap_canvas.dart)
- [`client/lib/features/project/roadmap_step_flow.dart`](../../client/lib/features/project/roadmap_step_flow.dart)
- [`client/lib/features/project/roadmap_flow_routes.dart`](../../client/lib/features/project/roadmap_flow_routes.dart)
- [`client/lib/features/project/roadmap_painters.dart`](../../client/lib/features/project/roadmap_painters.dart)
