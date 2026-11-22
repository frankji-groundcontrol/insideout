# 2026-11-22 — Node information in movable, resizable windows

Plan: [node windows](../plans/2026-11-22-node-windows/2026-11-22-node-windows.md).
Owner comment: "I hope when I clik in the node, information can be shown in a
movable and sizable popup window"
([design QA](../design-qa/2026-11-22-node-windows.md)).

| Change | Commits |
| --- | --- |
| A tap on a Canvas step, a Bands card or a Progress item opens the node in a floating window. The window has a title bar with a status chip, "Open roadmap" and close. The title bar drags it, kept on screen. The right edge, bottom edge and corner resize it, from 320×240 up to the viewport. Several windows can be open, the touched one comes to the front, and Escape closes the front one. The body shows the person-in-charge glance, then the node's details, reusing the detail modal's content. A double-click or "Open roadmap" goes to the node's own page. On a phone, the window is a full-screen sheet. | `9dc9bbbe` |
| A window shows every waits-on and blocking title. The banner and My work keep two lines. | `7b43c83d` |

## Notes

- **A single tap waits 300 ms** before opening, so a double-click can drill
  in instead.
- **Node cards outside the roadmap page** ignore taps; the page is their only
  use today.
- **Not yet handled:** windows are not re-fitted when the browser shrinks, and
  resizing is by pointer only.

## Verification

- Go build, vet and gofmt: clean.
- `go test ./...`: passes without a database, and on a fresh database with 0
  skipped tests.
- `flutter analyze`: 8 infos (baseline). `flutter test`: 744 passed.
- Doc and redaction checks: pass.
- Live as Maya on the Spark data:
  - Onboarding & profiles opened as Priya's view.
  - The window was dragged and Launch opened beside it, reading as Maya's
    own view as project owner.
  - The Launch window was resized, and its full waits-on list shows.
