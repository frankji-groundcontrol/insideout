-- Align AI RPC definitions with edge functions and add traffic shaping primitives.

-- =====================================================
-- 1) Missing/undocumented AI RPCs used by edge functions
-- =====================================================

CREATE OR REPLACE FUNCTION juanleme.get_ai_config()
RETURNS JSONB
LANGUAGE plpgsql
SECURITY DEFINER
STABLE
SET search_path = pg_catalog, juanleme
AS $$
DECLARE
  v_base_url TEXT;
  v_auth_token TEXT;
BEGIN
  IF to_regclass('vault.decrypted_secrets') IS NULL THEN
    RETURN jsonb_build_object('base_url', NULL, 'auth_token', NULL);
  END IF;

  SELECT ds.decrypted_secret
  INTO v_base_url
  FROM vault.decrypted_secrets AS ds
  WHERE ds.name IN ('ai_base_url', 'anthropic_base_url')
  ORDER BY CASE WHEN ds.name = 'ai_base_url' THEN 0 ELSE 1 END
  LIMIT 1;

  SELECT ds.decrypted_secret
  INTO v_auth_token
  FROM vault.decrypted_secrets AS ds
  WHERE ds.name IN ('ai_auth_token', 'anthropic_auth_token')
  ORDER BY CASE WHEN ds.name = 'ai_auth_token' THEN 0 ELSE 1 END
  LIMIT 1;

  RETURN jsonb_build_object('base_url', v_base_url, 'auth_token', v_auth_token);
END;
$$;

CREATE OR REPLACE FUNCTION juanleme.ai_get_workshop_node(p_node_id UUID)
RETURNS TABLE (
  id UUID,
  workspace_id UUID
)
LANGUAGE sql
SECURITY DEFINER
STABLE
SET search_path = pg_catalog, juanleme
AS $$
  SELECT wn.id, wn.workspace_id
  FROM juanleme.workshop_nodes AS wn
  WHERE wn.id = p_node_id;
$$;

CREATE OR REPLACE FUNCTION juanleme.ai_get_workspace_membership_me(p_workspace_id UUID)
RETURNS TABLE (
  user_id UUID
)
LANGUAGE sql
SECURITY DEFINER
STABLE
SET search_path = pg_catalog, juanleme
AS $$
  SELECT wm.user_id
  FROM juanleme.workspace_memberships AS wm
  WHERE wm.workspace_id = p_workspace_id
    AND wm.user_id = auth.uid()
  LIMIT 1;
$$;

CREATE OR REPLACE FUNCTION juanleme.ai_get_run_by_idempotency_me(p_idempotency_key TEXT)
RETURNS TABLE (
  id UUID,
  status TEXT,
  response TEXT,
  error_message TEXT,
  updated_at TIMESTAMPTZ
)
LANGUAGE sql
SECURITY DEFINER
STABLE
SET search_path = pg_catalog, juanleme
AS $$
  SELECT ar.id, ar.status, ar.response, ar.error_message, ar.updated_at
  FROM juanleme.ai_runs AS ar
  WHERE ar.user_id = auth.uid()
    AND ar.idempotency_key = p_idempotency_key
  ORDER BY ar.created_at DESC
  LIMIT 1;
$$;

CREATE OR REPLACE FUNCTION juanleme.ai_create_run_service(
  p_workspace_id UUID,
  p_node_id UUID,
  p_user_id UUID,
  p_prompt TEXT,
  p_status TEXT DEFAULT 'pending',
  p_idempotency_key TEXT DEFAULT NULL
)
RETURNS UUID
LANGUAGE plpgsql
SECURITY DEFINER
VOLATILE
SET search_path = pg_catalog, juanleme
AS $$
DECLARE
  v_run_id UUID;
BEGIN
  INSERT INTO juanleme.ai_runs (
    workspace_id,
    node_id,
    user_id,
    prompt,
    status,
    idempotency_key
  )
  VALUES (
    p_workspace_id,
    p_node_id,
    p_user_id,
    p_prompt,
    p_status,
    p_idempotency_key
  )
  RETURNING id INTO v_run_id;

  RETURN v_run_id;
END;
$$;

