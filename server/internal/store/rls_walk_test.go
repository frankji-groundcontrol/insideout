package store

import (
	"context"
	"fmt"
	"sort"
	"testing"

	"github.com/jackc/pgx/v5"
)

// SEC-04 / SEC-13: the RLS boundary is checked, not remembered. TestRLSWalk
// reads every table with RLS from pg_class (never from a list), then
//
//  1. fails on any such table that is not FORCEd or not listed in rlsTables,
//     and on any listed table that lost its RLS;
//  2. seeds one workspace (an admin, a member) with a row in every table and
//     checks the member sees rows in all of them, so the zero-row checks below
//     cannot pass on an empty table;
//  3. checks a user in no workspace sees nothing, bar their own users row;
//  4. checks a session with no app.user_id, the "system context" of login,
//     registration and the stale-run reaper, sees rows only in the tables
//     rlsTables marks as system-readable;
//  5. checks no policy mentions `current_user_id() IS NULL` outside those
//     tables (the clause on an INSERT/UPDATE/DELETE policy is invisible to the
//     SELECT counts above).
//
// A new RLS table, or a new actor-less clause, is red until a reviewer decides
// which way it goes.

// rlsTables maps every RLS table to whether a session without an actor may
// read it. Only users (pre-auth login and registration, users.go) and ai_runs
// (the stale-run reaper, 20260721160000) are: each has a
// `insideout.current_user_id() IS NULL` policy clause on purpose.
var rlsTables = map[string]bool{
	"agent_conversations":   false,
	"agent_messages":        false,
	"ai_runs":               true,
	"ideas":                 false,
	"prd_commits":           false,
	"prd_revisions":         false,
	"prds":                  false,
	"project_updates":       false,
	"projects":              false,
	"proposal_decisions":    false,
	"proposal_items":        false,
	"roadmap_evidence":      false,
	"roadmap_nodes":         false,
	"users":                 true,
	"workspace_memberships": false,
	"workspaces":            false,
}

// rlsCatalog returns the RLS tables the catalog reports, reporting (not
// stopping) any that is unforced, unlisted, or listed but no longer RLS.
func rlsCatalog(t *testing.T, st *Store) []string {
	t.Helper()
	rows, err := st.Pool.Query(context.Background(), `
		SELECT c.relname, c.relforcerowsecurity
		FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'insideout' AND c.relkind IN ('r', 'p') AND c.relrowsecurity
		ORDER BY c.relname`)
	if err != nil {
		t.Fatalf("list RLS tables: %v", err)
	}
	defer rows.Close()
	var tables []string
	seen := map[string]bool{}
	for rows.Next() {
		var name string
		var forced bool
		if err := rows.Scan(&name, &forced); err != nil {
			t.Fatalf("scan RLS table: %v", err)
		}
		seen[name] = true
		if !forced {
			t.Errorf("%s has RLS enabled but not FORCEd: its owner would skip every policy", name)
		}
		if _, ok := rlsTables[name]; !ok {
			t.Errorf("%s is an RLS table the walk does not know: list it in rlsTables (system-readable or not) and seed a row for it in seedRLSWalk", name)
			continue
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("list RLS tables: %v", err)
	}
	var listed []string
	for name := range rlsTables {
		listed = append(listed, name)
	}
	sort.Strings(listed)
	for _, name := range listed {
		if !seen[name] {
			t.Errorf("rlsTables lists %s but the catalog shows no RLS on it (dropped, renamed or RLS disabled?)", name)
		}
	}
	return tables
}

// seedRLSWalk puts one row in every table of rlsTables through the store, all
// of it visible to member, and returns member and a user who shares no workspace.
func seedRLSWalk(t *testing.T, st *Store) (member, outsider *User) {
	t.Helper()
	ctx := context.Background()
	admin, member, outsider := mkUser(t, st), mkUser(t, st), mkUser(t, st)
	must := func(what string, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("seed %s: %v", what, err)
		}
	}

	ws, err := st.CreateWorkspace(ctx, admin.ID, "RLS Walk WS", "")
	must("workspace", err)
	_, err = st.JoinWorkspace(ctx, member.ID, ws.Code)
	must("membership", err)
	proj, err := st.CreateProject(ctx, member.ID, ws.ID, "Walk project", "")
	must("project", err)
	_, err = st.AddProjectUpdate(ctx, member.ID, proj.ID, "note", "walk note")
	must("project update", err)
	prop, err := st.AddProjectUpdate(ctx, member.ID, proj.ID, "agent_proposal", "walk proposal")
	must("proposal", err)
	must("proposal item", st.AddProposalItems(ctx, member.ID, prop.ID, []ProposalItem{{Action: "add_node", Title: "walk item"}}))
	_, err = st.DecideProposal(ctx, member.ID, prop.ID, "accepted", "walk")
	must("proposal decision", err)
	node, err := st.CreateRoadmapNode(ctx, member.ID, proj.ID, nil, "Walk node", "")
	must("roadmap node", err)
	must("roadmap evidence", st.AddRoadmapEvidence(ctx, member.ID, node.ID, "commit", "walk evidence", ""))
	idea, err := st.CreateIdea(ctx, member.ID, ws.ID, "Walk idea", "")
	must("idea", err)
	prd, conv, err := st.ConvertIdea(ctx, member.ID, idea.ID)
	must("prd and conversation", err)
	_, err = st.CreateRevision(ctx, member.ID, prd.ID, nil)
	must("prd revision", err)
	_, err = st.CreatePrdCommit(ctx, member.ID, prd.ID, "walk commit", "walkers", "", nil, "")
	must("prd commit", err)
	_, err = st.InsertAgentMessage(ctx, member.ID, conv.ID, "user", "walk message", nil, nil, nil)
	must("agent message", err)
	_, err = st.CreateAIRun(ctx, ws.ID, &conv.ID, member.ID, "walk prompt")
	must("ai run", err)
	return member, outsider
}

