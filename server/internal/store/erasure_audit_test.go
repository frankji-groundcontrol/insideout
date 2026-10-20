package store

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func ownerCount(t *testing.T, st *Store, query string, args ...any) int {
	t.Helper()
	var n int
	if err := st.Pool.QueryRow(context.Background(), query, args...).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

func ownerUUID(t *testing.T, st *Store, query string, args ...any) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := st.Pool.QueryRow(context.Background(), query, args...).Scan(&id); err != nil {
		t.Fatalf("uuid: %v", err)
	}
	return id
}

// textsNaming lists schema columns whose text still contains needle. The
// audit store is the owner role, so row-level security cannot hide a hit.
func textsNaming(t *testing.T, st *Store, needle string) []string {
	t.Helper()
	ctx := context.Background()
	rows, err := st.Pool.Query(ctx, `
		SELECT table_name, column_name
		FROM information_schema.columns
		WHERE table_schema = 'insideout'
		  AND table_name <> 'schema_migrations'
		  AND (data_type IN ('text', 'character varying', 'jsonb', 'json') OR udt_name IN ('_text', '_varchar'))
		ORDER BY table_name, column_name`)
	if err != nil {
		t.Fatalf("list columns: %v", err)
	}
	defer rows.Close()
	type col struct{ table, name string }
	var cols []col
	for rows.Next() {
		var c col
		if err := rows.Scan(&c.table, &c.name); err != nil {
			t.Fatalf("scan column: %v", err)
		}
		cols = append(cols, c)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("list columns: %v", err)
	}
	var hits []string
	for _, c := range cols {
		q := `SELECT count(*) FROM ` + pgx.Identifier{"insideout", c.table}.Sanitize() +
			` WHERE ` + pgx.Identifier{c.name}.Sanitize() + `::text LIKE '%' || $1 || '%'`
		var n int
		if err := st.Pool.QueryRow(ctx, q, needle).Scan(&n); err != nil {
			t.Fatalf("scan %s.%s: %v", c.table, c.name, err)
		}
		if n > 0 {
			hits = append(hits, c.table+"."+c.name)
		}
	}
	return hits
}