CREATE OR REPLACE FUNCTION juanleme.ai_update_run_service(
  p_run_id UUID,
  p_status TEXT,
  p_response TEXT DEFAULT NULL,
  p_error_message TEXT DEFAULT NULL
)
RETURNS BOOLEAN
LANGUAGE sql
SECURITY DEFINER
VOLATILE
SET search_path = pg_catalog, juanleme
AS $$
  WITH upd AS (
    UPDATE juanleme.ai_runs
    SET
      status = p_status,
      response = COALESCE(p_response, response),
      error_message = COALESCE(p_error_message, error_message),
      updated_at = clock_timestamp()
    WHERE id = p_run_id
    RETURNING 1
  )
  SELECT EXISTS (SELECT 1 FROM upd);
$$;

-- =====================================================
-- 2) Rate limiting RPC (sliding window over ai_runs)
-- =====================================================

CREATE OR REPLACE FUNCTION juanleme.ai_check_rate_limit(
  p_user_id UUID,
  p_workspace_id UUID DEFAULT NULL
)
RETURNS TABLE (
  allowed BOOLEAN,
  current_count INTEGER,
  max_requests INTEGER,
  retry_after_seconds INTEGER
)
LANGUAGE plpgsql
SECURITY DEFINER
VOLATILE
SET search_path = pg_catalog, juanleme
AS $$
DECLARE
  v_now TIMESTAMPTZ := clock_timestamp();
  v_minute_limit INTEGER := 10;
  v_hour_limit INTEGER := 60;
  v_minute_count INTEGER := 0;
  v_hour_count INTEGER := 0;
  v_minute_oldest TIMESTAMPTZ;
  v_hour_oldest TIMESTAMPTZ;
  v_retry_seconds INTEGER := 0;
BEGIN
  PERFORM pg_advisory_xact_lock(hashtextextended(p_user_id::TEXT, 0));

  SELECT
    COUNT(*) FILTER (WHERE ar.created_at >= (v_now - INTERVAL '1 minute')),
    COUNT(*) FILTER (WHERE ar.created_at >= (v_now - INTERVAL '1 hour')),
    MIN(ar.created_at) FILTER (WHERE ar.created_at >= (v_now - INTERVAL '1 minute')),
    MIN(ar.created_at) FILTER (WHERE ar.created_at >= (v_now - INTERVAL '1 hour'))
  INTO v_minute_count, v_hour_count, v_minute_oldest, v_hour_oldest
  FROM juanleme.ai_runs AS ar
  WHERE ar.user_id = p_user_id;

  IF v_minute_count >= v_minute_limit THEN
    v_retry_seconds := GREATEST(1, CEIL(EXTRACT(EPOCH FROM ((v_minute_oldest + INTERVAL '1 minute') - v_now)))::INTEGER);
    RETURN QUERY SELECT FALSE, v_minute_count, v_minute_limit, v_retry_seconds;
    RETURN;
  END IF;

  IF v_hour_count >= v_hour_limit THEN
    v_retry_seconds := GREATEST(1, CEIL(EXTRACT(EPOCH FROM ((v_hour_oldest + INTERVAL '1 hour') - v_now)))::INTEGER);
    RETURN QUERY SELECT FALSE, v_hour_count, v_hour_limit, v_retry_seconds;
    RETURN;
  END IF;

  RETURN QUERY SELECT TRUE, v_minute_count, v_minute_limit, 0;
END;
$$;

-- =====================================================
-- 3) DB-persisted circuit breaker
-- =====================================================

