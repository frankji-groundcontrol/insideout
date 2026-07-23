# Verification — what was actually checked

All against the real `DATABASE_URL` and the real `ANTHROPIC_AUTH_TOKEN`
endpoint, per the project's no-mocks rule.

## Backend

- `go build ./... && go vet ./...` green; `go test ./...` green (agent, auth,
  export, store).
- Migration applied to the real shared instance; verified `roadmap_nodes`
  table, `projects.repo_url` column, RLS `FORCE` + the four policies, and the
  `set_updated_at` trigger (via `psql`).
- **Roadmap integration test** (`store/roadmap_test.go`) against real
  Postgres: branched tree build, sibling `position` ordering, cross-project
  parent → `ErrConflict`, member-collaborative update, reparent, **cycle
  guard** (root-under-descendant and node-under-self → `ErrConflict`),
  delete-subtree cascade, and stranger deny (`ErrNotFound` via RLS).
- **Live HTTP** against the running server: register → workspace → project →
  roadmap create/list/update/move/`409` cycle-guard/cascade-delete.
- **GitHub sync live**: linked `octocat/Hello-World`, first sync added 3 real
  commits, second sync added 0 (cursor dedupe), timeline shows real subjects.
- **AI build live**: `POST /prds/{id}/build` with the real Anthropic endpoint
  generated an 8-node branched tree; `POST /roadmap/{nid}/expand` returned 6
  real subtasks.

## Frontend

- `nuxi typecheck` 0 errors; `pnpm test` 20/20; `pnpm build` green.
- Roadmap tree browser-verified light + dark (hierarchy, seals, connectors,
  progress bar, tabs).
- GitHub card + synced-commit timeline browser-verified.
- **"Build the MVP" through the real UI**: PRD page → click → real LLM
  generated an 11-node branched roadmap → navigated to the project.
- Prisma re-theme browser-verified: landing (dark + light), roadmap, dashboard
  — cream-on-black dark, warm-paper light, Almarai/Instrument-Serif fonts,
  motion-v reveals.

## Notes / limitations

- GitHub sync is owner/admin and uses the unauthenticated public API (60
  req/hr without `GITHUB_TOKEN`); private repos unsupported.
- The LLM roadmap build/expand falls back to a deterministic template when no
  provider is configured or output can't be parsed — never hard-fails.
- The dev sandbox blocks `fonts.gstatic.com`, so screenshots show fallback
  fonts; Almarai/Instrument Serif load in a real browser.
