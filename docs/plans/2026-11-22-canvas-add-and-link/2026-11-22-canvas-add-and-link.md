# 2026-11-22 — Add and link steps on the Canvas

> "what if I want to add a next for this layer"
> "what if I want to add a parallel node? and what if I want to link the node
> to other node?"

The owner's screenshot marks an empty slot after Launch, the last step of the
flow. Before this plan, the Canvas could not add a step. "New item" in the
header adds a top-level goal, not a step in the shown layer. A link could
only be made from the step menu's "Depends on…" picker.

## Decisions

These are the owner's answers to the coordinator's questions:

- **Next step:** a "+ Next step" slot sits after the last column. It adds a
  step to the shown layer (same parent as the steps). The new step waits on
  every step that nothing in the layer waits on.
- **Step after:** each step's menu gains "Add a step after", which adds a
  step that waits on that step: a new branch from it.
- **Parallel step:** each step's menu gains "Add a parallel step", which adds
  a step that waits on what the chosen step waits on (its lifted step links)
  and is waited on by what waits on it. Links the caller cannot write are
  skipped, and the app says so.
- **Drag to link:** each step has a handle on its right edge. Dragging it
  onto another step makes that step wait on this one. A line follows the
  pointer. "Depends on…" stays in the menu for links to nodes deeper in the
  tree.

The coordinator decided the following without asking:

- The add actions show only where the caller may write the layer's parent
  (a plan writer for top-level steps). They do not show on Mine, whose
  steps come from different layers, or on an opened leaf, whose only step
  is the node itself.
- These are client-only changes over the existing create-node and
  add-dependency routes. The CLI and MCP already have both commands.
- **Layout** (owner: "please come up with a layout algorithm for good
  representation"):
  - Columns stay by longest path, so column 0 still means "can start now"
    and the server's steps order is unchanged.
  - The drawing becomes a layered layout: lanes for long links, barycenter
    sweeps to uncross each column, and each box aligned with the boxes it
    links to.
  - A helper agent wrote the pure layout module in parallel with the add
    and link work, since it touches only new files.

## Checklist

- [x] `addRoadmapStep` and the next-step slot; the menu items.
- [x] The link handle, drag line and drop.
- [x] Tests: slot, step after, parallel step (skipped link), drag link.
- [x] The layered layout (pure module plus render box) wired into the Canvas.
- [x] Gates green, a live check on the Spark data, records written, PR merged.
