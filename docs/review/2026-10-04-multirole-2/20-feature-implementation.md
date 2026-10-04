# 20 · Feature Implementation

**Lens:** 
**Score:** **7.5 / 10**
**Bottom line:** Every README-advertised feature is verifiably implemented end-to-end (50 MCP tools verified exactly, deadline picker exists, admin UI exists — three suspected gaps refuted by code). The gaps live in PRODUCT.md's target: Product Branches/Merge absent, Decision Log an orphaned string field with no web input or display, agent proposals have no web decision loop, node evidence and load-more unreadable in web, lenses/personas/adaptive-coach docs-only.

Reviewed main @ 26db979; a skeptic re-checked every High finding from this review. [Correction] FEAT-01 and FEAT-02 were downgraded High→Medium — the gaps are real (verified by grep) but PRODUCT.md explicitly frames Branches/Merge and the Decision Log as end-state loop steps, not current promises. FEAT-03 was confirmed High: proposals-until-accepted is documented, the decision route and MCP tool exist, and the web has zero proposal handling.

---

This review was persisted findings-only; the section below is assembled verbatim from its structured findings (the prose wrapper was not retained). 12 findings as filed, worst first.

## Findings

### FEAT-01 · High ·  **Skeptic: DOWNGRADED → Medium.**

**Product Branches and Merge — a core PRODUCT.md mechanism — are entirely absent (no table, no route, no UI).**

- Where: `PRODUCT.md:209`
- Evidence: PRODUCT.md:209-213 'Product Branch / Explore direction… Merge / Combine conclusion'; zero matches for productbranch/product_branch across server/ and client/.
- Fix: Smallest slice: a prd_branches table (source_prd_commit_id, title, driver) + POST /prds/{id}/branch + branch badge on versions_page. First check: store/prd_commits.go for the commit shape to fork from.
- Diff size: Large

### FEAT-02 · High ·  **Skeptic: DOWNGRADED → Medium.**

**Decision Log is one orphaned string field: the web Commit dialog collects name/audience/summary only — no decisionNote input, and no surface anywhere renders it; overrides/reversals/Merges are not logged at all.**

- Where: `client/lib/features/prd/versions_commit_dialog.dart:57`
- Evidence: PRODUCT.md:220-222 'A Decision Log accompanies meaningful Commits…'; server accepts it (prd_commits.go:31 DecisionNote) and CLI/MCP pass it (cli_version.go:125, commit tool note), but versions_commit_dialog.dart:57-70 has three fields; grep decisionNote in client/lib/features: nothing.
- Fix: Add a decisionNote TextField to VersionsCommitDialog, pass through commitPrd; render decisionNote in versions_stack.dart commit detail.
- Diff size: Small-Medium

### FEAT-03 · High ·  **Skeptic: CONFIRMED.**

**Agent proposals have no web decision loop: agent_proposal updates render as generic editable/deletable notes, and the accept/reject/apply endpoint is reachable only via CLI/MCP.**

- Where: `client/lib/features/project/project_page.dart:125`
- Evidence: PRODUCT.md:243-245 'Contributions… remain proposals until accepted'; route exists (agent_routes.go:21 POST /agent/proposals/{uid}/decision) and MCP proposal_decide exists, but client/lib has zero 'proposal' matches; project_page.dart:125-145 renders every update with the same edit/delete menu.
- Fix: Special-case u.kind == 'agent_proposal' with Accept/Reject buttons calling the decision endpoint; hide edit/delete for that kind.
- Diff size: Medium

### FEAT-04 · Medium · 

**Roadmap node evidence is unreadable in the web: GET /roadmap/{nid}/evidence exists but the client has no method or UI for it.**

- Where: `client/lib/api/client.dart:1`
- Evidence: Route registered (roadmap.go:22) with CLI 'evidence' verb and MCP roadmap_evidence; grep nodeEvidence/evidence in client.dart: nothing; project_page 'Evidence' heading is marketing copy over the updates list.
- Fix: Add client api.nodeEvidence(id) + an evidence section in the node detail dialog.
- Diff size: Small-Medium

### FEAT-05 · Medium · 

**Project update history beyond the first page is unreachable in the web — nextCursor is emitted but never consumed.**