CREATE TABLE IF NOT EXISTS juanleme.ai_circuit_breaker (
  id TEXT PRIMARY KEY,
  state TEXT NOT NULL DEFAULT 'closed' CHECK (state IN ('closed', 'open', 'half_open')),
  failure_count INTEGER NOT NULL DEFAULT 0,
  success_count INTEGER NOT NULL DEFAULT 0,
  last_failure_at TIMESTAMPTZ,
  opened_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

DROP TRIGGER IF EXISTS set_ai_circuit_breaker_updated_at ON juanleme.ai_circuit_breaker;

CREATE TRIGGER set_ai_circuit_breaker_updated_at BEFORE UPDATE ON juanleme.ai_circuit_breaker
  FOR EACH ROW EXECUTE FUNCTION juanleme.set_updated_at();

ALTER TABLE juanleme.ai_circuit_breaker ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS ai_circuit_breaker_select_deny ON juanleme.ai_circuit_breaker;
DROP POLICY IF EXISTS ai_circuit_breaker_insert_deny ON juanleme.ai_circuit_breaker;
DROP POLICY IF EXISTS ai_circuit_breaker_update_deny ON juanleme.ai_circuit_breaker;
DROP POLICY IF EXISTS ai_circuit_breaker_delete_deny ON juanleme.ai_circuit_breaker;

CREATE POLICY ai_circuit_breaker_select_deny
ON juanleme.ai_circuit_breaker
FOR SELECT
TO authenticated
USING (false);

CREATE POLICY ai_circuit_breaker_insert_deny
ON juanleme.ai_circuit_breaker
FOR INSERT
TO authenticated
WITH CHECK (false);

CREATE POLICY ai_circuit_breaker_update_deny
ON juanleme.ai_circuit_breaker
FOR UPDATE
TO authenticated
USING (false)
WITH CHECK (false);

CREATE POLICY ai_circuit_breaker_delete_deny
ON juanleme.ai_circuit_breaker
FOR DELETE
TO authenticated
USING (false);

INSERT INTO juanleme.ai_circuit_breaker (id)
VALUES ('anthropic')
ON CONFLICT (id) DO NOTHING;

CREATE OR REPLACE FUNCTION juanleme.ai_check_circuit()
RETURNS TABLE (
  allowed BOOLEAN,
  state TEXT,
  retry_after_seconds INTEGER
)
LANGUAGE plpgsql
SECURITY DEFINER
VOLATILE
SET search_path = pg_catalog, juanleme
AS $$
DECLARE
  v_now TIMESTAMPTZ := clock_timestamp();
  v_state TEXT;
  v_opened_at TIMESTAMPTZ;
  v_cooldown_seconds INTEGER := 30;
  v_retry_seconds INTEGER := 0;
BEGIN
  INSERT INTO juanleme.ai_circuit_breaker (id)
  VALUES ('anthropic')
  ON CONFLICT (id) DO NOTHING;

  SELECT cb.state, cb.opened_at
  INTO v_state, v_opened_at
  FROM juanleme.ai_circuit_breaker AS cb
  WHERE cb.id = 'anthropic'
  FOR UPDATE;

  IF v_state = 'open' THEN
    IF v_opened_at IS NOT NULL AND v_opened_at + make_interval(secs => v_cooldown_seconds) <= v_now THEN
      UPDATE juanleme.ai_circuit_breaker
      SET state = 'half_open', success_count = 0, updated_at = clock_timestamp()
      WHERE id = 'anthropic';
      RETURN QUERY SELECT TRUE, 'half_open'::TEXT, 0;
      RETURN;
    END IF;

    v_retry_seconds := GREATEST(1, CEIL(EXTRACT(EPOCH FROM ((COALESCE(v_opened_at, v_now) + make_interval(secs => v_cooldown_seconds)) - v_now)))::INTEGER);
    RETURN QUERY SELECT FALSE, 'open'::TEXT, v_retry_seconds;
    RETURN;
  END IF;

  RETURN QUERY SELECT TRUE, v_state, 0;
END;
$$;

CREATE OR REPLACE FUNCTION juanleme.ai_record_circuit_result(p_success BOOLEAN)
RETURNS TABLE (
  state TEXT,
  failure_count INTEGER,
  success_count INTEGER
)
LANGUAGE plpgsql
SECURITY DEFINER
VOLATILE
SET search_path = pg_catalog, juanleme
AS $$
DECLARE
  v_state TEXT;
  v_failure_count INTEGER;
  v_success_count INTEGER;
  v_failure_threshold INTEGER := 5;
  v_success_threshold INTEGER := 2;
BEGIN
  INSERT INTO juanleme.ai_circuit_breaker (id)
  VALUES ('anthropic')
  ON CONFLICT (id) DO NOTHING;

  SELECT cb.state, cb.failure_count, cb.success_count
  INTO v_state, v_failure_count, v_success_count
  FROM juanleme.ai_circuit_breaker AS cb
  WHERE cb.id = 'anthropic'
  FOR UPDATE;

  IF p_success THEN
    IF v_state = 'half_open' THEN
      v_success_count := v_success_count + 1;
      IF v_success_count >= v_success_threshold THEN
        v_state := 'closed';
        v_success_count := 0;
        v_failure_count := 0;
      END IF;
    ELSE
      v_failure_count := 0;
      v_success_count := 0;
      v_state := 'closed';
    END IF;
  ELSE
    v_success_count := 0;
    v_failure_count := v_failure_count + 1;
    IF v_failure_count >= v_failure_threshold THEN
      v_state := 'open';
    END IF;
  END IF;

  UPDATE juanleme.ai_circuit_breaker
  SET
    state = v_state,
    failure_count = v_failure_count,
    success_count = v_success_count,
    last_failure_at = CASE WHEN p_success THEN last_failure_at ELSE clock_timestamp() END,
    opened_at = CASE WHEN v_state = 'open' THEN COALESCE(opened_at, clock_timestamp()) ELSE NULL END,
    updated_at = clock_timestamp()
  WHERE id = 'anthropic';

  RETURN QUERY
  SELECT cb.state, cb.failure_count, cb.success_count
  FROM juanleme.ai_circuit_breaker AS cb
  WHERE cb.id = 'anthropic';
END;
$$;

-- =====================================================
-- 4) Grants / revokes
-- =====================================================

