# Practice: Simulate a user across CLI, Web, and MCP

**Date**: 2026-10-06

## Trigger

A claim that a feature works "on all surfaces," or a product promise
(clarity from day zero, a reader who must not change the plan) that
code review cannot settle. Reading the handler is not a substitute for
one person doing the thing three ways.

## Sequence

1. Isolate it. A fresh local database on the local Postgres, never the
   remote one, never a container this machine does not have. Connection
   strings and tokens live in a `0600` directory outside the repo.
   Print lengths and shapes, never values.
2. Boot the real server (dev mailer, no real email) and the release web
   build. Debug `flutter run` injects a client that crashes; the
   release bundle does not. Export `INSIDEOUT_API` — the CLI defaults
   to `:8080`, which is not this server.
3. One persona, one case, three surfaces, in that order: do the thing
   on the surface that persona actually uses, then repeat the same
   read on the other two. Compare the facts (titles, dates, counts,
   role), not the formatting.
4. Drive the web client through a real browser. Flutter web is one
   canvas: coordinate clicks miss unless you first enable semantics and
   click the element's bounding rect. A full navigation to a deep route
   against the static file server returns that server's 404 and drops
   the session; change the route inside the running app.
5. A write you did not mean is data. Undo it as the person who is
   allowed to, and record that the gate let the wrong person through.

## Verification

A parity table: one row per fact, three columns, and a note wherever
the columns disagree. Disagreements become issues, not silent fixes.

## Failure signals

- CLI answers 404 against a server you know is up — `INSIDEOUT_API`
  was not exported, so the call went to the default port.
- Web sits on the boot spinner with no API traffic — a font load never
  settled and the boot deadline never got to run
  ([BUG-016](../issues/2026-10-06-bug-016-web-boot-gate-hangs-on-font-load.md)).
- A "reader" case ends with a new node — membership was treated as
  permission to write
  ([BUG-020](../issues/2026-10-06-bug-020-workspace-member-writes-the-plan.md)).
