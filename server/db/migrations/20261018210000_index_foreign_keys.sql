-- rv4-DATA-07: PostgreSQL indexes the referenced side of a foreign key (a
-- primary key or unique constraint) and never the referencing side. Without an
-- index on the child columns, deleting or updating a parent row scans the whole
-- child table, once per parent row a cascade removes (workspace -> ai_runs and
-- agent_conversations, project update -> proposal_items, node -> child nodes,
-- user -> every created_by / author_id column).
--
-- This indexes every foreign key of schema insideout that has no supporting
-- index, found in the catalog rather than listed by hand, so it also covers a
-- key an earlier migration added without one. Supporting means a valid,
-- non-partial index whose first columns are the key columns in either order.
-- Names are table_column[_column]_idx, shortened with a hash past the 63-byte
-- limit. It adds only non-unique indexes: no data can make it fail, it adds no
-- constraint, and CREATE INDEX IF NOT EXISTS makes a second run find nothing to
-- do. CREATE INDEX holds a SHARE lock on its table while it builds, which these
-- tables make a matter of milliseconds.
DO $$
DECLARE
  fk record;
  idx text;
BEGIN
  FOR fk IN
    SELECT cl.relname::text AS tbl,
           (SELECT array_agg(a.attname::text ORDER BY k.ord)
              FROM unnest(c.conkey) WITH ORDINALITY k(attnum, ord)
              JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = k.attnum) AS cols
      FROM pg_constraint c
      JOIN pg_class cl ON cl.oid = c.conrelid
      JOIN pg_namespace n ON n.oid = cl.relnamespace
     WHERE c.contype = 'f' AND n.nspname = 'insideout'
       AND NOT EXISTS (
         SELECT 1 FROM pg_index i
          WHERE i.indrelid = c.conrelid AND i.indisvalid AND i.indpred IS NULL
            AND (SELECT array_agg(x ORDER BY x)
                   FROM (SELECT x FROM unnest(i.indkey::int2[]) WITH ORDINALITY u(x, ord)
                          ORDER BY ord LIMIT cardinality(c.conkey)) q)
              = (SELECT array_agg(x ORDER BY x) FROM unnest(c.conkey) x))
     ORDER BY cl.relname, c.conname
  LOOP
    idx := fk.tbl || '_' || array_to_string(fk.cols, '_') || '_idx';
    IF length(idx) > 63 THEN
      idx := left(idx, 50) || '_' || substr(md5(idx), 1, 8) || '_idx';
    END IF;
    EXECUTE format(
      'CREATE INDEX IF NOT EXISTS %I ON insideout.%I (%s)',
      idx, fk.tbl,
      (SELECT string_agg(format('%I', col), ', ' ORDER BY ord) FROM unnest(fk.cols) WITH ORDINALITY u(col, ord)));
  END LOOP;
END
$$;
