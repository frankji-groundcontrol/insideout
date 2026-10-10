package main

// The own-URL precondition (docs/issues/2026-10-07-own-url-fresh-db-bootstrap.md):
// migrations need the two NOSUPERUSER roles that
// docker/postgres-init/001-create-app-role.sh provisions for the bundled
// database, and a fresh own-URL database does not have them. Before the first
// migration setup probes the target through pg_roles, then proceeds, provisions
// with the user's consent, or stops with the SQL to run. No DSN or password is
// ever printed: generated passwords exist only in the CREATE ROLE statement
// sent to the server and in .env.

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/frankji-groundcontrol/insideout/server/internal/setup/catalog"
	"github.com/frankji-groundcontrol/insideout/server/internal/setup/envwrite"
)

const (
	ownerRole = "insideout_owner"
	appRole   = "insideout_app"
)

// rolesProbe is what one catalog round trip tells the decision.
type rolesProbe struct {
	owner, app bool // the role exists
	// canTry: the connecting role may be able to create them — a superuser, or
	// CREATEROLE with BYPASSRLS (only a BYPASSRLS role may hand it out). A guess
	// that only spares an ordinary role a pointless prompt; the attempt itself
	// stays authoritative (see provisionRoles).
	canTry   bool
	database string // current_database(): what the provisioning SQL targets

	// What the migration needs of this database, as demonstrated facts (the
	// zero value is "no problem known"). Its one database-level need is CREATE
	// for insideout_owner, to create the insideout schema, and only while that
	// is missing — owning the database or a grant provide it. A database
	// somebody else owns migrates fine once an admin pre-created the schema (the
	// shared-instance layout), so ownership alone is not the test.
	schemaMissing   bool   // no insideout schema yet
	ownerCantCreate bool   // insideout_owner exists and lacks CREATE on this database
	dbOwner         string // who owns this database, for the message
}

