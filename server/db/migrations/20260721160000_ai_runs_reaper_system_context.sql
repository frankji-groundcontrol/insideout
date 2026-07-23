-- The stale-run reaper (server/internal/store/agent_messages.go) runs on
-- a ticker with no authenticated actor, so it needs the same "no
-- app.user_id set = trusted system context" exception the users table
-- already uses (20260720150000_row_level_security.sql), or its UPDATE
-- matches zero rows under FORCE ROW LEVEL SECURITY.
-- 无认证 actor 运行的失效运行清理器（reaper）需要与 users 表相同的
-- "未设置 app.user_id 视为受信任的系统上下文" 例外，否则在 FORCE ROW
-- LEVEL SECURITY 下其 UPDATE 匹配不到任何行。
DROP POLICY IF EXISTS ai_runs_all ON insideout.ai_runs;
CREATE POLICY ai_runs_all ON insideout.ai_runs
  FOR ALL
  USING (insideout.current_user_id() IS NULL OR user_id = insideout.current_user_id())
  WITH CHECK (insideout.current_user_id() IS NULL OR user_id = insideout.current_user_id());
