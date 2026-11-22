# 2026-11-22 — Open a node as its person in charge sees it

Plan: [view as the person in charge](../plans/2026-11-22-view-as-the-person-in-charge/2026-11-22-view-as-the-person-in-charge.md).
Owner comment: "when I click in, I want to see the roadmap of this node, the
view the person who's in charge. so I can view things as other colleagues"
([design QA](../design-qa/2026-11-22-open-a-node-as-its-person-in-charge.md)).

| Area | Change | Commits |
| --- | --- | --- |
| Server | `GET /api/v1/roadmap/{nid}/glance` returns the node, its person in charge, and their first glance for the node's subtree: `now`, `waitsOn` and `blocking`. The person in charge is the node's owner, else the nearest owned ancestor (named in `via`), else the project owner. Covers the Go client `RoadmapGlance`, CLI `roadmap glance <node>` and MCP `roadmap_glance` ([architecture](../architecture/roadmap-tree.md)). | `603613e0` |
| Web | An opened node shows a banner: "PRIYA NAIR'S VIEW · in charge", or "YOUR VIEW", with a via or project-owner note. Under it are their Now items with pressure chips and "Waits on" and "Blocking" lines. The page opens on Progress for the node: Canvas step taps and My work taps no longer force the Canvas. The fourth toggle becomes "Priya's" and shows that person's nodes across the whole project. A narrow Canvas level is centred. | `6e27bdf9`, `776eb22b` |

## Verification

- `go build`, `go vet`, `gofmt`: clean.
- `go test ./...`: passes without a database, and on a fresh database with 0
  skipped tests. Store and API tests run on the Spark world:
  - Onboarding & profiles resolves to Priya.
  - Profile builder resolves to Priya via Onboarding & profiles.
  - Spark MVP resolves to Maya as project owner.
  - A non-member gets 404.
- `flutter analyze`: 8 infos (baseline). `flutter test`: 726 passed.
- Doc and redaction checks: pass.
- Live, as Maya:
  - Opening Onboarding & profiles from the Canvas shows Priya's view: Photo
    upload is near, and the area blocks App Store listing and the closed
    beta.
  - Opening Matching shows Leo's view, and "Leo's" lists Matching → Chat.