func probeRoles(ctx context.Context, pool *pgxpool.Pool, owner, app string) (rolesProbe, error) {
	var p rolesProbe
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1),
		       EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $2),
		       r.rolsuper OR (r.rolcreaterole AND r.rolbypassrls),
		       current_database(),
		       NOT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'insideout'),
		       COALESCE((SELECT NOT has_database_privilege(o.rolname, current_database(), 'CREATE')
		                 FROM pg_roles o WHERE o.rolname = $1), false),
		       pg_get_userbyid(d.datdba)
		FROM pg_roles r, pg_database d
		WHERE r.rolname = current_user AND d.datname = current_database()`, owner, app).
		Scan(&p.owner, &p.app, &p.canTry, &p.database, &p.schemaMissing, &p.ownerCantCreate, &p.dbOwner)
	return p, err
}

type rolesAction int

const (
	rolesProceed  rolesAction = iota // both present: migrate exactly as before
	rolesCreate                      // provision without asking (--create-roles)
	rolesAsk                         // provision if the user confirms (terminal)
	rolesStop                        // print the SQL and stop before migrating
	rolesNotReady                    // both present, but the migration cannot create the schema here
)

// decideRoles is the whole policy. A piped run never prompts (its stdin is the
// DSN) and never provisions unasked: creating cluster-wide roles and changing
// the database owner needs an explicit yes, from a prompt or --create-roles.
// A lone existing role stops too: its password is unknown, so setup could not
// wire it. With both present it proceeds unless the migration would certainly
// die on this database — schema missing and insideout_owner unable to create
// it — and then it prints the fix and stops: changing a database's owner is
// the operator's call, not a side effect of --create-roles.
func decideRoles(p rolesProbe, terminal, create bool) rolesAction {
	switch {
	case p.owner && p.app && p.schemaMissing && p.ownerCantCreate:
		return rolesNotReady
	case p.owner && p.app:
		return rolesProceed
	case p.owner || p.app || !p.canTry:
		return rolesStop
	case create:
		return rolesCreate
	case terminal:
		return rolesAsk
	}
	return rolesStop
}

// ensureRoles gates the own-URL path before its first migration. It returns
// the owner DSN Migrate should attach (the one just provisioned, else the one
// .env already had) and false when setup must stop; every stop precedes any
// migration and any write of the supplied DSN as DATABASE_URL, which happens
// here: the supplied DSN when the roles exist, the app URL when provisioned.
func ensureRoles(ctx context.Context, pool *pgxpool.Pool, dsn, envPath, ownerURL string, create bool) (string, bool) {
	p, err := probeRoles(ctx, pool, ownerRole, appRole)
	if err != nil {
		fmt.Fprintf(os.Stderr, "setup: probe roles: %v\n", err)
		return "", false
	}
	act := decideRoles(p, isTerminal(os.Stdin), create)
	if act == rolesAsk && !confirmCreateRoles(p.database) {
		act = rolesStop
	}
	switch act {
	case rolesProceed:
		fmt.Fprintf(os.Stderr, "setup: roles %s and %s present\n", ownerRole, appRole)
		if !wireDatabaseURL(envPath, dsn) {
			return "", false
		}
		return ownerURL, true
	case rolesStop:
		explainMissingRoles(p)
		return "", false
	case rolesNotReady:
		explainNotReady(p)
		return "", false
	}
	appURL, newOwnerURL, err := provisionRoles(ctx, pool, dsn, envPath, p.database)
	if errors.Is(err, errNotPermitted) {
		fmt.Fprintf(os.Stderr, "setup: %v; nothing was changed\n", err)
		printRoleSQL(p.database)
		return "", false
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "setup: create roles: %v\n", err)
		return "", false
	}
	fmt.Fprintf(os.Stderr, "setup: created %s and %s; generated passwords stored in .env (not printed)\n", ownerRole, appRole)
	fmt.Fprintf(os.Stderr, "setup: wrote DATABASE_URL → %s (%s)\n", catalog.RedactDSN(appURL), appRole)
	fmt.Fprintf(os.Stderr, "setup: wrote DATABASE_OWNER_URL → %s (%s)\n", catalog.RedactDSN(newOwnerURL), ownerRole)
	return newOwnerURL, true
}

// confirmCreateRoles asks at a terminal, saying what will change: on a shared
// database the owner switch and the schema-public revoke are not invisible.
func confirmCreateRoles(database string) bool {
	fmt.Fprintf(os.Stderr, "setup: roles %s and %s do not exist on this server. Creating them:\n", ownerRole, appRole)
	fmt.Fprintf(os.Stderr, "  %s: NOSUPERUSER, BYPASSRLS, becomes the owner of database %q\n", ownerRole, database)
	fmt.Fprintf(os.Stderr, "  %s: NOSUPERUSER, CONNECT only; CREATE on schema public is revoked from PUBLIC\n", appRole)
	fmt.Fprintln(os.Stderr, "  generated passwords go to .env, never to the screen")
	return strings.HasPrefix(strings.ToLower(prompt("Create them now? [y/N] ")), "y")
}

// explainMissingRoles says why setup stopped and, whenever the generic SQL
// helps, prints it.
func explainMissingRoles(p rolesProbe) {
	switch {
	case p.owner || p.app:
		have := ownerRole
		if p.app {
			have = appRole
		}
		fmt.Fprintf(os.Stderr, "setup: only %s exists. Setup will not guess its password; create the missing role with\n", have)
		fmt.Fprintln(os.Stderr, "  server/db/provision_roles.sql (run as a superuser; it is idempotent), then re-run setup")
		return
	case !p.canTry:
		fmt.Fprintf(os.Stderr, "setup: roles %s and %s are missing and this connection's role cannot create them (needs SUPERUSER).\n", ownerRole, appRole)
	default:
		fmt.Fprintf(os.Stderr, "setup: roles %s and %s are missing. Setup creates them only on an explicit yes — the prompt at a\n", ownerRole, appRole)
		fmt.Fprintln(os.Stderr, "  terminal, or --create-roles (passwords are generated into .env) — or run the SQL below yourself.")
	}
	printRoleSQL(p.database)
}

func printRoleSQL(database string) {
	fmt.Fprintf(os.Stderr, "\nRun this as a superuser on database %q (choose both passwords yourself). Then put the %s\n", database, ownerRole)
	fmt.Fprintf(os.Stderr, "connection string in DATABASE_OWNER_URL in .env and re-run setup with the %s one:\n\n", appRole)
	for _, s := range roleStatements(pgx.Identifier{database}.Sanitize(), "'<owner-password>'", "'<app-password>'") {
		fmt.Fprintf(os.Stderr, "  %s;\n", s)
	}
}

// explainNotReady: the roles exist but this database is not theirs yet (a
// further database on a provisioned cluster). Both fixes change someone's
// database — its owner, or its schemas — so setup prints them and stops; the
// second is the documented shared-instance layout, for a database that must
// keep its owner.
func explainNotReady(p rolesProbe) {
	fmt.Fprintf(os.Stderr, "setup: roles %s and %s exist, but database %q is owned by %q,\n", ownerRole, appRole, p.database, p.dbOwner)
	fmt.Fprintf(os.Stderr, "  and %s cannot create the insideout schema in it: the migrations would fail with\n", ownerRole)
	fmt.Fprintln(os.Stderr, "  \"permission denied for database\". Nothing in the database was changed.")
	fmt.Fprintf(os.Stderr, "\nRun ONE of these on database %q as a superuser (or as its owner, if that role is a\n", p.database)
	fmt.Fprintf(os.Stderr, "member of %s), then re-run setup. To hand the database to the roles, as the bundled one is:\n\n", ownerRole)
	for _, s := range databaseStatements(pgx.Identifier{p.database}.Sanitize()) {
		fmt.Fprintf(os.Stderr, "  %s;\n", s)
	}
	fmt.Fprintln(os.Stderr, "\nOr, if the database must keep its owner (a shared instance), pre-create the schema instead:")
	fmt.Fprintf(os.Stderr, "\n  CREATE SCHEMA insideout AUTHORIZATION %s;\n", ownerRole)
}

// roleStatements is 001-create-app-role.sh's end state in fresh-create form:
// no NOT EXISTS guards and no trailing ALTER ROLE, which only matter when a
// role may already exist (here both are known absent). Plain SQL, so it pastes
// into any console — the script's \gexec is psql-only. db is a quoted
// identifier; ownerPW and appPW are SQL string literals (placeholders when
// printed).
func roleStatements(db, ownerPW, appPW string) []string {
	return append([]string{
		"CREATE ROLE " + ownerRole + " LOGIN NOSUPERUSER BYPASSRLS PASSWORD " + ownerPW,
		"CREATE ROLE " + appRole + " LOGIN NOSUPERUSER NOBYPASSRLS PASSWORD " + appPW,
	}, databaseStatements(db)...)
}

// databaseStatements is the per-database half of that contract: what a further
// database on an already-provisioned cluster still lacks. db is a quoted
// identifier.
func databaseStatements(db string) []string {
	return []string{
		"ALTER DATABASE " + db + " OWNER TO " + ownerRole,
		"GRANT CONNECT ON DATABASE " + db + " TO " + appRole,
		"REVOKE CREATE ON SCHEMA public FROM PUBLIC",
		"REVOKE ALL ON SCHEMA public FROM " + appRole,
	}
}

// sqlLiteral quotes a generated (hex) password; doubling the quote is the whole
// rule under the default standard_conforming_strings.
func sqlLiteral(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

// roleDSNs rebuilds the supplied DSN for each role: same host, database and
// parameters (sslmode, pgbouncer), new credentials. ponytail: postgres://
// URLs only — a keyword/value DSN stops here, before any SQL runs; add a
// rewriter if one ever turns up.
func roleDSNs(dsn, appPW, ownerPW string) (appURL, ownerURL string, err error) {
	u, perr := url.Parse(dsn)
	if perr != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		// Never echo perr: a url.Error carries the whole DSN, password included.
		return "", "", errors.New("the connection string is not a postgres:// URL, so the role connection strings cannot be derived from it")
	}
	with := func(role, pw string) string {
		c := *u
		c.User = url.UserPassword(role, pw)
		return c.String()
	}
	return with(appRole, appPW), with(ownerRole, ownerPW), nil
}

// errNotPermitted marks a provisioning attempt PostgreSQL refused (SQLSTATE
// 42501) — the only failure that is answered with the SQL to run by hand.
var errNotPermitted = errors.New("this connection's role is not permitted to create the roles")

// provisionRoles creates both roles in one transaction on the supplied
// connection, so a refusal at any statement leaves the database untouched.
// The fresh passwords reach .env (with the app and owner DSNs) before the
// commit: a crash in between leaves unused credentials the next run
// overwrites, never roles whose passwords were lost.
func provisionRoles(ctx context.Context, pool *pgxpool.Pool, dsn, envPath, database string) (appURL, ownerURL string, err error) {
	ownerPW, err := generateJWTSecret() // 256 random bits as hex; the name is historical
	if err != nil {
		return "", "", err
	}
	appPW, err := generateJWTSecret()
	if err != nil {
		return "", "", err
	}
	scrub := strings.NewReplacer(ownerPW, "<password>", appPW, "<password>").Replace
	fail := func(err error) (string, string, error) { return "", "", errors.New(scrub(err.Error())) }

	// Before any SQL: a DSN that cannot be rewritten must not leave roles that
	// cannot be wired.
	if appURL, ownerURL, err = roleDSNs(dsn, appPW, ownerPW); err != nil {
		return fail(err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fail(err)
	}
	defer tx.Rollback(ctx)
	for _, s := range roleStatements(pgx.Identifier{database}.Sanitize(), sqlLiteral(ownerPW), sqlLiteral(appPW)) {
		if _, err := tx.Exec(ctx, s); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "42501" {
				return "", "", fmt.Errorf("%w (%s)", errNotPermitted, scrub(pgErr.Message))
			}
			return fail(err)
		}
	}
	for _, kv := range [][2]string{
		{"POSTGRES_OWNER_PASSWORD", ownerPW},
		{"POSTGRES_APP_PASSWORD", appPW},
		{"DATABASE_OWNER_URL", ownerURL},
		{"DATABASE_URL", appURL},
	} {
		if err := envwrite.SetVar(envPath, kv[0], kv[1]); err != nil {
			return fail(fmt.Errorf("write %s: %w", kv[0], err))
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fail(err)
	}
	return appURL, ownerURL, nil
}
