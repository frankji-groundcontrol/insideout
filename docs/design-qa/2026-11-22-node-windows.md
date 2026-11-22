# 2026-11-22 — Node information in movable, resizable windows

## Comment

> "I hope when I clik in the node, information can be shown in a movable and
> sizable popup window."

## What was wrong

Clicking a branch node left the page for the node's own roadmap. Clicking a
leaf opened a fixed modal. Neither kept the plan in view, and neither let two
nodes be read side by side.

## Resolution

The stated assumptions are in the
[plan](../plans/2026-11-22-node-windows/2026-11-22-node-windows.md).

- **Popup:** a click opens a floating window over the page.
- **Handling:** the window is dragged by its title bar and resized from its
  corner and edges. Several can be open at once.
- **Contents:** the person-in-charge glance and the node's details.
- **Drilling in:** "Open roadmap" or a double-click goes to the node's page.
- **Phones:** on a phone, the window is a full-screen sheet.

## Status

Fixed: see the [changelog](../changelogs/2026-11-22-node-windows.md).
