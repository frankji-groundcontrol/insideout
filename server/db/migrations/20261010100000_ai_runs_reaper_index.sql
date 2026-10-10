-- The stale-run reaper (Store.ReapStaleAIRuns, run on a ticker) filters
-- ai_runs on status IN ('pending', 'running') plus an updated_at cutoff, and
-- until now had no index to do it with. ai_runs gains a row per agent message
-- and nearly all of them are finished, so a partial index over only the
-- in-flight rows stays tiny and replaces the sequential scan. The predicate
-- must match the reaper's IN list for the planner to use it.
CREATE INDEX IF NOT EXISTS ai_runs_reaper_idx
  ON insideout.ai_runs (updated_at)
  WHERE status IN ('pending', 'running');
