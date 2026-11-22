# 2026-11-22 — Open a node as the person in charge sees it

> "when I click in, I want to see the roadmap of this node, the view the person
> who's in charge. so I can view things as other colleagues"

The owner chose this design. Opening a node shows that node's roadmap as its
person in charge sees it:

- **A banner** says whose view it is, with their first glance for the node:
  Now items with pressure, what it waits on, and what it is blocking.
- **The page** opens on Progress scoped to the node.
- **The fourth toggle** shows that person's own nodes ("Priya's"), so the
  viewer can see things as that colleague does.

Opening a node deeper switches the banner to that node's person.

Two clusters run in parallel from `main` at `dd5ef7dd`, with the contract below
fixed. The coordinator integrates on `feat/view-as-in-charge`, runs the gates
once, checks the Spark data live, and merges one PR with `[skip ci]`.

## Contract

`GET /api/v1/roadmap/{nid}/glance` answers 200:

```json
{"node": <node view>,
 "inCharge": {"userId": "...", "name": "...", "via": {"id": "...", "title": "..."} | null, "projectOwner": false} | null,
 "now": [<progress item>],
 "waitsOn": [{"id","title","status","ownerName"}],
 "blocking": [{"id","title","status","ownerName"}]}
```

- **inCharge** is resolved in order:
  1. the node's owner;
  2. else the nearest ancestor with an owner, with `via` naming that
     ancestor;
  3. else the project owner, with `projectOwner: true`.

  It is `null` only when none of these exists.
- **now, waitsOn, blocking** mean what they mean on a `mine` root, computed
  for the node's subtree (`server/internal/store/roadmap_first_glance.go`).
- **Errors:** 404 when the node is not visible. A bad id answers 400.
- **Access:** agent tokens may call it. Reads stay workspace-wide.
- **Surfaces:** the Go client `RoadmapGlance(nodeID)`, CLI `roadmap glance
  <node>`, and MCP `roadmap_glance(node_id)`.

## Checklist

- [ ] V1, server: the route, the in-charge resolution with `via`, the Go
      client, CLI, MCP, docs, and real-PostgreSQL tests on the Spark world.
- [ ] V2, client:
  - The banner on an opened node, saying "Priya Nair's view · in charge", or
    "Your view" for yourself.
  - A Canvas step tap and a My work tap open the node on Progress.
  - The fourth toggle shows the person in charge's nodes.
  - A Canvas level narrower than the screen is centred.
- [ ] Integrated gates green, checked live as Maya opening Onboarding (Priya)
      and Matching (Leo); records written; PR merged.
