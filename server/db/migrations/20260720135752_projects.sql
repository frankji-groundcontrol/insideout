CREATE TABLE insideout.projects (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id uuid NOT NULL REFERENCES insideout.workspaces(id) ON DELETE CASCADE,
  title text NOT NULL,
  description text NOT NULL DEFAULT '',
  owner_id uuid REFERENCES insideout.users(id),
  created_by uuid NOT NULL REFERENCES insideout.users(id),
  status text NOT NULL DEFAULT 'active'
    CHECK (status IN ('planning', 'active', 'paused', 'done', 'archived')),
  meta jsonb NOT NULL DEFAULT '{}',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ON insideout.projects (workspace_id);
CREATE INDEX ON insideout.projects (owner_id);

CREATE TRIGGER set_updated_at
  BEFORE UPDATE ON insideout.projects
  FOR EACH ROW EXECUTE FUNCTION insideout.set_updated_at();

CREATE TABLE insideout.project_updates (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id uuid NOT NULL REFERENCES insideout.projects(id) ON DELETE CASCADE,
  author_id uuid NOT NULL REFERENCES insideout.users(id),
  kind text NOT NULL CHECK (kind IN ('progress', 'blocker', 'note')),
  content text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ON insideout.project_updates (project_id, created_at DESC);