// rlsCounts counts the rows tx can see in each table.
func rlsCounts(ctx context.Context, tx pgx.Tx, tables []string) (map[string]int, error) {
	out := make(map[string]int, len(tables))
	for _, name := range tables {
		var n int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM `+pgx.Identifier{"insideout", name}.Sanitize()).Scan(&n); err != nil {
			return nil, fmt.Errorf("count %s: %w", name, err)
		}
		out[name] = n
	}
	return out, nil
}

func TestRLSWalk(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	// The walk only means something as a role RLS applies to.
	var exempt bool
	if err := st.Pool.QueryRow(ctx, `SELECT rolbypassrls OR rolsuper FROM pg_roles WHERE rolname = current_user`).Scan(&exempt); err != nil || exempt {
		t.Fatalf("DATABASE_URL must connect as a role subject to RLS (exempt=%v, err=%v)", exempt, err)
	}

	tables := rlsCatalog(t, st)
	member, outsider := seedRLSWalk(t, st)

	// A member sees rows everywhere: the positive control.
	var got map[string]int
	if err := st.withUserContext(ctx, member.ID, func(tx pgx.Tx) (err error) {
		got, err = rlsCounts(ctx, tx, tables)
		return err
	}); err != nil {
		t.Fatalf("member walk: %v", err)
	}
	for _, name := range tables {
		if got[name] < 1 {
			t.Errorf("member sees no rows in %s: seed one, or the zero-row checks on it prove nothing", name)
		}
	}

	// A user in no workspace sees nothing, except themselves.
	if err := st.withUserContext(ctx, outsider.ID, func(tx pgx.Tx) (err error) {
		got, err = rlsCounts(ctx, tx, tables)
		return err
	}); err != nil {
		t.Fatalf("outsider walk: %v", err)
	}
	for _, name := range tables {
		want := 0
		if name == "users" {
			want = 1
		}
		if got[name] != want {
			t.Errorf("outsider sees %d rows in %s, want %d", got[name], name, want)
		}
	}

	// No actor: only the deliberate system-context tables return rows.
	tx, err := st.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(ctx)
	var noActor bool
	if err := tx.QueryRow(ctx, `SELECT insideout.current_user_id() IS NULL`).Scan(&noActor); err != nil || !noActor {
		t.Fatalf("the bare transaction must have no app.user_id (noActor=%v, err=%v)", noActor, err)
	}
	if got, err = rlsCounts(ctx, tx, tables); err != nil {
		t.Fatalf("system-context walk: %v", err)
	}
	for _, name := range tables {
		if rlsTables[name] && got[name] < 1 {
			t.Errorf("system context sees no rows in %s, which is meant to be readable without an actor", name)
		}
		if !rlsTables[name] && got[name] != 0 {
			t.Errorf("system context (no app.user_id) sees %d rows in %s, want 0", got[name], name)
		}
	}

	// The clause itself, on any command, only where it is deliberate.
	rows, err := st.Pool.Query(ctx, `
		SELECT tablename, policyname, cmd FROM pg_policies
		WHERE schemaname = 'insideout'
		  AND coalesce(qual, '') || coalesce(with_check, '') ~ 'current_user_id\(\) IS NULL'
		ORDER BY tablename, policyname`)
	if err != nil {
		t.Fatalf("scan policies: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var table, policy, cmd string
		if err := rows.Scan(&table, &policy, &cmd); err != nil {
			t.Fatalf("scan policy: %v", err)
		}
		if !rlsTables[table] {
			t.Errorf("policy %s on %s (%s) trusts an unset app.user_id; only the system-readable tables in rlsTables may", policy, table, cmd)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("scan policies: %v", err)
	}
}
