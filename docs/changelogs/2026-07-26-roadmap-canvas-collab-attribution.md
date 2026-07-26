# 2026-07-26 — Roadmap canvas: B3 attribution (collab T6)

Workstream B3 of the [collaborative canvas plan](../plans/2026-07-24-roadmap-canvas-collab.md):
roadmap nodes now carry provenance — who created them and who last touched
them — surfaced as a neutral "last editor" initial on each card. Attribution
is provenance, not ownership (per the plan's D10); there is no assignee
model. All changes stay inside the `insideout` schema.

## What changed

**Schema** — [migration `20260726100000_roadmap_attribution.sql`](../../server/db/migrations/20260726100000_roadmap_attribution.sql):

- `roadmap_nodes` gains `created_by` and `updated_by`, both
  `uuid REFERENCES insideout.users(id) ON DELETE SET NULL`. Removing an
  author nulls the pointer rather than cascading away their nodes.
- Existing rows backfill NULL — pre-migration nodes report "unknown";
  no creator is fabricated.
- The `roadmap_nodes_insert` policy's `WITH CHECK` gains
  `created_by = insideout.current_user_id()` (plan D7), matching every
  sibling table so the RLS backstop rejects an app bug that writes a
  spoofed `created_by`. The membership `EXISTS` clause is preserved
  verbatim from the original policy.

**Store** — [roadmap.go](../../server/internal/store/roadmap.go),
[roadmap_plan.go](../../server/internal/store/roadmap_plan.go):

- `RoadmapNode` carries `CreatedBy`/`UpdatedBy` plus join-only
  `CreatorName`/`EditorName` (never stored on the row).
- Every mutation writes attribution: create/expand/replace set
  `created_by = updated_by = actor`; update/move set `updated_by = actor`
  only, leaving the original creator intact.
- `ListRoadmap` resolves display names via `LEFT JOIN insideout.users`
  (plan D6) for both columns, so a removed author yields NULL names —
  rendered "unknown" — instead of dropping the node. The node columns are
  alias-qualified to keep the join's shared `id`/`created_at` unambiguous.
- Single-node reads (`GetRoadmapNode`, mutation `RETURNING`) leave the
  names nil; the canvas re-lists after every mutation, so it always sees
  the joined names.

**API** — [api/roadmap.go](../../server/internal/api/roadmap.go): the
node view exposes `creatorName`/`editorName` (omitempty), populated from
the joined names.

**Frontend** — [RoadmapCanvasNode.vue](../../app/src/components/roadmap/RoadmapCanvasNode.vue):
each card shows a neutral roundel (`bg-surface-sunken text-fg-secondary`,
never vermilion — One Seal Rule) with the last editor's initial, or `?`
when unknown. `editorInitial` prefers `editorName`, falls back to
`creatorName`, then `?`. The tooltip reads "created by X · edited by Y"
(`roadmap.attribution`, both locales; `roadmap.unknownAuthor` = "unknown"/
"未知"). Exposed via `role="img"` + `aria-label` + `title`.

## How it was verified

No mocks; real PostgreSQL against `DATABASE_URL`.

- `go build ./... && go vet ./...` — clean.
- `go test ./...` — all packages ok.
- `go test ./internal/store/... -run TestRoadmap -v` — 6/6 pass, including
  the new `TestRoadmap_Attribution`, which covers: replace attributes
  creator+editor; an update by a second user moves `updated_by` while
  keeping `created_by`, and the list reflects both names; a move updates
  `updated_by`; an expand attributes children to the expander; a raw-SQL
  NULL-out of both columns (simulating `ON DELETE SET NULL`) still lists
  the node with nil names (D6); and a raw insert whose `created_by`
  mismatches the acting user is rejected by the tightened policy (D7).
- `pnpm test` — 44/44, including the EN/CN locale-parity check.
- `npx nuxi typecheck` — clean.

Migration applied with `go run ./cmd/insideout migrate`.

### Live verification — this repository as the subject (2026-07-26)

Both halves of the roadmap workstream were exercised end-to-end against the
running app, using `frankji-groundcontrol/insideout` itself as the test
subject, and reconciled against the real database (no mocks):

- **GitHub reading.** Linked this repo to a fresh project, then
  `POST /projects/{pid}/sync-github` returned `{"added":20}`; an immediate
  re-sync returned `{"added":0}` (the SHA cursor is idempotent — no
  duplicates). In PostgreSQL, 20 `[github …]` rows persisted as
  `kind='progress'` with `author_id` set; the newest is
  `[github eb1a012] feat(app): AI-generated baiwen seals … — frankdji`
  (this repo's HEAD), and `projects.meta->>'github_last_sha'` advanced to
  the full `eb1a012…` SHA.
- **Roadmap + B3 attribution.** Created a node, then PATCHed it to
  `in_progress`. The list endpoint returned `creatorName`/`editorName`
  (both `LiveTest`), and the row in PostgreSQL carries
  `created_by = updated_by`, both joining to the actor's username.
- **Roadmap card visual — light + dark (2026-07-26).** Logged into the
  running app as the test actor and opened the project page: the
  "Attribution probe" card renders the neutral last-editor roundel (`L`,
  `bg-surface-sunken text-fg-secondary`, never vermilion) with the
  accessible name/tooltip "由 LiveTest 创建 · 由 LiveTest 编辑"; the
  canvas toolbar (zoom / fit / review / open-canvas) and the full 20-row
  `[github …]` Activity timeline render correctly in both light and dark
  Ink & Seal themes. This closed the visual that the earlier pass could
  not reach. The blocker was **not** Node 25 — it was an IPv4/IPv6 bind
  split in the Nuxt dev server (see [HANDOFF](../HANDOFF.md) and the
  [learning note](../learning/2026-07-26-nuxt-dev-ipv6-426.md)): with no
  `HOST` set the app binds IPv6 `[::1]` and leaves the IPv4 `*:port`
  socket as an upgrade-only listener, so IPv4 clients saw a bare 426.
  Running the dev server with `HOST=127.0.0.1` binds one IPv4 socket
  that serves the app. EN-string parity for the tooltip is covered by
  the locale-parity test rather than a separate screenshot.

Note on tooling: raw `psql` against the same `DATABASE_URL` initially showed
zero rows. This is expected, not a bug — `FORCE ROW LEVEL SECURITY` hides
every row unless the session sets `app.user_id` (via
`set_config('app.user_id', …)`), which the store's `withUserContext` does on
every request. Setting it in the psql session made all of the above visible.

## Honest gaps

- **FK `ON DELETE SET NULL` is verified by the DDL and by the NULL-out
  test, not by a live user delete.** There is no user-deletion path under
  `FORCE ROW LEVEL SECURITY`, so the nulling was simulated with a raw
  `UPDATE` inside the actor's transaction.
- **Pre-migration rows render "unknown".** This is intended — no creator
  is backfilled or invented.

## Operator notes

Run the migration before deploying the new backend:
`go run ./cmd/insideout migrate`. The new columns are nullable and the
policy change only tightens inserts, so the migration is safe to apply
ahead of the code. No data backfill is performed or required.
