-- JWT + RLS defense-in-depth: the Go app layer already enforces every
-- rule in docs/plans/2026-07-20-go-rewrite/01-database.md §5 inside
-- transactions (TOCTOU-safe membership/ownership checks). This migration
-- adds the same rules as Postgres RLS policies, keyed off a per-
-- transaction session variable the Go store layer sets from the already-
-- validated JWT's user id — a DB-level backstop against app-layer bugs,
-- not a boundary against insideout_app itself (it's the only role that
-- ever connects; RLS here guards against *our own* logic mistakes).
--
-- insideout_app owns every table below, so RLS must be FORCEd — by
-- default Postgres exempts the table owner from its own RLS policies.

CREATE OR REPLACE FUNCTION insideout.current_user_id()
RETURNS uuid
LANGUAGE sql STABLE
AS $$
  SELECT NULLIF(current_setting('app.user_id', true), '')::uuid
$$;

-- ---------------------------------------------------------------------
-- users: read self or co-member of a shared workspace; update self only.
-- Pre-auth paths (login-by-email, registration) run with app.user_id
-- unset — current_user_id() IS NULL is treated as a trusted system
-- context, since insideout_app is never called by anything but our own
-- backend and those two operations inherently precede having an
-- authenticated identity. See internal/store/users.go.
-- ---------------------------------------------------------------------
ALTER TABLE insideout.users ENABLE ROW LEVEL SECURITY;
ALTER TABLE insideout.users FORCE ROW LEVEL SECURITY;

CREATE POLICY users_select ON insideout.users
  FOR SELECT USING (
    insideout.current_user_id() IS NULL
    OR id = insideout.current_user_id()
    OR EXISTS (
      SELECT 1 FROM insideout.workspace_memberships m1
      JOIN insideout.workspace_memberships m2 ON m1.workspace_id = m2.workspace_id
      WHERE m1.user_id = insideout.users.id AND m2.user_id = insideout.current_user_id()
    )
  );

CREATE POLICY users_insert ON insideout.users
  FOR INSERT WITH CHECK (true);

CREATE POLICY users_update ON insideout.users
  FOR UPDATE
  USING (id = insideout.current_user_id())
  WITH CHECK (id = insideout.current_user_id());

-- ---------------------------------------------------------------------
-- workspaces: read members only; update creator/admin; delete creator.
--
-- Exception: joining by invite code is an authenticated action where the
-- caller isn't a member *yet* — the whole point is to become one. RLS
-- can't inspect a query's own WHERE clause to know "the code" was asked
-- for, so JoinWorkspace (store/workspaces.go) sets a second, narrowly-
-- scoped session var (app.join_code) immediately before its lookup;
-- knowing the code is exactly the credential the invite system already
-- treats as sufficient, so this doesn't broaden visibility beyond what
-- the app already grants — it just lets that one exact-code lookup pass
-- RLS instead of the whole `workspaces` table needing a blanket bypass.
-- ---------------------------------------------------------------------
ALTER TABLE insideout.workspaces ENABLE ROW LEVEL SECURITY;
ALTER TABLE insideout.workspaces FORCE ROW LEVEL SECURITY;

CREATE POLICY workspaces_select ON insideout.workspaces
  FOR SELECT USING (
    EXISTS (
      SELECT 1 FROM insideout.workspace_memberships m
      WHERE m.workspace_id = insideout.workspaces.id AND m.user_id = insideout.current_user_id()
    )
    OR (
      status = 'active'
      AND code = NULLIF(current_setting('app.join_code', true), '')
    )
  );

CREATE POLICY workspaces_insert ON insideout.workspaces
  FOR INSERT WITH CHECK (creator_id = insideout.current_user_id());

CREATE POLICY workspaces_update ON insideout.workspaces
  FOR UPDATE USING (
    creator_id = insideout.current_user_id()
    OR EXISTS (
      SELECT 1 FROM insideout.workspace_memberships m
      WHERE m.workspace_id = insideout.workspaces.id AND m.user_id = insideout.current_user_id() AND m.role = 'admin'
    )
  );

