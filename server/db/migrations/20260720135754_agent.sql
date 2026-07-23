CREATE TABLE insideout.agent_conversations (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id uuid NOT NULL REFERENCES insideout.workspaces(id) ON DELETE CASCADE,
  user_id uuid NOT NULL REFERENCES insideout.users(id),
  prd_id uuid NOT NULL REFERENCES insideout.prds(id) ON DELETE CASCADE,
  stage text NOT NULL DEFAULT 'clarify'
    CHECK (stage IN ('clarify', 'draft', 'critique', 'finalize')),
  status text NOT NULL DEFAULT 'active'
    CHECK (status IN ('active', 'completed', 'abandoned')),
  meta jsonb NOT NULL DEFAULT '{}', -- rolling summary, stage notes / 滚动摘要、阶段笔记
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ON insideout.agent_conversations (user_id);
CREATE INDEX ON insideout.agent_conversations (prd_id);

CREATE TRIGGER set_updated_at
  BEFORE UPDATE ON insideout.agent_conversations
  FOR EACH ROW EXECUTE FUNCTION insideout.set_updated_at();

CREATE TABLE insideout.agent_messages (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  conversation_id uuid NOT NULL REFERENCES insideout.agent_conversations(id) ON DELETE CASCADE,
  role text NOT NULL CHECK (role IN ('user', 'assistant', 'tool')),
  content text NOT NULL DEFAULT '',
  tool_calls jsonb,      -- assistant tool-call payloads / assistant 的工具调用
  tool_call_id text,     -- for role='tool' results / role='tool' 的结果关联
  tokens int,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ON insideout.agent_messages (conversation_id, created_at);

-- ai_runs: one row per user message sent to the agent. Counting source for
-- the rate limiter (failed runs count too, by design — anti-abuse).
-- ai_runs：每条用户发给 Agent 的消息一行。限流器的计数来源（失败的调用也计数，属设计，用于防滥用）。
CREATE TABLE insideout.ai_runs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id uuid NOT NULL REFERENCES insideout.workspaces(id) ON DELETE CASCADE,
  conversation_id uuid REFERENCES insideout.agent_conversations(id) ON DELETE SET NULL,
  user_id uuid NOT NULL REFERENCES insideout.users(id),
  prompt text NOT NULL,
  response text,
  status text NOT NULL DEFAULT 'pending'
    CHECK (status IN ('pending', 'running', 'succeeded', 'failed')),
  error_message text,
  idempotency_key text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ON insideout.ai_runs (user_id, created_at);
CREATE UNIQUE INDEX ON insideout.ai_runs (idempotency_key) WHERE idempotency_key IS NOT NULL;

CREATE TRIGGER set_updated_at
  BEFORE UPDATE ON insideout.ai_runs
  FOR EACH ROW EXECUTE FUNCTION insideout.set_updated_at();

-- ai_run_events: per-call usage telemetry (kept from the live system's
-- ai_run_events table — see 01-database.md §4).
-- ai_run_events：逐次调用的用量遥测（沿用线上系统的 ai_run_events 表——见 01-database.md §4）。
CREATE TABLE insideout.ai_run_events (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  run_id uuid NOT NULL REFERENCES insideout.ai_runs(id) ON DELETE CASCADE,
  event_type text NOT NULL,
  model text,
  input_tokens int,
  output_tokens int,
  latency_ms int,
  request_messages jsonb,
  response_payload jsonb,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ON insideout.ai_run_events (run_id);
