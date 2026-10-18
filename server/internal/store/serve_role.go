package store

import (
	"context"
	"fmt"
	"strings"
)

// CheckServeRole refuses a request pool whose role skips row-level security
// (rv4-DATA-04). SUPERUSER and BYPASSRLS ignore every policy in db/migrations,
// FORCE ROW LEVEL SECURITY included, so a server started on such a role would
// run with tenant isolation silently off and nothing in the logs to say so.
// Only the request pool is checked: DATABASE_OWNER_URL (insideout_owner) is
// meant to bypass, and migrate and promote-admin keep using it.
//
// The error names the attribute and the variable, never the role or the DSN:
// a startup error reaches CI and hosted logs.
func (s *Store) CheckServeRole(ctx context.Context) error {
	var super, bypass bool
	if err := s.Pool.QueryRow(ctx,
		`SELECT rolsuper, rolbypassrls FROM pg_roles WHERE rolname = current_user`,
	).Scan(&super, &bypass); err != nil {
		return fmt.Errorf("inspect the role behind DATABASE_URL: %w", err)
	}
	return servePrivilegeError(super, bypass)
}

// servePrivilegeError is nil for an ordinary role and otherwise says which
// attributes make it unfit to serve, and what to do about it.
func servePrivilegeError(super, bypass bool) error {
	var attrs []string
	if super {
		attrs = append(attrs, "SUPERUSER")
	}
	if bypass {
		attrs = append(attrs, "BYPASSRLS")
	}
	if len(attrs) == 0 {
		return nil
	}
	return fmt.Errorf("the role in DATABASE_URL has %s and would skip every row-level security policy: "+
		"point DATABASE_URL at the insideout_app role and keep the privileged role in "+
		"DATABASE_OWNER_URL for migrations (docs/architecture/database-and-rls.md)", strings.Join(attrs, " and "))
}
