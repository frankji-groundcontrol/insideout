CREATE TABLE insideout.ideas (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id uuid NOT NULL REFERENCES insideout.workspaces(id) ON DELETE CASCADE,
  author_id uuid NOT NULL REFERENCES insideout.users(id),
  title text NOT NULL,
  content text NOT NULL DEFAULT '',
  status text NOT NULL DEFAULT 'inbox'
    CHECK (status IN ('inbox', 'refining', 'converted', 'dropped')),
  prd_id uuid, -- FK added below, after prds exists / 外键在 prds 建成后补加
  meta jsonb NOT NULL DEFAULT '{}',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ON insideout.ideas (workspace_id, status);
CREATE INDEX ON insideout.ideas (author_id);

CREATE TRIGGER set_updated_at
  BEFORE UPDATE ON insideout.ideas
  FOR EACH ROW EXECUTE FUNCTION insideout.set_updated_at();

CREATE TABLE insideout.prds (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id uuid NOT NULL REFERENCES insideout.workspaces(id) ON DELETE CASCADE,
  idea_id uuid REFERENCES insideout.ideas(id) ON DELETE SET NULL,
  project_id uuid REFERENCES insideout.projects(id) ON DELETE SET NULL,
  author_id uuid NOT NULL REFERENCES insideout.users(id),
  title text NOT NULL,
  -- Fixed keys: background, users, goals, nonGoals, stories, requirements,
  -- constraints, risks — each a markdown string. See 03-agents.md §2.
  -- 固定键：background/users/goals/nonGoals/stories/requirements/constraints/risks，
  -- 每个是 markdown 字符串。见 03-agents.md §2。
  sections jsonb NOT NULL DEFAULT '{}',
  status text NOT NULL DEFAULT 'draft'
    CHECK (status IN ('draft', 'reviewing', 'approved', 'rejected')),
  current_revision int NOT NULL DEFAULT 1,
  meta jsonb NOT NULL DEFAULT '{}',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ON insideout.prds (workspace_id, status);
CREATE INDEX ON insideout.prds (idea_id);
CREATE INDEX ON insideout.prds (project_id);

CREATE TRIGGER set_updated_at
  BEFORE UPDATE ON insideout.prds
  FOR EACH ROW EXECUTE FUNCTION insideout.set_updated_at();

ALTER TABLE insideout.ideas
  ADD CONSTRAINT ideas_prd_id_fkey
  FOREIGN KEY (prd_id) REFERENCES insideout.prds(id) ON DELETE SET NULL;

CREATE TABLE insideout.prd_revisions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  prd_id uuid NOT NULL REFERENCES insideout.prds(id) ON DELETE CASCADE,
  revision int NOT NULL,
  sections jsonb NOT NULL,
  created_by uuid NOT NULL REFERENCES insideout.users(id),
  note text,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (prd_id, revision)
);
