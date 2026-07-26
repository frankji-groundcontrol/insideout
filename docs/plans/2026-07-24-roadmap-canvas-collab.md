# 2026-07-24 — Roadmap canvas: collaborative model

Status: **complete.** All workstreams (A–D) and all tasks (T1–T10) have
landed. Workstream A (T1–T5), the frontend review-mode + edge re-semantics
lanes (T7–T8), the B3 attribution migration (T6), the full verification pass
(T10), and Workstream D (T9: glide transitions + sibling bands + minimap) are
all in. T6 + T10 are verified against the real DB (tests + typecheck) **and**
visually in a real browser, light + dark — GitHub reading shows all 20 synced
commits on the timeline and the attribution card shows the neutral last-editor
roundel + tooltip. T9 is live-verified light/dark × embedded/full with a clean
52/52 + typecheck, and an adversarial-verify pass confirmed its changes (one
surfaced popover-stacking defect was fixed in-line; two pre-existing low items
are tracked in the
[follow-ups issue](../issues/2026-07-26-roadmap-canvas-adversarial-followups.md)).
All decisions (D1–D11 across both eng passes) are resolved in § Decisions.
Follow-up to [2026-07-24 — Roadmap: tree on a canvas](2026-07-24-roadmap-canvas.md).
Owner directive: "read the layout, and think about the connections and user
behaviors hard about how user would use them collaboratively."

## Context

The canvas shipped as a polished **single-player editor on a shared URL**.
PRODUCT.md stakes the product on "collaborative team and workshop rhythms"
(board reviews, in-the-moment capture, refinement sessions) with two audiences —
a group lead tracking *others'* projects, and workshop/cohort members editing
together — under a flat invite-code role model. The backend confirms the flat
model: `requireMember` gates every roadmap op
([roadmap.go](../../server/internal/store/roadmap.go)), so any member can edit,
reparent, or cascade-delete any node, anonymously, last-write-wins. This plan
closes the gap in **one pass** (decision D1), ordered cheapest-correctness-first.

## Findings (from the analysis)

**Layout** ([tidyTree.ts](../../app/src/utils/tidyTree.ts)): rightward implies
"later," but children are decompositions — depth reads as progress; the seals
carry the real state. The root band grows tall and left-heavy; `fitTo` zooms the
embedded shell to an unreadable thumbnail (embedded = map, full = territory).
"Parallel tracks" have no visual grouping. Blank 留白 ground is unanchored.

**Connections** ([RoadmapCanvas.vue](../../app/src/components/roadmap/RoadmapCanvas.vue)):
edge tint = child status duplicates the node's seal and spends vermilion (erodes
the One Seal Rule). Edges encode status (redundant) and none of the collaborative
signal the lead needs. During drag the edge shows the *past*, never the
prospective edge. Edges are inert and cross on deep trees.

**Collaborative behavior**: board review has no read mode (editor chrome is a
misclick hazard); live sessions diverge (no refresh) and hit the lost-update bug
below; cascade delete + tree-replace have no undo; "whose branch is this?" is
unanswerable; concurrent edits teleport cards.

### The lost-update bug (top correctness finding)

`UpdateRoadmapNode` is a blind full-field write (`SET title, description,
status`), and `cycleStatus()` re-sends the title/description it holds *in memory*
with the new status ([RoadmapCanvasNode.vue:42](../../app/src/components/roadmap/RoadmapCanvasNode.vue)).
A edits a title while B clicks the status seal; B's PATCH overwrites A's title
with B's stale copy. `saveEdit` does the symmetric thing to `status`. The most
common collaborative action — advancing a status — is the one that silently
clobbers a teammate's edit.

## Decisions (all resolved)

- **D1 — single pass, ordered workstreams.** Land A → B → C → D together, not
  dribbled out. "One pass" ≠ one commit: Workstream A (correctness) lands as the
  first commit, then B/C, then D.
- **D2 — status moves off edges.** Edges → neutral hairlines; node seals own
  status; freshness gets an emphasis instead (C2).
- **D3 — attribution in scope now.** B3's migration lands this pass.
- **D4 (re-review) — A5 rollback test uses a store fault-injection hook.** The
  atomic-expansion store method exposes a package-level test seam to fail the
  Nth child insert, so the real-DB test forces a mid-loop failure and asserts
  zero children persist. `roadmap_nodes` has no title-length CHECK (length is
  handler-enforced only), so a deterministic constraint violation isn't
  available — the hook is the no-mocks mechanism. Test-only surface, not used
  by prod paths.
