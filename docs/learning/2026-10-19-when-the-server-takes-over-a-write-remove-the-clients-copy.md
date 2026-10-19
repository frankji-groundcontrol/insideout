# When the server takes over a write, remove the client's copy in the same wave

**Date:** 2026-10-19
**Source:** [multirole review 4](../changelogs/2026-10-18-multirole-review-4.md), finding FEAT-02 (accepting a proposal)
**Fixed in:** [wave 2, proposal apply](../changelogs/2026-10-18-rv4-remediation/w2-proposal-apply.md) (server half) and [wave 3, client apply](../changelogs/2026-10-18-rv4-remediation/w3-client-apply.md) (client half)

## The lesson

Wave 2 made the server apply an accepted proposal's items inside the decision's
own transaction, once (`proposal_items.applied_at`). The web updates feed still
ran its own apply after the decision: it listed the roadmap and called
`createRoadmapNode` for every `add_node` item. Both ran, so every web accept
built each node twice. The server fix was correct and verified; the finding
stayed `PARTIAL` until the client step was deleted, one wave later.

When a server starts doing what a client used to do in several calls, the
change is not finished until the client's copy is gone. Two defences:

- Put the client file in the same cluster's owned files at planning time. The
  plan omitted it although its brief said "remove the client-side apply".
- Test the client for what it must not do (create no node itself) and for what
  replaces it (refresh after the decision). Dropping the `onChanged()` call kept
  the other two assertions green, so the read count is asserted too.

## When to apply

- Moving any multi-step write behind one server call: grep the client for every
  caller of the old steps, and delete or flag them with the server change.
- Reading a "server fix verified" report: ask whether a second writer remains.

## Evidence

The red test failed on `api.created` (the client created the node itself); after
the fix a live accept with `apply: true` leaves one node and a repeat adds none,
while the removed client sequence replayed against the same API left two nodes
with the same title. The web widget path is covered by the widget test with the
repository's fake `ApiClient`, not by a browser.
