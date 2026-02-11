-- =====================================================
-- AI rate limit + circuit breaker assertions
-- Run RED assertion before applying migration, then GREEN suite after.
-- =====================================================

-- RED: this should fail before migration is applied.
DO $$
DECLARE
  v_count INT;
BEGIN
  SELECT count(*) INTO v_count
  FROM information_schema.routines
  WHERE routine_schema = 'juanleme'
    AND routine_name = 'ai_check_rate_limit';

  ASSERT v_count = 1, format('RED expected failure: ai_check_rate_limit missing (count=%s)', v_count);
END;
$$;

-- GREEN: required RPCs exist.
DO $$
DECLARE
  v_count INT;
BEGIN
  SELECT count(*) INTO v_count
  FROM information_schema.routines
  WHERE routine_schema = 'juanleme'
    AND routine_name IN (
      'get_ai_config',
      'ai_get_workshop_node',
      'ai_get_workspace_membership_me',
      'ai_get_run_by_idempotency_me',
      'ai_create_run_service',
      'ai_update_run_service',
      'ai_check_rate_limit',
      'ai_check_circuit',
      'ai_record_circuit_result'
    );

  ASSERT v_count = 9, format('GREEN expected 9 RPCs, got %s', v_count);
END;
$$;

-- GREEN: circuit breaker table exists.
DO $$
DECLARE
  v_count INT;
BEGIN
  SELECT count(*) INTO v_count
  FROM information_schema.tables
  WHERE table_schema = 'juanleme'
    AND table_name = 'ai_circuit_breaker';

  ASSERT v_count = 1, format('GREEN expected ai_circuit_breaker table, got %s', v_count);
END;
$$;