CREATE POLICY workspaces_delete ON insideout.workspaces
  FOR DELETE USING (creator_id = insideout.current_user_id());

-- ---------------------------------------------------------------------
-- workspace_memberships: read fellow members; join/self-insert; role
-- change by admin; remove by admin or self-leave.
-- ---------------------------------------------------------------------
ALTER TABLE insideout.workspace_memberships ENABLE ROW LEVEL SECURITY;
ALTER TABLE insideout.workspace_memberships FORCE ROW LEVEL SECURITY;

CREATE POLICY workspace_memberships_select ON insideout.workspace_memberships
  FOR SELECT USING (
    EXISTS (
      SELECT 1 FROM insideout.workspace_memberships m2
      WHERE m2.workspace_id = insideout.workspace_memberships.workspace_id AND m2.user_id = insideout.current_user_id()
    )
  );

CREATE POLICY workspace_memberships_insert ON insideout.workspace_memberships
  FOR INSERT WITH CHECK (user_id = insideout.current_user_id());

CREATE POLICY workspace_memberships_update ON insideout.workspace_memberships
  FOR UPDATE USING (
    EXISTS (
      SELECT 1 FROM insideout.workspace_memberships m2
      WHERE m2.workspace_id = insideout.workspace_memberships.workspace_id AND m2.user_id = insideout.current_user_id() AND m2.role = 'admin'
    )
  );

CREATE POLICY workspace_memberships_delete ON insideout.workspace_memberships
  FOR DELETE USING (
    user_id = insideout.current_user_id()
    OR EXISTS (
      SELECT 1 FROM insideout.workspace_memberships m2
      WHERE m2.workspace_id = insideout.workspace_memberships.workspace_id AND m2.user_id = insideout.current_user_id() AND m2.role = 'admin'
    )
  );

-- ---------------------------------------------------------------------
-- projects: read members; create any member; update owner/admin; delete
-- admin.
-- ---------------------------------------------------------------------
ALTER TABLE insideout.projects ENABLE ROW LEVEL SECURITY;
ALTER TABLE insideout.projects FORCE ROW LEVEL SECURITY;

CREATE POLICY projects_select ON insideout.projects
  FOR SELECT USING (
    EXISTS (
      SELECT 1 FROM insideout.workspace_memberships m
      WHERE m.workspace_id = insideout.projects.workspace_id AND m.user_id = insideout.current_user_id()
    )
  );

CREATE POLICY projects_insert ON insideout.projects
  FOR INSERT WITH CHECK (
    created_by = insideout.current_user_id()
    AND EXISTS (
      SELECT 1 FROM insideout.workspace_memberships m
      WHERE m.workspace_id = insideout.projects.workspace_id AND m.user_id = insideout.current_user_id()
    )
  );

CREATE POLICY projects_update ON insideout.projects
  FOR UPDATE USING (
    owner_id = insideout.current_user_id()
    OR EXISTS (
      SELECT 1 FROM insideout.workspace_memberships m
      WHERE m.workspace_id = insideout.projects.workspace_id AND m.user_id = insideout.current_user_id() AND m.role = 'admin'
    )
  );

CREATE POLICY projects_delete ON insideout.projects
  FOR DELETE USING (
    EXISTS (
      SELECT 1 FROM insideout.workspace_memberships m
      WHERE m.workspace_id = insideout.projects.workspace_id AND m.user_id = insideout.current_user_id() AND m.role = 'admin'
    )
  );

-- ---------------------------------------------------------------------
-- project_updates: read members; create any member; edit/delete author
-- or admin.
-- ---------------------------------------------------------------------
ALTER TABLE insideout.project_updates ENABLE ROW LEVEL SECURITY;
ALTER TABLE insideout.project_updates FORCE ROW LEVEL SECURITY;