- Where: `server/internal/api/projects.go:167`
- Evidence: projects.go:166-167 'surfaced as nextCursor for a future load-more (frontend follow-up)'; zero nextCursor/loadMore matches in client/lib.
- Fix: Thread nextCursor through the project model and add a Load more button in project_page.dart appending GET with the cursor.
- Diff size: Small

### FEAT-06 · Medium · 

**Per-project product authority roles (Decision Owner / Driver / Contributor / Reviewer / Viewer) are docs-only; only workspace owner/admin/member exists.**

- Where: `PRODUCT.md:231`
- Evidence: PRODUCT.md:231-240 five roles 'assigned per Project' vs memberships.go workspace roles only; commit auth is author-or-admin (prd_commits.go:50).
- Fix: Direction: project-level role column gating commit/decision endpoints. First check: memberships.go and the prior role migration pattern.
- Diff size: Medium-Large

### FEAT-07 · Medium · 

**The adaptive coaching contract (complexity-adaptive depth, question priority/reader/why metadata, no forced sequence) is docs-only; the running coach is the rigid four-stage flow PRODUCT.md explicitly supersedes.**

- Where: `PRODUCT.md:88`
- Evidence: PRODUCT.md:89-98 'adapts interview depth… does not force a fixed number of questions' + priority tags vs prompts.go:14-17 fixed stages and coach_panel.dart:103 canned per-stage suggestions.
- Fix: Direction: attach priority/reader/why fields to coach questions in prompts.go and allow stage skipping. First check: agent/responses.go for where question metadata could ride.
- Diff size: Large

### FEAT-08 · Medium · 

**Roadmap Lenses (Decision/Product/Delivery/Validation projections) are docs-only; the roadmap ships progress/bands/canvas plus agent-context modes.**

- Where: `PRODUCT.md:277`
- Evidence: PRODUCT.md:279-285 four Lenses vs roadmap_page.dart:28 'enum _RoadmapView { progress, bands, canvas }'.
- Fix: Direction: reuse the progress endpoint with a lens query param, different field emphasis. First check: roadmap_progress.go for already-computable lens fields.
- Diff size: Large

### FEAT-09 · Medium · 

**Overdue action model is half-built: pressure states render and the deadline is editable, but there is no reason recording, no transfer, no explicit abandon.**

- Where: `PRODUCT.md:297`
- Evidence: PRODUCT.md:297-301 'Overdue work is never silently closed… records the reason' vs roadmap.go:17 PATCH with no reason field; deadline editable in two clicks.
- Fix: Direction: require a reason string when PATCH changes a deadline on an overdue node (or status→done). First check: roadmap update handler.
- Diff size: Medium

### FEAT-10 · Low · 

**Persona model with grounded/synthesized/assumed labeling and per-persona arrival/do/value/leave records is docs-only; PRDs have a fixed free-text 'users' section.**

- Where: `PRODUCT.md:123`
- Evidence: PRODUCT.md:129-131 provenance labels vs fixed 8 sections (product_tools.go:268), no persona entity.
- Fix: Direction: structured personas with a provenance enum. First check: record_fact flow for where personas already surface.
- Diff size: Large

### FEAT-11 · Low · 

**Scoped feedback invitations for external advisers (no workspace join) and feedback-as-suggestion flow are absent.**

- Where: `PRODUCT.md:255`
- Evidence: PRODUCT.md:255-259 'scoped feedback invitation without joining' — zero feedback matches in server/internal/api and store.
- Fix: Direction: a scoped read-only feedback token on the audience view. First check: invitations.go TTL/token pattern.
- Diff size: Large

### FEAT-12 · Low · 

**'Start from an existing GitHub repository' is partial: manual project create + repo link + sync exist, but no baseline import, and unmatched activity has no review queue.**

- Where: `PRODUCT.md:74`
- Evidence: PRODUCT.md:74-75 'connect work that has already started' + :323-325 'Unmatched activity stays visibly unassociated for human review' vs unmatched events dropped by guide.Match.
- Fix: Direction: a 'seed from repo' flow listing matched activity for triage. First check: guide.Match for the reverse mapping.
- Diff size: Medium-Large

