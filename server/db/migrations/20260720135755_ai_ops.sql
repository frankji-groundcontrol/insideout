-- Rate limiter + circuit breaker: kept in Postgres (DB-persisted state is
-- correct across multiple Go server instances, unlike in-process counters).
-- Ported verbatim in semantics from the old Supabase RPCs; see
-- docs/plans/2026-07-20-go-rewrite/01-database.md §6.
-- 限流器 + 熔断器：保留在 Postgres 中（数据库持久化状态在多个 Go 服务实例下依然正确，
-- 而进程内计数器做不到）。语义逐字沿用旧 Supabase RPC；见 01-database.md §6。

CREATE TABLE insideout.ai_circuit_breaker (
  id text PRIMARY KEY,
  state text NOT NULL DEFAULT 'closed' CHECK (state IN ('closed', 'open', 'half_open')),
  failure_count int NOT NULL DEFAULT 0,
  success_count int NOT NULL DEFAULT 0,
  last_failure_at timestamptz,
  opened_at timestamptz
);

INSERT INTO insideout.ai_circuit_breaker (id) VALUES ('anthropic')
  ON CONFLICT (id) DO NOTHING;

-- Per-user sliding window: 10/minute and 60/hour. Failed runs count too
-- (anti-abuse, matches the old behavior). Serialized per-user with an
-- advisory lock so concurrent requests from the same user can't race past
-- the limit.
-- 按用户滑动窗口：10/分钟、60/小时。失败的调用也计数（防滥用，沿用旧行为）。
-- 按用户用 advisory lock 串行化，避免同一用户的并发请求抢过限流。
CREATE OR REPLACE FUNCTION insideout.ai_check_rate_limit(p_user_id uuid)
RETURNS TABLE (allowed boolean, current_count int, max_requests int, retry_after_seconds int)
LANGUAGE plpgsql
SET search_path = pg_catalog, insideout
AS $$
DECLARE
  v_count_minute int;
  v_count_hour int;
  v_oldest_minute timestamptz;
  v_oldest_hour timestamptz;
BEGIN
  PERFORM pg_advisory_xact_lock(hashtextextended(p_user_id::text, 0));

  SELECT count(*), min(created_at) INTO v_count_minute, v_oldest_minute
    FROM insideout.ai_runs
    WHERE user_id = p_user_id AND created_at > now() - interval '1 minute';

  SELECT count(*), min(created_at) INTO v_count_hour, v_oldest_hour
    FROM insideout.ai_runs
    WHERE user_id = p_user_id AND created_at > now() - interval '1 hour';

  IF v_count_minute >= 10 THEN
    RETURN QUERY SELECT
      false,
      v_count_minute,
      10,
      GREATEST(1, ceil(extract(epoch FROM (v_oldest_minute + interval '1 minute' - now())))::int);
    RETURN;
  END IF;

  IF v_count_hour >= 60 THEN
    RETURN QUERY SELECT
      false,
      v_count_hour,
      60,
      GREATEST(1, ceil(extract(epoch FROM (v_oldest_hour + interval '1 hour' - now())))::int);
    RETURN;
  END IF;

  RETURN QUERY SELECT true, v_count_minute, 10, 0;
END;
$$;

-- Circuit breaker: 30s cooldown open->half_open, 2 successes in half_open
-- close, 5 consecutive failures open.
-- 熔断器：30 秒冷却 开启->半开，半开状态 2 次成功即关闭，连续 5 次失败即开启。
CREATE OR REPLACE FUNCTION insideout.ai_check_circuit()
RETURNS TABLE (allowed boolean, state text, retry_after_seconds int)
LANGUAGE plpgsql
SET search_path = pg_catalog, insideout
AS $$
DECLARE
  v_row insideout.ai_circuit_breaker%ROWTYPE;
  v_retry int;
BEGIN
  SELECT * INTO v_row FROM insideout.ai_circuit_breaker WHERE id = 'anthropic' FOR UPDATE;

  IF v_row.state = 'open' THEN
    IF now() - v_row.opened_at >= interval '30 seconds' THEN
      UPDATE insideout.ai_circuit_breaker
        SET state = 'half_open'
        WHERE id = 'anthropic';
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

CREATE OR REPLACE FUNCTION insideout.ai_record_circuit_result(p_success boolean)
RETURNS void
LANGUAGE plpgsql
SET search_path = pg_catalog, insideout
AS $$
DECLARE
  v_row insideout.ai_circuit_breaker%ROWTYPE;
BEGIN
  SELECT * INTO v_row FROM insideout.ai_circuit_breaker WHERE id = 'anthropic' FOR UPDATE;

  IF p_success THEN
    IF v_row.state = 'half_open' THEN
      IF v_row.success_count + 1 >= 2 THEN
        UPDATE insideout.ai_circuit_breaker
          SET state = 'closed', failure_count = 0, success_count = 0, opened_at = NULL
          WHERE id = 'anthropic';
      ELSE
        UPDATE insideout.ai_circuit_breaker
          SET success_count = success_count + 1
          WHERE id = 'anthropic';
      END IF;
    ELSE
      UPDATE insideout.ai_circuit_breaker
        SET failure_count = 0, success_count = 0
        WHERE id = 'anthropic';
    END IF;
  ELSE
    IF v_row.state = 'half_open' THEN
      -- half-open test failed: reopen immediately / 半开测试失败：立即重新开启
      UPDATE insideout.ai_circuit_breaker
        SET state = 'open', opened_at = now(), last_failure_at = now(), success_count = 0
        WHERE id = 'anthropic';
    ELSIF v_row.failure_count + 1 >= 5 THEN
      UPDATE insideout.ai_circuit_breaker
        SET state = 'open', opened_at = now(), last_failure_at = now(),
            failure_count = failure_count + 1, success_count = 0
        WHERE id = 'anthropic';
    ELSE
      UPDATE insideout.ai_circuit_breaker
        SET failure_count = failure_count + 1, last_failure_at = now()
        WHERE id = 'anthropic';
    END IF;
  END IF;
END;
$$;
