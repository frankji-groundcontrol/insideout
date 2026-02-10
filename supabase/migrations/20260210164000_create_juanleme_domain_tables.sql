-- =====================================================
-- juanleme 核心域表 (workspace-centric)
-- 说明：updated_at 触发器使用 clock_timestamp()，确保同事务内更新也会推进时间
-- =====================================================

-- 自动更新 updated_at 触发器函数
CREATE OR REPLACE FUNCTION juanleme.set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = clock_timestamp();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 1. 用户档案 (映射 UserProfile 类型)
CREATE TABLE juanleme.profiles (
  id UUID PRIMARY KEY REFERENCES auth.users(id) ON DELETE CASCADE,
  email TEXT NOT NULL,
  username TEXT NOT NULL,
  avatar_url TEXT,
  bio TEXT,
  keywords TEXT[],
  role TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('admin', 'user')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TRIGGER set_profiles_updated_at BEFORE UPDATE ON juanleme.profiles
  FOR EACH ROW EXECUTE FUNCTION juanleme.set_updated_at();

-- 2. 工作空间 (映射 Workshop 类型)
CREATE TABLE juanleme.workspaces (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  cover_url TEXT,
  code TEXT NOT NULL UNIQUE,
  creator_id UUID NOT NULL REFERENCES juanleme.profiles(id) ON DELETE CASCADE,
  status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'active', 'completed')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_workspaces_creator_id ON juanleme.workspaces(creator_id);
CREATE INDEX idx_workspaces_code ON juanleme.workspaces(code);
CREATE TRIGGER set_workspaces_updated_at BEFORE UPDATE ON juanleme.workspaces
  FOR EACH ROW EXECUTE FUNCTION juanleme.set_updated_at();

-- 3. 工作空间成员 (workspace-centric tenancy)
CREATE TABLE juanleme.workspace_memberships (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id UUID NOT NULL REFERENCES juanleme.workspaces(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES juanleme.profiles(id) ON DELETE CASCADE,
  role TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('admin', 'member')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (workspace_id, user_id)
);
CREATE INDEX idx_memberships_workspace_id ON juanleme.workspace_memberships(workspace_id);
CREATE INDEX idx_memberships_user_id ON juanleme.workspace_memberships(user_id);
CREATE TRIGGER set_memberships_updated_at BEFORE UPDATE ON juanleme.workspace_memberships
  FOR EACH ROW EXECUTE FUNCTION juanleme.set_updated_at();

-- 4. 项目
CREATE TABLE juanleme.projects (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id UUID NOT NULL REFERENCES juanleme.workspaces(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  created_by UUID NOT NULL REFERENCES juanleme.profiles(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_projects_workspace_id ON juanleme.projects(workspace_id);
CREATE INDEX idx_projects_created_by ON juanleme.projects(created_by);
CREATE TRIGGER set_projects_updated_at BEFORE UPDATE ON juanleme.projects
  FOR EACH ROW EXECUTE FUNCTION juanleme.set_updated_at();

-- 5. 工作坊节点 (路线图定义，映射 RoadmapNode 结构部分)
-- 注意: 前端类型用 workshop_id，但 DB 用 workspace_id 作为 FK 列名
-- 适配器层负责映射 workspace_id <-> workshop_id
CREATE TABLE juanleme.workshop_nodes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id UUID NOT NULL REFERENCES juanleme.workspaces(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  "order" INT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (workspace_id, "order")
);
CREATE INDEX idx_workshop_nodes_workspace_id ON juanleme.workshop_nodes(workspace_id);
CREATE TRIGGER set_workshop_nodes_updated_at BEFORE UPDATE ON juanleme.workshop_nodes
  FOR EACH ROW EXECUTE FUNCTION juanleme.set_updated_at();

-- 6. 文档 (用户针对节点的内容/草稿)
CREATE TABLE juanleme.documents (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  node_id UUID NOT NULL REFERENCES juanleme.workshop_nodes(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES juanleme.profiles(id) ON DELETE CASCADE,
  content JSONB,
  status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'submitted', 'completed')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (node_id, user_id)
);
CREATE INDEX idx_documents_node_id ON juanleme.documents(node_id);
CREATE INDEX idx_documents_user_id ON juanleme.documents(user_id);
CREATE TRIGGER set_documents_updated_at BEFORE UPDATE ON juanleme.documents
  FOR EACH ROW EXECUTE FUNCTION juanleme.set_updated_at();

-- 7. 文档版本
CREATE TABLE juanleme.document_revisions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  document_id UUID NOT NULL REFERENCES juanleme.documents(id) ON DELETE CASCADE,
  content JSONB NOT NULL,
  revision INT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_revisions_document_id ON juanleme.document_revisions(document_id);

-- 8. AI 运行记录
CREATE TABLE juanleme.ai_runs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id UUID NOT NULL REFERENCES juanleme.workspaces(id) ON DELETE CASCADE,
  node_id UUID REFERENCES juanleme.workshop_nodes(id) ON DELETE SET NULL,
  user_id UUID NOT NULL REFERENCES juanleme.profiles(id),
  prompt TEXT NOT NULL,
  response TEXT,
  status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'succeeded', 'failed')),
  error_message TEXT,
  idempotency_key TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_ai_runs_workspace_id ON juanleme.ai_runs(workspace_id);
CREATE INDEX idx_ai_runs_user_id ON juanleme.ai_runs(user_id);
CREATE INDEX idx_ai_runs_idempotency ON juanleme.ai_runs(idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE TRIGGER set_ai_runs_updated_at BEFORE UPDATE ON juanleme.ai_runs
  FOR EACH ROW EXECUTE FUNCTION juanleme.set_updated_at();

-- 9. 导出任务
CREATE TABLE juanleme.export_jobs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id UUID NOT NULL REFERENCES juanleme.workspaces(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES juanleme.profiles(id),
  format TEXT NOT NULL CHECK (format IN ('markdown', 'print')),
  status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'succeeded', 'failed')),
  artifact_path TEXT,
  error_message TEXT,
  idempotency_key TEXT,
  expires_at TIMESTAMPTZ DEFAULT (now() + INTERVAL '30 days'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_export_jobs_workspace_id ON juanleme.export_jobs(workspace_id);
CREATE INDEX idx_export_jobs_user_id ON juanleme.export_jobs(user_id);
CREATE INDEX idx_export_jobs_expires_at ON juanleme.export_jobs(expires_at);
CREATE INDEX idx_export_jobs_idempotency ON juanleme.export_jobs(idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE TRIGGER set_export_jobs_updated_at BEFORE UPDATE ON juanleme.export_jobs
  FOR EACH ROW EXECUTE FUNCTION juanleme.set_updated_at();