CREATE POLICY project_updates_select ON insideout.project_updates
  FOR SELECT USING (
    EXISTS (
      SELECT 1 FROM insideout.projects p
      JOIN insideout.workspace_memberships m ON m.workspace_id = p.workspace_id
      WHERE p.id = insideout.project_updates.project_id AND m.user_id = insideout.current_user_id()
    )
  );

CREATE POLICY project_updates_insert ON insideout.project_updates
  FOR INSERT WITH CHECK (
    author_id = insideout.current_user_id()
    AND EXISTS (
      SELECT 1 FROM insideout.projects p
      JOIN insideout.workspace_memberships m ON m.workspace_id = p.workspace_id
      WHERE p.id = insideout.project_updates.project_id AND m.user_id = insideout.current_user_id()
    )
  );

CREATE POLICY project_updates_update ON insideout.project_updates
  FOR UPDATE USING (
    author_id = insideout.current_user_id()
    OR EXISTS (
      SELECT 1 FROM insideout.projects p
      JOIN insideout.workspace_memberships m ON m.workspace_id = p.workspace_id
      WHERE p.id = insideout.project_updates.project_id AND m.user_id = insideout.current_user_id() AND m.role = 'admin'
    )
  );

CREATE POLICY project_updates_delete ON insideout.project_updates
  FOR DELETE USING (
    author_id = insideout.current_user_id()
    OR EXISTS (
      SELECT 1 FROM insideout.projects p
      JOIN insideout.workspace_memberships m ON m.workspace_id = p.workspace_id
      WHERE p.id = insideout.project_updates.project_id AND m.user_id = insideout.current_user_id() AND m.role = 'admin'
    )
  );

-- ---------------------------------------------------------------------
-- ideas: read members; create any member; update author only; drop
-- (soft, via UPDATE status) author or admin — no hard delete exists.
-- ---------------------------------------------------------------------
ALTER TABLE insideout.ideas ENABLE ROW LEVEL SECURITY;
ALTER TABLE insideout.ideas FORCE ROW LEVEL SECURITY;

CREATE POLICY ideas_select ON insideout.ideas
  FOR SELECT USING (
    EXISTS (
      SELECT 1 FROM insideout.workspace_memberships m
      WHERE m.workspace_id = insideout.ideas.workspace_id AND m.user_id = insideout.current_user_id()
    )
  );

CREATE POLICY ideas_insert ON insideout.ideas
  FOR INSERT WITH CHECK (
    author_id = insideout.current_user_id()
    AND EXISTS (
      SELECT 1 FROM insideout.workspace_memberships m
      WHERE m.workspace_id = insideout.ideas.workspace_id AND m.user_id = insideout.current_user_id()
    )
  );

-- ponytail: a single broad UPDATE policy (author-or-admin) covers both
-- UpdateIdea (author-only in Go) and DropIdea (author-or-admin in Go) —
-- RLS is a coarser backstop here, not a line-for-line replica of each
-- Go function's exact rule; the app layer keeps the precise split.
CREATE POLICY ideas_update ON insideout.ideas
  FOR UPDATE USING (
    author_id = insideout.current_user_id()
    OR EXISTS (
      SELECT 1 FROM insideout.workspace_memberships m
      WHERE m.workspace_id = insideout.ideas.workspace_id AND m.user_id = insideout.current_user_id() AND m.role = 'admin'
    )
  );

-- ---------------------------------------------------------------------
-- prds: read members; edit sections author-or-admin; status transitions
-- keep their author-vs-admin split app-side (see prds.go comment above
-- ideas_update — same reasoning: RLS is a coarser author-or-admin
-- backstop, not a transition-aware replica).
-- ---------------------------------------------------------------------
ALTER TABLE insideout.prds ENABLE ROW LEVEL SECURITY;
ALTER TABLE insideout.prds FORCE ROW LEVEL SECURITY;

