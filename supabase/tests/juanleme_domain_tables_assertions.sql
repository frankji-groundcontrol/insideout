-- =====================================================
-- juanleme 域表 TDD 断言脚本
-- 使用方式：先执行 RED 段（迁移前），再执行 GREEN 段（迁移后）
-- =====================================================

-- RED：期望失败（迁移前应为 0 张表）
DO $$
DECLARE
  v_count INT;
BEGIN
  SELECT count(*) INTO v_count
  FROM information_schema.tables
  WHERE table_schema = 'juanleme';

  ASSERT v_count = 9, format('RED phase expected failure: expected 9 tables, got %s', v_count);
END;
$$;

-- RED 基线：确认迁移前当前计数
SELECT count(*) AS table_count
FROM information_schema.tables
WHERE table_schema = 'juanleme';

-- GREEN：迁移后断言通过
DO $$
DECLARE
  v_count INT;
BEGIN
  SELECT count(*) INTO v_count
  FROM information_schema.tables
  WHERE table_schema = 'juanleme';

  ASSERT v_count = 9, format('GREEN phase expected 9 tables, got %s', v_count);
END;
$$;

-- 1) 9 张表都在 juanleme schema 下
SELECT table_name
FROM information_schema.tables
WHERE table_schema = 'juanleme'
ORDER BY table_name;

-- 2) public schema 不应出现业务表
SELECT count(*) AS leaked_count
FROM information_schema.tables
WHERE table_schema = 'public'
  AND table_name IN (
    'profiles',
    'workspaces',
    'workspace_memberships',
    'projects',
    'workshop_nodes',
    'documents',
    'document_revisions',
    'ai_runs',
    'export_jobs'
  );

-- 3) FK 完整链路插入校验
DO $$
DECLARE
  v_profile_id UUID := gen_random_uuid();
  v_ws_id UUID;
  v_project_id UUID;
  v_node_id UUID;
  v_doc_id UUID;
BEGIN
  INSERT INTO auth.users (id, email, instance_id, aud, role, encrypted_password, email_confirmed_at, created_at, updated_at, confirmation_token, recovery_token)
  VALUES (v_profile_id, 'test_fk@test.com', '00000000-0000-0000-0000-000000000000', 'authenticated', 'authenticated', crypt('test1234', gen_salt('bf')), now(), now(), now(), '', '')
  ON CONFLICT (id) DO NOTHING;

  INSERT INTO juanleme.profiles (id, email, username) VALUES (v_profile_id, 'test_fk@test.com', 'tester');
  INSERT INTO juanleme.workspaces (id, title, code, creator_id) VALUES (gen_random_uuid(), 'Test WS', 'FKTEST', v_profile_id) RETURNING id INTO v_ws_id;
  INSERT INTO juanleme.workspace_memberships (workspace_id, user_id, role) VALUES (v_ws_id, v_profile_id, 'admin');
  INSERT INTO juanleme.projects (workspace_id, title, created_by) VALUES (v_ws_id, 'Test Proj', v_profile_id) RETURNING id INTO v_project_id;
  INSERT INTO juanleme.workshop_nodes (workspace_id, title, "order") VALUES (v_ws_id, 'Node 1', 1) RETURNING id INTO v_node_id;
  INSERT INTO juanleme.documents (node_id, user_id, content, status) VALUES (v_node_id, v_profile_id, '{"text":"hello"}'::jsonb, 'draft') RETURNING id INTO v_doc_id;
  INSERT INTO juanleme.document_revisions (document_id, content, revision) VALUES (v_doc_id, '{"text":"hello"}'::jsonb, 1);
  INSERT INTO juanleme.ai_runs (workspace_id, node_id, user_id, prompt, status) VALUES (v_ws_id, v_node_id, v_profile_id, 'test prompt', 'pending');
  INSERT INTO juanleme.export_jobs (workspace_id, user_id, format, status) VALUES (v_ws_id, v_profile_id, 'markdown', 'pending');

  -- 先删 workspace，触发级联清理依赖行，再删用户档案和 auth 用户
  DELETE FROM juanleme.workspaces WHERE id = v_ws_id;
  DELETE FROM juanleme.profiles WHERE id = v_profile_id;
  DELETE FROM auth.users WHERE id = v_profile_id;

  RAISE NOTICE 'FK chain integrity: PASSED';
END;
$$;

-- 4) 唯一约束校验（workspace.code）
DO $$
DECLARE
  v_uid UUID := gen_random_uuid();
BEGIN
  INSERT INTO auth.users (id, email, instance_id, aud, role, encrypted_password, email_confirmed_at, created_at, updated_at, confirmation_token, recovery_token)
  VALUES (v_uid, 'test_uniq@test.com', '00000000-0000-0000-0000-000000000000', 'authenticated', 'authenticated', crypt('test1234', gen_salt('bf')), now(), now(), now(), '', '');
  INSERT INTO juanleme.profiles (id, email, username) VALUES (v_uid, 'test_uniq@test.com', 'uniq_tester');

  INSERT INTO juanleme.workspaces (title, code, creator_id) VALUES ('WS A', 'UNIQ01', v_uid);
  BEGIN
    INSERT INTO juanleme.workspaces (title, code, creator_id) VALUES ('WS B', 'UNIQ01', v_uid);
    RAISE EXCEPTION 'Should have failed on duplicate code';
  EXCEPTION WHEN unique_violation THEN
    RAISE NOTICE 'Unique code constraint: PASSED';
  END;

  DELETE FROM juanleme.profiles WHERE id = v_uid;
  DELETE FROM auth.users WHERE id = v_uid;
END;
$$;

-- 5) updated_at 审计触发器校验
DO $$
DECLARE
  v_uid UUID := gen_random_uuid();
  v_created TIMESTAMPTZ;
  v_updated TIMESTAMPTZ;
BEGIN
  INSERT INTO auth.users (id, email, instance_id, aud, role, encrypted_password, email_confirmed_at, created_at, updated_at, confirmation_token, recovery_token)
  VALUES (v_uid, 'test_audit@test.com', '00000000-0000-0000-0000-000000000000', 'authenticated', 'authenticated', crypt('test1234', gen_salt('bf')), now(), now(), now(), '', '');
  INSERT INTO juanleme.profiles (id, email, username) VALUES (v_uid, 'test_audit@test.com', 'audit_tester');

  SELECT created_at, updated_at INTO v_created, v_updated FROM juanleme.profiles WHERE id = v_uid;
  ASSERT v_created IS NOT NULL, 'created_at must not be null';
  ASSERT v_updated IS NOT NULL, 'updated_at must not be null';

  PERFORM pg_sleep(0.1);
  UPDATE juanleme.profiles SET username = 'updated_tester' WHERE id = v_uid;

  SELECT updated_at INTO v_updated FROM juanleme.profiles WHERE id = v_uid;
  ASSERT v_updated > v_created, 'updated_at must be > created_at after update';

  DELETE FROM juanleme.profiles WHERE id = v_uid;
  DELETE FROM auth.users WHERE id = v_uid;

  RAISE NOTICE 'Audit trigger: PASSED';
END;
$$;
