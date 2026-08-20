package store

import (
	"context"
	"testing"
)

func TestRoles_OwnerIsNotSuperuserAndAppIsNotTableOwner(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	var current string
	if err := st.Pool.QueryRow(ctx, `SELECT current_user`).Scan(&current); err != nil {
		t.Fatal(err)
	}
	if current != "insideout_app" {
		t.Fatalf("runtime pool user = %q, want insideout_app", current)
	}

	var ownerSuper bool
	if err := st.Pool.QueryRow(ctx, `SELECT rolsuper FROM pg_roles WHERE rolname = 'insideout_owner'`).Scan(&ownerSuper); err != nil {
		t.Fatalf("insideout_owner missing: %v", err)
	}
	if ownerSuper {
		t.Fatal("insideout_owner must not be a superuser")
	}
	var ownerBypass bool
	if err := st.Pool.QueryRow(ctx, `SELECT rolbypassrls FROM pg_roles WHERE rolname = 'insideout_owner'`).Scan(&ownerBypass); err != nil {
		t.Fatal(err)
	}
	if !ownerBypass {
		t.Fatal("insideout_owner needs BYPASSRLS so DEFINER helpers are not superuser-owned and do not recurse under FORCE RLS")
	}

	var tableOwner, fnOwner string
	if err := st.Pool.QueryRow(ctx, `
		SELECT pg_catalog.pg_get_userbyid(c.relowner)
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'insideout' AND c.relname = 'workspace_memberships'
	`).Scan(&tableOwner); err != nil {
		t.Fatal(err)
	}
	if tableOwner != "insideout_owner" {
		t.Fatalf("workspace_memberships owner = %q, want insideout_owner", tableOwner)
	}
	if err := st.Pool.QueryRow(ctx, `
		SELECT pg_catalog.pg_get_userbyid(p.proowner)
		FROM pg_proc p
		JOIN pg_namespace n ON n.oid = p.pronamespace
		WHERE n.nspname = 'insideout' AND p.proname = '_is_member'
	`).Scan(&fnOwner); err != nil {
		t.Fatal(err)
	}
	if fnOwner != "insideout_owner" {
		t.Fatalf("_is_member owner = %q, want insideout_owner (not a superuser, not insideout_app)", fnOwner)
	}

	var forced bool
	if err := st.Pool.QueryRow(ctx, `
		SELECT c.relforcerowsecurity
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'insideout' AND c.relname = 'workspace_memberships'
	`).Scan(&forced); err != nil {
		t.Fatal(err)
	}
	if !forced {
		t.Fatal("workspace_memberships should FORCE ROW LEVEL SECURITY now that DEFINER runs as owner")
	}
}