CREATE POLICY prds_select ON insideout.prds
  FOR SELECT USING (
    EXISTS (
      SELECT 1 FROM insideout.workspace_memberships m
      WHERE m.workspace_id = insideout.prds.workspace_id AND m.user_id = insideout.current_user_id()
    )
  );

CREATE POLICY prds_insert ON insideout.prds
  FOR INSERT WITH CHECK (author_id = insideout.current_user_id());

CREATE POLICY prds_update ON insideout.prds
  FOR UPDATE USING (
    author_id = insideout.current_user_id()
    OR EXISTS (
      SELECT 1 FROM insideout.workspace_memberships m
      WHERE m.workspace_id = insideout.prds.workspace_id AND m.user_id = insideout.current_user_id() AND m.role = 'admin'
    )
  );

-- ---------------------------------------------------------------------
-- prd_revisions: read members (via prd's workspace); insert by the PRD's
-- author only; immutable log, no update/delete policy (default deny).
-- ---------------------------------------------------------------------
ALTER TABLE insideout.prd_revisions ENABLE ROW LEVEL SECURITY;
ALTER TABLE insideout.prd_revisions FORCE ROW LEVEL SECURITY;

CREATE POLICY prd_revisions_select ON insideout.prd_revisions
  FOR SELECT USING (
    EXISTS (
      SELECT 1 FROM insideout.prds p
      JOIN insideout.workspace_memberships m ON m.workspace_id = p.workspace_id
      WHERE p.id = insideout.prd_revisions.prd_id AND m.user_id = insideout.current_user_id()
    )
  );

CREATE POLICY prd_revisions_insert ON insideout.prd_revisions
  FOR INSERT WITH CHECK (
    created_by = insideout.current_user_id()
    AND EXISTS (
      SELECT 1 FROM insideout.prds p
      WHERE p.id = insideout.prd_revisions.prd_id AND p.author_id = insideout.current_user_id()
    )
  );

-- ---------------------------------------------------------------------
-- agent_conversations / agent_messages: owner-only, both read and write.
-- ---------------------------------------------------------------------
ALTER TABLE insideout.agent_conversations ENABLE ROW LEVEL SECURITY;
ALTER TABLE insideout.agent_conversations FORCE ROW LEVEL SECURITY;

CREATE POLICY agent_conversations_all ON insideout.agent_conversations
  FOR ALL
  USING (user_id = insideout.current_user_id())
  WITH CHECK (user_id = insideout.current_user_id());

ALTER TABLE insideout.agent_messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE insideout.agent_messages FORCE ROW LEVEL SECURITY;

CREATE POLICY agent_messages_all ON insideout.agent_messages
  FOR ALL
  USING (
    EXISTS (
      SELECT 1 FROM insideout.agent_conversations c
      WHERE c.id = insideout.agent_messages.conversation_id AND c.user_id = insideout.current_user_id()
    )
  )
  WITH CHECK (
    EXISTS (
      SELECT 1 FROM insideout.agent_conversations c
      WHERE c.id = insideout.agent_messages.conversation_id AND c.user_id = insideout.current_user_id()
    )
  );

-- ---------------------------------------------------------------------
-- ai_runs: read/write own only (writes are server-internal, always tied
-- to the authenticated user whose action triggered the run).
-- ai_run_events and ai_circuit_breaker are pure system telemetry with no
-- per-end-user access story (not in the checklist) — left without RLS.
-- ---------------------------------------------------------------------
ALTER TABLE insideout.ai_runs ENABLE ROW LEVEL SECURITY;
ALTER TABLE insideout.ai_runs FORCE ROW LEVEL SECURITY;

CREATE POLICY ai_runs_all ON insideout.ai_runs
  FOR ALL
  USING (user_id = insideout.current_user_id())
  WITH CHECK (user_id = insideout.current_user_id());