- **D5 (re-review) — cycle-guard CTE `UNION ALL` → `UNION` (P1).** The
  move/reparent cycle check's recursive CTE at
  [roadmap.go:209](../../server/internal/store/roadmap.go) walks a node's
  descendants with `UNION ALL`; on a pre-existing cycle (corrupt data) it
  generates rows forever and the move hangs. `UNION` (de-dup) terminates. Folded
  into Workstream A as A6.
- **D6 (re-review) — B3 creator join is `LEFT JOIN` + unknown fallback.** A
  creator removed from the workspace must not vanish their nodes from the
  response; the users join is `LEFT JOIN`, and a NULL/absent profile renders an
  "unknown" initial rather than dropping the row.
- **D7 (re-review) — B3 tightens `roadmap_nodes_insert`.** Add `created_by =
  insideout.current_user_id()` to the insert policy's `WITH CHECK`, matching
  every sibling table (projects/ideas/prds/project_updates), so the RLS backstop
  catches an app bug that writes a spoofed `created_by`.
- **D8 (re-review) — A4 count is threaded and the guard is atomic.** Thread the
  expected count through `IPrdService.build(id, expectedCount)` (a real POST
  body, not the empty body it sends today) into `handleBuildFromPrd`. Inside
  `ReplaceRoadmapTree`'s existing transaction, take a per-project advisory lock
  (`pg_advisory_xact_lock(hashtext(project_id::text))`) before the count check +
  DELETE — serializing concurrent builds and closing both the check-then-delete
  TOCTOU and the two-empty-builds double-tree edge in one line.
- **D9 (re-review) — frontend PATCH body is sparse.** `IRoadmapService.update`
  serializes only the keys being changed (absent keys omitted, not
  empty-filled) — an empty-fill would either trip the handler's validation or
  clear a field the caller meant to keep.
- **D10 (re-review) — the visible initial is the last editor.** The card's
  attribution initial is `updated_by` (who touched it most recently — who you'd
  ask about current state); a tooltip shows "created by X · edited by Y".
  Pre-migration NULLs render "unknown" (per D6). `created_by` alone never backs
  the visible mark.
- **D11 (re-review) — `?review=1` is a view-state deep-link, not access
  control.** It seeds review mode on mount (B1 already specifies this); under
  the flat invite model every recipient is still an editor who can toggle off,
  so it must not be read as a permissions boundary. No new build — one
  clarifying line.
- **Partial update via COALESCE, not optimistic concurrency (yet).** Every
  `update` field optional, `col = COALESCE($n, col)`, so a status cycle sends
  only `{status}`. Same-field concurrent edits (title vs title) still
  last-write-wins — accepted debt, see NOT in scope.
- **Focus-refresh only, no poll** (review Finding 1). Refresh on
  `visibilitychange`→visible / window `focus`, as a **silent** refresh (skeleton
  only on first mount) with a latest-request-wins guard. Applied to the embedded
  canvas too, not just the full route.
- **Tree-replace is guarded** (outside-voice P1). `ReplaceRoadmapTree` refuses a
  non-empty roadmap unless the request carries the live node count; the frontend
  confirms "this replaces N nodes" with that count.
- **Freshness uses two distinct signals** (not redundant): the card shows the
  exact "updated X ago"; the edge emphasis is a branch-level "is this hot" scan
  aid for a lead. Different granularities, deliberately.
- **Attribution = provenance, not ownership.** `created_by`/`updated_by` tell
  who created / last touched a node — a proxy for "whose branch." A true
  assignee/owner model is out of scope.
- **Reuse, don't build.** Freshness rides `updated_at`; confirms reuse
  `window.confirm`; the one new build is a minimal `useToast` (none exists).

## Workstream A — correctness floor (lands first)

**A1. Partial PATCH.**
- `server/internal/api/roadmap.go`: `updateRoadmapNodeRequest` fields →
  `*string`; validate title/status only when present; require ≥1 field (400
  otherwise).
- `server/internal/store/roadmap.go`: `RoadmapNodeFields` → pointer fields;
  `UPDATE ... SET title=COALESCE($2,title), description=COALESCE($3,description),
  status=COALESCE($4,status)`. `''` is a real value; only NULL means "don't
  touch"; title/status are never legitimately NULL; `description:""` still clears.
  **Caller rule (D1 re-review):** never pass a *present-but-empty* pointer when
  you mean "don't touch" — omit the field (leave it nil) instead. The data-loss
  case is the inverse of the clear case: a status-only PATCH must leave a
  populated `description` intact, and that's covered by a dedicated store test.