REVOKE ALL ON FUNCTION juanleme.get_ai_config() FROM PUBLIC, anon, authenticated, service_role;
REVOKE ALL ON FUNCTION juanleme.ai_get_workshop_node(UUID) FROM PUBLIC, anon, authenticated, service_role;
REVOKE ALL ON FUNCTION juanleme.ai_get_workspace_membership_me(UUID) FROM PUBLIC, anon, authenticated, service_role;
REVOKE ALL ON FUNCTION juanleme.ai_get_run_by_idempotency_me(TEXT) FROM PUBLIC, anon, authenticated, service_role;
REVOKE ALL ON FUNCTION juanleme.ai_create_run_service(UUID, UUID, UUID, TEXT, TEXT, TEXT) FROM PUBLIC, anon, authenticated, service_role;
REVOKE ALL ON FUNCTION juanleme.ai_update_run_service(UUID, TEXT, TEXT, TEXT) FROM PUBLIC, anon, authenticated, service_role;
REVOKE ALL ON FUNCTION juanleme.ai_check_rate_limit(UUID, UUID) FROM PUBLIC, anon, authenticated, service_role;
REVOKE ALL ON FUNCTION juanleme.ai_check_circuit() FROM PUBLIC, anon, authenticated, service_role;
REVOKE ALL ON FUNCTION juanleme.ai_record_circuit_result(BOOLEAN) FROM PUBLIC, anon, authenticated, service_role;

GRANT EXECUTE ON FUNCTION juanleme.ai_get_workshop_node(UUID) TO authenticated;
GRANT EXECUTE ON FUNCTION juanleme.ai_get_workspace_membership_me(UUID) TO authenticated;
GRANT EXECUTE ON FUNCTION juanleme.ai_get_run_by_idempotency_me(TEXT) TO authenticated;

GRANT EXECUTE ON FUNCTION juanleme.get_ai_config() TO service_role;
GRANT EXECUTE ON FUNCTION juanleme.ai_create_run_service(UUID, UUID, UUID, TEXT, TEXT, TEXT) TO service_role;
GRANT EXECUTE ON FUNCTION juanleme.ai_update_run_service(UUID, TEXT, TEXT, TEXT) TO service_role;
GRANT EXECUTE ON FUNCTION juanleme.ai_check_rate_limit(UUID, UUID) TO service_role;
GRANT EXECUTE ON FUNCTION juanleme.ai_check_circuit() TO service_role;
GRANT EXECUTE ON FUNCTION juanleme.ai_record_circuit_result(BOOLEAN) TO service_role;
