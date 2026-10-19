package store

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// rv4-DATA-07: PostgreSQL indexes the referenced side of a foreign key (it is
// a primary key or a unique constraint) and never the referencing side. With
// no index on the child columns, deleting or updating a parent row scans the
// whole child table, once per parent row a cascade removes (workspace ->
// ai_runs, project update -> proposal_items, node -> child nodes). Real
// PostgreSQL, like every store-backed test: the schema is read from the
// catalog after migrate, not from the migration files.

// fkWithoutIndexSQL lists the foreign keys of schema insideout whose
// referencing columns are not the leading columns of a valid, non-partial
// index on the same table. The first len(conkey) index columns must be the key
// columns in any order: equality on all of them is served by either order.
const fkWithoutIndexSQL = `
SELECT cl.relname, c.conname,
       (SELECT array_agg(a.attname::text ORDER BY k.ord)
          FROM unnest(c.conkey) WITH ORDINALITY k(attnum, ord)
          JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = k.attnum)
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
 ORDER BY cl.relname, c.conname`

// noIndexNeeded names the foreign keys allowed to go unindexed, keyed
// "table.constraint", each with the reason it never needs one (for example a
// table that stays a handful of rows). It is empty: every foreign key of the
// schema is indexed, so a new one that is not fails this test instead of
// quietly turning a later delete into a table scan.
var noIndexNeeded = map[string]string{}

func TestEveryForeignKeyHasASupportingIndex(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	var total int
	if err := st.Pool.QueryRow(ctx, `SELECT count(*) FROM pg_constraint c JOIN pg_namespace n ON n.oid = c.connamespace
		WHERE c.contype = 'f' AND n.nspname = 'insideout'`).Scan(&total); err != nil {
		t.Fatalf("count foreign keys: %v", err)
	}
	if total == 0 {
		t.Fatal("no foreign keys in schema insideout: the check below would pass without looking at anything")
	}
	for key, why := range noIndexNeeded {
		if strings.TrimSpace(why) == "" {
			t.Errorf("exception %s has no reason", key)
		}
	}

	rows, err := st.Pool.Query(ctx, fkWithoutIndexSQL)
	if err != nil {
		t.Fatalf("list unindexed foreign keys: %v", err)
	}
	defer rows.Close()
	var missing []string
	for rows.Next() {
		var table, constraint string
		var cols []string
		if err := rows.Scan(&table, &constraint, &cols); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if _, exempt := noIndexNeeded[table+"."+constraint]; !exempt {
			missing = append(missing, fmt.Sprintf("%s.%s (%s)", table, constraint, strings.Join(cols, ", ")))
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("read unindexed foreign keys: %v", err)
	}
	if len(missing) > 0 {
		t.Fatalf("%d of %d foreign keys in schema insideout have no supporting index (apply the migrations, or index the new key):\n  %s",
			len(missing), total, strings.Join(missing, "\n  "))
	}
}

// The four columns the review named, looked up by column rather than by
// constraint name so a renamed constraint cannot hide them.
func TestTheForeignKeyColumnsTheReviewNamedAreIndexed(t *testing.T) {
	st := newTestStore(t)
	for _, c := range []struct{ table, column string }{
		{"proposal_items", "update_id"},
		{"ai_runs", "workspace_id"},
		{"agent_conversations", "workspace_id"},
		{"roadmap_nodes", "parent_id"},
	} {
		var indexed bool
		err := st.Pool.QueryRow(context.Background(), `SELECT EXISTS (
			SELECT 1 FROM pg_index i
			  JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = i.indkey[0]
			 WHERE i.indrelid = ('insideout.' || $1::text)::regclass
			   AND i.indisvalid AND i.indpred IS NULL AND a.attname = $2)`, c.table, c.column).Scan(&indexed)
		if err != nil {
			t.Fatalf("look up an index on %s.%s: %v", c.table, c.column, err)
		}
		if !indexed {
			t.Errorf("no index leads with %s.%s", c.table, c.column)
		}
	}
}
