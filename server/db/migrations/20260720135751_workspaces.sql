CREATE TABLE insideout.workspaces (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  title text NOT NULL,
  description text NOT NULL DEFAULT '',
  cover_url text,
  code text NOT NULL UNIQUE,
  creator_id uuid NOT NULL REFERENCES insideout.users(id),
  status text NOT NULL DEFAULT 'active' CHECK (status IN ('draft', 'active', 'completed')),
  meta jsonb NOT NULL DEFAULT '{}',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ON insideout.workspaces (creator_id);
CREATE UNIQUE INDEX ON insideout.workspaces (code);

CREATE TRIGGER set_updated_at
  BEFORE UPDATE ON insideout.workspaces
  FOR EACH ROW EXECUTE FUNCTION insideout.set_updated_at();

CREATE TABLE insideout.workspace_memberships (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id uuid NOT NULL REFERENCES insideout.workspaces(id) ON DELETE CASCADE,
  user_id uuid NOT NULL REFERENCES insideout.users(id) ON DELETE CASCADE,
  role text NOT NULL DEFAULT 'member' CHECK (role IN ('admin', 'member')),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (workspace_id, user_id)
);

CREATE INDEX ON insideout.workspace_memberships (user_id);

CREATE TRIGGER set_updated_at
  BEFORE UPDATE ON insideout.workspace_memberships
  FOR EACH ROW EXECUTE FUNCTION insideout.set_updated_at();
