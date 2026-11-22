# 2026-11-22 — Node information in movable, resizable windows

> "I hope when I clik in the node, information can be shown in a movable and
> sizable popup window."

The coordinator made these assumptions and told the owner, who can redirect:

- **Popup:** clicking a node anywhere on the roadmap page (a Canvas step, a
  Bands card or a Progress item) opens a floating window over the page.
- **Move, resize, close:** the title bar drags the window and keeps it inside
  the viewport. A corner and edge handles resize it, with a minimum of
  320×240. A × button closes it.
- **Several windows:** several can be open at once to compare colleagues'
  areas. The one clicked comes to the front. Opening a node that already has
  a window brings that window forward.
- **Contents:** the node's details as the current detail modal shows them,
  plus the person-in-charge glance from `GET /roadmap/{nid}/glance`.
- **Drilling in:** an "Open roadmap" button goes to the node's page, as a
  click did before, and double-clicking a node goes there directly.
- **Leaves:** a leaf's window replaces its detail modal.
- **Phones:** below the narrow breakpoint, the window is a full-screen sheet.

Client only; the glance and node reads already exist.

## Checklist

- [x] W1 (client): the window layer, contents, opening from Canvas, Bands and
      Progress, the double-click drill, the phone sheet, and tests.
- [x] Gates green, a live check on the Spark data, records written, PR merged.
