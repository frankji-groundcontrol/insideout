-- TEST-06's fix: the circuit-breaker SQL functions were hardwired to the
-- one shared 'anthropic' row, so no test could own its breaker state -- the
-- agent package's rule pins had to delta-assert against a row other
-- packages' tests reset mid-flight. Both functions now take a row id that
-- defaults to the production 'anthropic', making a dedicated test row
-- first-class. Bodies are the 20260720135755 definitions with the id
-- parameterized; no rule changed.

-- CREATE OR REPLACE cannot change a function's signature: without these
-- drops the parameterized forms would overload the originals and every
-- zero-arg call would be ambiguous.
DROP FUNCTION IF EXISTS insideout.ai_check_circuit();
DROP FUNCTION IF EXISTS insideout.ai_record_circuit_result(boolean);

CREATE OR REPLACE FUNCTION insideout.ai_check_circuit(p_id text DEFAULT 'anthropic')
RETURNS TABLE (allowed boolean, state text, retry_after_seconds int)
LANGUAGE plpgsql
SET search_path = pg_catalog, insideout
AS $$
DECLARE
  v_row insideout.ai_circuit_breaker%ROWTYPE;
  v_retry int;
BEGIN
  SELECT * INTO v_row FROM insideout.ai_circuit_breaker WHERE id = p_id FOR UPDATE;

  IF v_row.state = 'open' THEN
    IF now() - v_row.opened_at >= interval '30 seconds' THEN
      UPDATE insideout.ai_circuit_breaker
        SET state = 'half_open'
        WHERE id = p_id;
      RETURN QUERY SELECT true, 'half_open'::text, 0;
      RETURN;
    END IF;

    v_retry := GREATEST(1, ceil(extract(epoch FROM (v_row.opened_at + interval '30 seconds' - now())))::int);
    RETURN QUERY SELECT false, 'open'::text, v_retry;
    RETURN;
  END IF;

  RETURN QUERY SELECT true, v_row.state, 0;
END;
$$;

CREATE OR REPLACE FUNCTION insideout.ai_record_circuit_result(p_success boolean, p_id text DEFAULT 'anthropic')
RETURNS void
LANGUAGE plpgsql
SET search_path = pg_catalog, insideout
AS $$
DECLARE
  v_row insideout.ai_circuit_breaker%ROWTYPE;
BEGIN
  SELECT * INTO v_row FROM insideout.ai_circuit_breaker WHERE id = p_id FOR UPDATE;

  IF p_success THEN
    IF v_row.state = 'half_open' THEN
      IF v_row.success_count + 1 >= 2 THEN
        UPDATE insideout.ai_circuit_breaker
          SET state = 'closed', failure_count = 0, success_count = 0, opened_at = NULL
          WHERE id = p_id;
      ELSE
        UPDATE insideout.ai_circuit_breaker
          SET success_count = success_count + 1
          WHERE id = p_id;
      END IF;
    ELSE
      UPDATE insideout.ai_circuit_breaker
        SET failure_count = 0, success_count = 0
        WHERE id = p_id;
    END IF;
  ELSE
    IF v_row.state = 'half_open' THEN
      -- half-open test failed: reopen immediately / 半开测试失败：立即重新开启
      UPDATE insideout.ai_circuit_breaker
        SET state = 'open', opened_at = now(), last_failure_at = now(), success_count = 0
        WHERE id = p_id;
    ELSIF v_row.failure_count + 1 >= 5 THEN
      UPDATE insideout.ai_circuit_breaker
        SET state = 'open', opened_at = now(), last_failure_at = now(),
            failure_count = failure_count + 1, success_count = 0
        WHERE id = p_id;
    ELSE
      UPDATE insideout.ai_circuit_breaker
        SET failure_count = failure_count + 1, last_failure_at = now()
        WHERE id = p_id;
    END IF;
  END IF;
END;
$$;