- `app/src/types/services.ts` ([IRoadmapService.update:109](../../app/src/types/services.ts)):
  payload → `Partial<{ title; description; status }>`. **Sparse body (D9
  re-review):** the service serializes only the keys being changed — absent keys
  are omitted from the JSON, not empty-filled (an empty-fill would either trip
  the ≥1-field validation or clear a field the caller meant to keep).
- `RoadmapCanvasNode.vue`: `cycleStatus` sends `{ status }` only; `saveEdit`
  sends `{ title, description }` only.

**A2. Focus-refresh (silent, latest-wins, embedded too).**
- `RoadmapCanvas.vue`: split `load()` (initial, shows skeleton) from a silent
  `refresh()` (updates `nodes`, never touches `loading`). A monotonically
  increasing request id; only the latest response is applied.
- Listeners: `visibilitychange`→visible + window `focus`, on **both** the
  embedded map and the full route. No 20s poll.
- Refit stays guarded by `interacted`. On refresh, if the open-edit node's id no
  longer exists, close its popover.

**A3. Failure feedback + safer delete.**
- NEW minimal `useToast` (a ref array + one fixed outlet). Surface failures from
  **every** mutation — status, edit, create, add-child, move, delete, load,
  expand — not just move/delete. Note: `cycleStatus`/`addChild`/`saveEdit`/
  `removeNode` currently have **no** `catch` at all (rejections go unhandled);
  `expandWithAI`/`onDragUp` swallow with a bare `catch{}` — wire all of them.
- **Toast spec (design D5).** Surface = the card's own popover recipe
  (`bg-surface-raised border-stroke-subtle shadow-popover rounded-card`); error
  glyph + text in `text-fg-danger` only — **never** vermilion (One Seal Rule).
  Fixed **bottom-center** outlet (clears the top-right toolbar and the bottom
  card marks). `role="alert"` / `aria-live="assertive"`. 5s auto-dismiss + a
  44px close button; max 3 stacked, newest on top. i18n: add `roadmap.toast.*`
  keys in both `en-US`/`zh-CN`. Silent `refresh()` failures stay quiet (retry on
  next focus); only the initial `load()` failure toasts.
- Delete confirm: client-computed descendant count ("this deletes N nodes") via
  `window.confirm`. Advisory; the deeper issue (no undo) is NOT in scope.

**A4. Guard tree-replace** (outside-voice P1).
- `ReplaceRoadmapTree` takes the expected live node count; if the roadmap is
  non-empty and the count doesn't match → `ErrConflict` (409) returning the
  current count. `handleBuildFromPrd` surfaces it.
- **Atomic + threaded (D8 re-review):** the count check runs inside the existing
  transaction behind a per-project advisory lock
  (`pg_advisory_xact_lock(hashtext(project_id::text))`) taken first — serializing
  concurrent builds and closing both the check-then-delete TOCTOU and the
  two-empty-builds double-tree edge. The count is threaded end-to-end:
  `IPrdService.build(id, expectedCount)` sends `{expectedCount}` in the POST body
  (it sends an empty body today), through `handleBuildFromPrd`, into the store.
- Frontend "build from PRD": fetch the count, `window.confirm("this replaces N
  nodes")`, send the count. Closes accidental wipes and stale-tab clobbering.

**A5. Atomic AI expansion** (outside-voice P2).
- Replace the per-child `CreateRoadmapNode` loop in `handleExpandNode`
  ([roadmap_ai.go:119](../../server/internal/api/roadmap_ai.go)) with one store
  method that inserts all children in a single transaction — a mid-loop failure
  rolls back the whole expansion.

**A6. Cycle-guard termination** (re-review P1, D5).
- The move/reparent cycle check's recursive CTE at
  [roadmap.go:209](../../server/internal/store/roadmap.go) uses `UNION ALL`; a
  pre-existing cycle (corrupt data) makes it generate rows forever and the move
  hangs. Change to `UNION` (de-dup) so the traversal terminates. One-word fix;
  covered by a move-under-existing-cycle test.

## Workstream B — collaborative visibility (lead's rhythm)

