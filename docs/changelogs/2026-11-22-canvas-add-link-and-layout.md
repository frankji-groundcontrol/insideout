# 2026-11-22 — Add, link and lay out steps on the Canvas

Plan: [add and link steps on the Canvas](../plans/2026-11-22-canvas-add-and-link/2026-11-22-canvas-add-and-link.md).
Owner comments: "what if I want to add a next for this layer", "what if I
want to add a parallel node? and what if I want to link the node to other
node?", "how this layout is determined? please come up with a layout
algorithm for good representation"
([design QA](../design-qa/2026-11-22-canvas-add-link-and-layout.md)).

| Change | Commits |
| --- | --- |
| **Next step.** A "+ Next step" slot sits after the Canvas's last column. It adds a step to the shown layer that waits on every step nothing in the layer waits on yet. | `210f101c` |
| **Step after, parallel step.** A step's menu gains "Add a step after", for a step that waits on this one, and "Add a parallel step". A parallel step waits on what this step waits on and is waited on by what waits on it, so it takes the same column and merges in the same place. A link the caller cannot write is skipped, and a snack bar names the step. | `210f101c` |
| **Drag to link.** Each step has a handle on its right edge. Dragged onto another step, it makes that step wait on this one, and a line follows the pointer. A drop on empty canvas does nothing. | `210f101c` |
| **Layered layout.** Columns stay by longest path, as the server's `roadmap steps` gives them. Each column is now ordered to cross as few links as it can, using barycenter sweeps and keeping the best order seen. Each box moves level with what it links to, and long links get one lane per column. On the Spark roadmap this removes every crossing: Chat sits level with Matching, and Launch sits among what feeds it. | `210f101c` |

## Notes

- The add actions show only where the caller may write the layer's parent.
  They do not show on Mine, whose steps come from several layers, or on an
  opened leaf.
- The layout is client-only; the order inside a column is not part of the
  shared steps contract.
- Where a step and a lane compete for room, the step wins. A lane can then
  bend, as the Matching → Launch link does under Chat.

## Verification

- `flutter analyze`: 8 infos (baseline).
- Full gates on a fresh database: build, vet, gofmt, Go tests with and without a database (0 skipped), dart format, `flutter test` (770 passed), doc checks and redaction all pass.
- New tests:
  - `roadmap_flow_layout_test.dart`: 5 tests of the pure layout.
  - `roadmap_canvas_layout_test.dart`: 2 tests of the Spark shape and an
    uncrossing.
  - `roadmap_canvas_edit_test.dart`: 6 tests covering the slot, step after,
    parallel step with a skipped link, drag link, empty drop, and a reader.
- Live as Maya on the Spark roadmap:
  - The flow draws with no crossings, and "+ Next step" is level with Launch.
  - A drag from Safety & trust's handle draws the line without panning the
    view, and a drop on empty canvas changes nothing.
