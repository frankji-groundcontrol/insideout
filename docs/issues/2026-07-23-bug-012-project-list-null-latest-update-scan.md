# BUG-012: workspace board 500 — scanning a NULL `latest_update` into value types

**Found**: 2026-07-22, during the frontend pages verification sweep — the workspace board (`/workspaces/{id}`) returned HTTP 500 whenever it contained a project with **no updates yet**.

**Symptom**: `GET /api/v1/workspaces/{id}/projects` failed with a 500 and the board never rendered; projects that already had at least one update loaded fine, so the failure only appeared on freshly created / untouched projects.

**Root cause**: `internal/store/projects.go`'s `ListProjectsForWorkspace` annotates each project with its most recent update via a `LEFT JOIN LATERAL (... LIMIT 1) lu`. When a project has zero rows in `project_updates`, the lateral join still emits one row, but the `lu.*` columns (`kind`, `content`, `created_at`) are all `NULL`. The scan originally targeted plain value fields:

```go
LatestUpdateKind    string
LatestUpdateContent string
LatestUpdateAt      time.Time
```

`database/sql` cannot scan a `NULL` into a non-pointer `string`/`time.Time` — it returns `sql: Scan error on column index N: converting NULL to string is unsupported`, the query aborts, and the handler surfaces a 500. The COALESCE in the `ORDER BY` only guarded *ordering*, not the *selected* columns.

**Fix**: made the three "latest update" fields nullable pointers so a project with no updates scans cleanly as `nil`:

```go
LatestUpdateKind    *string
LatestUpdateContent *string
LatestUpdateAt      *time.Time
```

The API layer already treats these as optional, so a `nil` trio simply renders as "No updates yet" on the board.

**Why it matters**: any `LEFT JOIN` (lateral or otherwise) that feeds a `rows.Scan` must scan the nullable side into pointer/`sql.Null*` types, not value types — the join's nullability is invisible in the SQL text but fatal at scan time. When adding a "most recent child" annotation to a list query, default to pointer fields for every column the join can leave `NULL`. Verified live: board loads with a mix of updated and never-updated projects, no 500.