- **B1. Review/present mode — truly read-only (design D6).** A toggle in the
  top-right toolbar cluster on **both** placements (embedded + full route), with
  `aria-pressed` and a persistent quiet chip **"Reviewing · read-only"**
  (`bg-surface-sunken text-fg-secondary` + EyeIcon — never vermilion, One Seal
  Rule). Enforcement is real, not cosmetic: mutation controls render with `v-if`
  (out of the tab order) **and** interactive elements get `:disabled` — NOT
  `pointer-events:none` alone, which leaves the seal/delete buttons reachable by
  Tab+Enter. Touch targets bumped toward 44px. Manual (flat authz), not
  role-derived. i18n: add `roadmap.reviewing` + the toggle label in both locales.
  **Every mutation surface, enumerated (D2 re-review):** the status seal, the
  hover edit/delete/AI-expand buttons, the edit + add-child popovers, **and the
  add-goal form/input** all render `v-if`-off / `:disabled`. **Drag-to-reparent**
  is not a button — it lives on the card wrapper's `@pointerdown`
  ([RoadmapCanvas.vue:216](../../app/src/components/roadmap/RoadmapCanvas.vue)),
  so `onCardPointerDown` returns early when reviewing. Zoom / pan / fit /
  open-full stay enabled (read operations, not mutations).
  **Entry point (design D7):** honor `?review=1` on mount to seed read-only, so a
  lead can share a view-only link. The in-session toggle still flips it; the URL
  is NOT rewritten on toggle — the param is an entry point, not live-synced state.
  **View-state, not access control (D11 re-review):** under the flat invite model
  every recipient is still an editor who can toggle off, so `?review=1` is a
  deep-link into a read-only *view*, never a permissions boundary.
- **B2. Freshness on the card.** `updated_at` → "updated X ago" + a "quiet N
  days" stale tint. **Must be i18n-aware** — the existing
  [useTimeAgo.ts](../../app/src/composables/useTimeAgo.ts) is hard-coded English;
  localize (today/yesterday/Nd/Nmo/Ny) in `en-US`/`zh-CN` to pass the 中文 check.
  No schema change.
- **B3. Attribution (migration).** `ALTER TABLE insideout.roadmap_nodes ADD
  created_by uuid, updated_by uuid` → `insideout.users(id) ON DELETE SET NULL`;
  existing rows backfill NULL (unknown creator — don't fabricate). Populate
  `updated_by` on **every** mutation (create/update/move/expand/replace);
  `created_by` on create/expand/replace. Add to the response view + a users join
  for the display initial. `insideout` schema only.
  - **Join is `LEFT JOIN` + unknown fallback (D6 re-review):** a creator later
    removed from the workspace must not vanish their nodes from the response —
    the users join is `LEFT JOIN`, and a NULL profile renders an "unknown"
    initial rather than dropping the row.
  - **Insert policy tightened (D7 re-review):** `roadmap_nodes_insert`'s `WITH
    CHECK` gains `created_by = insideout.current_user_id()`, matching every
    sibling table, so the RLS backstop catches an app bug writing a spoofed
    `created_by`.
  - **Visible initial = last editor (D10 re-review):** the card shows the
    `updated_by` initial (who to ask about current state); a tooltip shows
    "created by X · edited by Y". Pre-migration NULLs render "unknown".

## Workstream C — connection re-semantics

- **C1. Prospective edge during drag (design D8).** A live bezier from the
  **drop target (parent) to the drag ghost (child)** — matching tree edge
  direction — so the gesture previews the structure it creates. **Coordinate
  fix (correctness):** the ghost tracks the cursor in viewport px
  ([RoadmapCanvas.vue:124](../../app/src/components/roadmap/RoadmapCanvas.vue))
  but edges render inside the world/transformed SVG (`:184`); convert the cursor
  via the existing `toWorld`
  ([usePanZoom.ts](../../app/src/composables/usePanZoom.ts)) and draw the
  prospective path inside that world SVG. Style: `stroke-subtle`,
  `stroke-dasharray="4 4"`; render **nothing** when `dropTargetId` is null
  (move-to-root). Drops still send `position 0` (append); sibling-order control
  is a v1 limit (NOT in scope).
- **C2. Edges → neutral hairlines (design D8).** Replace all 4 status tints with
  a single neutral `stroke-subtle` at **1.5px** (constant width) — node seals own
  status (restores One Seal rarity). **Hot-branch emphasis:** a branch is "hot"
  when any descendant's `updated_at` is within **7 days** (one tree walk); encode
  it as a two-step neutral `stroke-subtle → stroke-strong`. **Dark-mode rule (D3
  re-review):** the token swap alone is a perceptual near-no-op in dark
  (`--color-stroke-subtle` 46 52 46 → `--color-stroke-strong` 58 66 58, Δ≈13 per
  channel on a near-black ground), so in **dark** a hot edge is `stroke-strong`
  **and** bumps to **2.5px** width; in **light** the token swap at constant 1.5px
  already reads. Same neutral tokens throughout — no new color, no accent,
  vermilion stays reserved. The card's exact "updated X ago" timestamp is the
  redundant fallback. Distinct from the card's exact timestamp (see Decisions).

## Workstream D — layout & orientation (largest; last)

- Animated layout transitions on data change (cards glide, don't teleport) —
  **this** is where `useReducedMotion` applies (moved out of A2).
- Visual grouping for parallel tracks (a band per sibling set).
- Minimap / orientation aid on the full route.

## Test plan

Real-DB integration in [roadmap_test.go](../../server/internal/store/roadmap_test.go)
(+ handler tests); frontend unit in `app/src/utils/__tests__`. No mocks.

```
CODE PATHS (A1 partial-PATCH)                     USER FLOWS
handleUpdateRoadmapNode                           [→E2E] two tabs converge on focus;
 ├─ status-only → title+desc intact [REGRESSION]    status cycle doesn't clobber a
 ├─ title-only → status+desc intact [REGRESSION]    concurrent title (live verify)
 ├─ status-only w/ populated desc                  [→E2E] build-from-PRD on a non-empty
 │   → desc preserved (D1)          [GAP→test]     roadmap → 409 until confirmed with
 ├─ desc-only / desc:"" clears      [GAP→test]     the live count (live verify)
 ├─ all-three → all updated         [GAP→test]   [→E2E] review mode: status/drag/add
 ├─ title empty → 400               [GAP→test]     -goal/seal/popovers all inert
 ├─ status invalid → 400            [GAP→test]     (live verify, D2 surfaces)
 └─ zero fields → 400 (≥1 field)    [GAP→test]
A4: replace non-empty w/o count → 409; w/ count → ok
    (D8: count check is in-tx behind a per-project advisory lock; two
    concurrent builds serialize — the loser re-reads and 409s, no double tree)
A5: expansion failing on child 3/5 → rolls back all
    (D4: store fault-injection hook fails the Nth insert; real DB, no mocks)
A6: move a node under an existing cycle → completes (UNION terminates),
    no hang (D5)
Frontend: cycleStatus sends {status}; saveEdit sends {title,desc};
delete-count helper (reuses tree walk); silent refresh leaves an open
draft intact and never flashes the skeleton; review mode guards
onCardPointerDown (no drag) + add-goal form (D2)
```

## Failure modes

- Partial-PATCH write fails → toast; single-row COALESCE means no half-written
  node. Test + toast cover it.
- Focus-refresh race (older response lands last) → latest-wins guard. Covered.
- Tree-replace with a stale count → 409; client re-confirms. Test covers it.
- Node deleted while open in a popover → popover closes on refresh; a stale save
  → 404 → toast. Covered.
- AI expansion partial failure → atomic (A5); rollback test covers it.
- **Concurrent A↔B move creates a cycle** (both cycle-checks pass pre-commit) —
  KNOWN GAP, no test/ handling yet → deferred, see NOT in scope.

## NOT in scope (deferred)

- **Optimistic concurrency** (`updated_at` → 409) for same-field concurrent
  edits. The real fix for the residual lost-update; COALESCE only removes the
  cross-field case. Hardening.
- **Concurrent-move cycle prevention** (row lock / serializable) — the A↔B race.
  Low-probability; hardening.
- **True ownership/assignee model** — B3 is provenance only.
- **Real-time presence / cursors** (websocket collab) — focus-refresh suffices.
- **Sibling-order control on drop** (drops append at position 0).
- **Undo** for delete / tree-replace.

## What already exists (reuse map)

- `updated_at` + `set_updated_at` trigger → B2 freshness (no schema change).
- [useTimeAgo.ts](../../app/src/composables/useTimeAgo.ts) → base for B2 (needs i18n).
- `window.confirm` → delete + tree-replace confirms (existing pattern).
- `isDescendant` / tidyTree walk → delete-count helper + C1.
- `meta jsonb` → stays the extension point for non-relational extras.
- [roadmap_test.go](../../server/internal/store/roadmap_test.go) → home for A1/A4/A5.
- `useReducedMotion` → Phase D.
- Migration pattern (`20260722120000_roadmap_nodes.sql`) → B3.

## Parallelization

| Workstream | Modules touched | Depends on |
|------------|-----------------|-----------|
| A-backend (A1, A4, A5, A6, B3-backend) | `server/internal/store`, `server/internal/api`, `server/db/migrations` | — |
| A/B/C-frontend (A2, A3, B1, B2, C1, C2) | `app/src/components/roadmap`, `app/src/composables`, `app/src/i18n` | A1 API contract |
| D (transitions, grouping, minimap) | `app/src/components/roadmap`, `app/src/utils` | C |

Lane A (backend) and Lane B (frontend) run in parallel once the A1 JSON contract
is agreed (it's the same shape, now partial). **Conflict flag:** both lanes touch
the `IRoadmapService` contract in `app/src/types/services.ts` — coordinate. B3
display depends on B3 backend. D is sequential after C.

## Implementation tasks

- [x] **T1 (P1, human ~2h / CC ~20min)** — backend — A1 partial PATCH (COALESCE) + handler validation
- [x] **T2 (P1, human ~1h / CC ~15min)** — backend — A4 tree-replace count-precondition (409, advisory-locked, count threaded from `IPrdService.build`) + A5 atomic expansion + A6 cycle-guard `UNION` fix
- [x] **T3 (P1, human ~1h / CC ~15min)** — backend — A1/A4/A5/A6 integration tests (real DB)
- [x] **T4 (P1, human ~1h / CC ~15min)** — frontend — A1 partial payloads (sparse body) + A3 useToast + safer delete *(the contract-critical sparse payloads + build-count threading shipped and are pinned by the D9 test; A3's stacked toast + the delete descendant-count confirm shipped instead as an in-canvas error banner + static confirm — deferred, see [issue](../issues/2026-07-25-canvas-failure-feedback-is-a-banner-not-the-specced-toast.md))*
- [x] **T5 (P2, human ~1h / CC ~15min)** — frontend — A2 silent focus-refresh + latest-wins + orphan close
- [x] **T6 (P2, human ~2h / CC ~20min)** — backend+frontend — B3 attribution migration + LEFT JOIN + tightened insert policy + card initial (last editor) *(tests + typecheck + live DB reconciliation; card visual verified light + dark in T10 — [changelog](../changelogs/2026-07-26-roadmap-canvas-collab-attribution.md))*
- [x] **T7 (P2, human ~1h / CC ~15min)** — frontend — B1 truly-read-only review mode + B2 i18n freshness *(live-verified light/dark × 中文/EN; [changelog](../changelogs/2026-07-26-roadmap-canvas-collab-review-and-edges.md))*
- [x] **T8 (P2, human ~1h / CC ~15min)** — frontend — C1 prospective edge + C2 neutral edges + freshness emphasis *(live-verified; C1 prospective edge + neutral edge by code/compile only — [changelog](../changelogs/2026-07-26-roadmap-canvas-collab-review-and-edges.md))*
- [x] **T9 (P3, human ~3h / CC ~30min)** — frontend — D transitions + track grouping + minimap *(done 2026-07-26: keyed-wrapper glide + reduced-motion-gated transition, neutral sibling bands, full-route minimap, plus a load-bearing routing fix and a one-line popover-stacking fix; live-verified light/dark × embedded/full, 52/52 + typecheck — [changelog](../changelogs/2026-07-26-roadmap-canvas-workstream-d.md), [routing learning note](../learning/2026-07-26-nuxt-dynamic-route-parent-shadowing.md))*
- [x] **T10 (P1, human ~1h / CC ~15min)** — verify — full Verification pass below *(done 2026-07-26: backend `go build/vet/test` green, `pnpm test` 44/44 + typecheck clean; GitHub reading + roadmap + B3 attribution reconciled against the real DB **and** visually confirmed light + dark in a real browser — 20 synced commits on the timeline, neutral roundel + "由 X 创建 · 由 Y 编辑" tooltip on the card. The earlier browser blocker was an IPv4/IPv6 dev-server bind split, not Node 25 — see [changelog](../changelogs/2026-07-26-roadmap-canvas-collab-attribution.md) + [learning note](../learning/2026-07-26-nuxt-dev-ipv6-426.md))*

## Verification

- `cd server && go build ./... && go vet ./... && go test ./...`; the
  `DATABASE_URL`-gated roadmap integration tests (A1/A4/A5) pass against real
  PostgreSQL. No mocks.
- `cd app && pnpm test && npx nuxi typecheck && pnpm build`.
- Live (real backend + DB, random high ports, proxy in sync): two sessions on
  the same tree converge on focus; a status cycle no longer clobbers a title
  edited in the other session; build-from-PRD on a non-empty roadmap 409s until
  confirmed with the live count; review mode makes status/drag/add inert; delete
  confirms with the right count; light + dark; EN + 中文.
- Update [docs/plans/README.md](README.md) index + changelog on completion.

## Checklist

- [x] Plan filed (this doc)
- [x] plan-eng-review pass; findings resolved
- [x] Owner decisions D1/D2/D3 + review finding + outside-voice points recorded
- [x] Workstream A (partial PATCH + focus-refresh + feedback/delete + tree-replace guard + atomic expand) *(A3 landed as an in-canvas banner, not the specced toast — [issue](../issues/2026-07-25-canvas-failure-feedback-is-a-banner-not-the-specced-toast.md))*
- [x] Workstream B (review mode + freshness + attribution) *(B1+B2 done/verified; B3 attribution landed + live-verified against the real DB — [changelog](../changelogs/2026-07-26-roadmap-canvas-collab-attribution.md); card visual verified light + dark in T10)*
- [x] Workstream C (prospective edge + neutral edges + freshness emphasis)
- [x] Workstream D (transitions + grouping + minimap) *(glide + sibling bands + minimap; [changelog](../changelogs/2026-07-26-roadmap-canvas-workstream-d.md))*
- [x] Verification above; changelog + index updates *(verification completed in T10; changelog + HANDOFF + this index reconciled 2026-07-26)*

## GSTACK REVIEW REPORT

| Review | Trigger | Why | Runs | Status | Findings |
|--------|---------|-----|------|--------|----------|
| CEO Review | `/plan-ceo-review` | Scope & strategy | 0 | — | — |
| Codex Review | outside voice | Independent 2nd opinion | 2 | issues_found | pass 1 (codex): 16 raised, 3 confirmed (1 P1); pass 2 (subagent): 1 P1 + 5 P2 + 1 ponytail note, all dispositioned |
| Eng Review | `/plan-eng-review` | Architecture & tests (required) | 3 | clean | pass 1: 22 issues, 1 critical gap (tree-replace, guarded); pass 2 (re-review): 10 issues, 0 critical gaps → D5–D11 folded |
| Design Review | `/plan-design-review` | UI/UX gaps | 1 | clean | score 5/10 → 9/10; 14 findings confirmed + dispositioned into 4 decisions (D5–D8); 2 TODOs deferred |
| DX Review | `/plan-devex-review` | Developer experience gaps | 0 | — | — |

- **CODEX (pass 1):** confirmed 3 missed defects — unguarded `ReplaceRoadmapTree`
  (P1), English-only freshness formatter, non-atomic AI expand — folded into A4,
  B2, A5; the rest dispositioned as folds, accepted debt, or NOT-in-scope.
- **OUTSIDE VOICE (pass 2, Claude subagent):** 1 P1 (cycle-guard `UNION ALL` →
  `UNION`, [roadmap.go:209](../../server/internal/store/roadmap.go)) + 5 P2 (B3
  LEFT JOIN, B3 insert-policy `created_by` CHECK, A4 count-threading + advisory
  lock, sparse-JSON body, visible-initial FK) + 1 ponytail challenge (`?review=1`
  reframed as view-state, not access control) — all folded as D5–D11.
- **CROSS-MODEL:** one tension — freshness on both cards and edges (codex:
  redundant). Resolved by the owner: keep both as distinct granularities (card =
  exact time, edge = branch scan aid).
- **DESIGN (single-model):** the design outside voice ran Claude-subagent-only
  (codex timed out); its 14 findings were each verified against source
  (file:line) and folded into A3 (toast), B1 (review mode + `?review=1`), and
  C1/C2 (edge re-semantics). Two a11y/destructive-op debts deferred to
  [docs/TODO.md](../TODO.md) (keyboard-only operation; undo).
- **VERDICT:** ENG + DESIGN CLEARED (both clean @ eb1a012) — ready to implement.

NO UNRESOLVED DECISIONS
