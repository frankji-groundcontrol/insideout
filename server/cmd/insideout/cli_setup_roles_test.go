package main

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// The own-URL role gate (cli_setup_roles.go). The policy and the SQL are pure
// and tested here; the probe is held to a real catalog by checkProbeRoles; the
// composed paths run against a roleless cluster in
// cli_setup_roles_roleless_test.go (build tag roleless).

func TestDecideRoles(t *testing.T) {
	absent := rolesProbe{canTry: true}
	for _, tt := range []struct {
		name             string
		p                rolesProbe
		terminal, create bool
		want             rolesAction
	}{
		{"both present proceeds untouched", rolesProbe{owner: true, app: true}, false, false, rolesProceed},
		{"both present ignores the flag", rolesProbe{owner: true, app: true, canTry: true}, true, true, rolesProceed},
		{"both present needs no privilege", rolesProbe{owner: true, app: true}, true, false, rolesProceed},
		{"flag creates", absent, false, true, rolesCreate},
		{"terminal asks", absent, true, false, rolesAsk},
		{"flag beats the prompt", absent, true, true, rolesCreate},
		{"piped without the flag never prompts", absent, false, false, rolesStop},
		{"no privilege stops despite the flag", rolesProbe{}, true, true, rolesStop},
		{"no privilege stops at a terminal", rolesProbe{}, true, false, rolesStop},
		{"only the owner exists", rolesProbe{owner: true, canTry: true}, true, true, rolesStop},
		{"only the app role exists", rolesProbe{app: true, canTry: true}, true, true, rolesStop},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := decideRoles(tt.p, tt.terminal, tt.create); got != tt.want {
				t.Fatalf("decideRoles(%+v, terminal=%v, create=%v) = %d, want %d", tt.p, tt.terminal, tt.create, got, tt.want)
			}
		})
	}
}

// roleStatements is the bundled init script's end state in fresh-create form.
// The script is the contract: if it gains or changes an attribute, grant or
// revoke, this fails until the Go statements follow.
func TestRoleStatementsMatchInitScript(t *testing.T) {
	t.Setenv("INSIDEOUT_SETUP_REPO_ROOT", "")
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("locate the checkout holding the init script: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "docker", "postgres-init", "001-create-app-role.sh"))
	if err != nil {
		t.Fatal(err)
	}
	script := string(raw)
	stmts := roleStatements(`"$POSTGRES_DB"`, "'owner-pw'", "'app-pw'")
	if len(stmts) != 6 {
		t.Fatalf("roleStatements has %d statements, want the script's 2 roles + 4 database/schema statements", len(stmts))
	}

	// Database-level statements appear in the script verbatim.
	for _, s := range stmts[2:] {
		if !strings.Contains(script, s+";") {
			t.Errorf("statement not in the init script: %s;", s)
		}
	}
	// Role attributes: the script's ALTER ROLE line is the set CREATE ROLE must carry.
	attrs := func(s string) string {
		f := strings.Fields(s)
		sort.Strings(f)
		return strings.Join(f, " ")
	}
	for i, role := range []string{ownerRole, appRole} {
		have, _, _ := strings.Cut(strings.TrimPrefix(stmts[i], "CREATE ROLE "+role+" "), " PASSWORD ")
		var want string
		for _, line := range strings.Split(script, "\n") {
			if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "ALTER ROLE "+role+" "); ok {
				want = strings.TrimSuffix(rest, ";")
			}
		}
		if want == "" {
			t.Fatalf("no ALTER ROLE %s line in the init script", role)
		}
		if attrs(have) != attrs(want) {
			t.Errorf("%s attributes = %q, init script says %q", role, have, want)
		}
	}
}

func TestSQLLiteral(t *testing.T) {
	for in, want := range map[string]string{"abc123": "'abc123'", "it's": "'it''s'", "": "''"} {
		if got := sqlLiteral(in); got != want {
			t.Errorf("sqlLiteral(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestRoleDSNs(t *testing.T) {
	for _, tt := range []struct {
		name, in, wantApp, wantOwner string
	}{
		{
			"params survive, credentials swap",
			"postgres://postgres:oldsecret@db.example:6543/appdb?sslmode=require&pgbouncer=true",
			"postgres://insideout_app:APPPW@db.example:6543/appdb?sslmode=require&pgbouncer=true",
			"postgres://insideout_owner:OWNERPW@db.example:6543/appdb?sslmode=require&pgbouncer=true",
		},
		{
			"postgresql scheme, no password in",
			"postgresql://postgres@h/db",
			"postgresql://insideout_app:APPPW@h/db",
			"postgresql://insideout_owner:OWNERPW@h/db",
		},
		{
			"socket host in the query",
			"postgres:///db?host=/tmp/sock&port=5612",
			"postgres://insideout_app:APPPW@/db?host=/tmp/sock&port=5612",
			"postgres://insideout_owner:OWNERPW@/db?host=/tmp/sock&port=5612",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app, owner, err := roleDSNs(tt.in, "APPPW", "OWNERPW")
			if err != nil {
				t.Fatal(err)
			}
			if app != tt.wantApp || owner != tt.wantOwner {
				t.Fatalf("roleDSNs = (%s, %s), want (%s, %s)", app, owner, tt.wantApp, tt.wantOwner)
			}
			if strings.Contains(app+owner, "oldsecret") {
				t.Fatal("the supplied role's password leaked into a derived DSN")
			}
		})
	}
}

// A DSN that cannot be rewritten is refused before any SQL runs, and the
// error never echoes it (a url.Error would carry the password).
func TestRoleDSNsRefuseWhatTheyCannotRewrite(t *testing.T) {
	for _, in := range []string{
		"host=localhost user=postgres password=oldsecret dbname=x",
		"postgres://postgres:oldsecret@host:notaport/db",
		"mysql://postgres:oldsecret@host/db",
	} {
		_, _, err := roleDSNs(in, "A", "O")
		if err == nil {
			t.Errorf("roleDSNs accepted %q", in)
			continue
		}
		if strings.Contains(err.Error(), "oldsecret") {
			t.Errorf("error echoes the DSN's password: %v", err)
		}
	}
}

// What the user reads when setup stops: the cause, the way forward, and the
// SQL with placeholders only — never a password.
func TestStopMessages(t *testing.T) {
	const create = "CREATE ROLE insideout_owner LOGIN NOSUPERUSER BYPASSRLS PASSWORD '<owner-password>'"
	for _, tt := range []struct {
		name    string
		p       rolesProbe
		want    []string
		wantSQL bool
	}{
		{
			"only the owner exists",
			rolesProbe{owner: true, canTry: true, database: "appdb"},
			[]string{"only insideout_owner exists", "provision_roles.sql"},
			false,
		},
		{
			"only the app role exists",
			rolesProbe{app: true, canTry: true, database: "appdb"},
			[]string{"only insideout_app exists", "provision_roles.sql"},
			false,
		},
		{
			"no privilege",
			rolesProbe{database: "appdb"},
			[]string{"needs SUPERUSER", create, "'<app-password>'", `ALTER DATABASE "appdb" OWNER TO insideout_owner`, `GRANT CONNECT ON DATABASE "appdb" TO insideout_app`},
			true,
		},
		{
			"capable but unasked",
			rolesProbe{canTry: true, database: "appdb"},
			[]string{"--create-roles", create},
			true,
		},
		{
			"database name is quoted",
			rolesProbe{database: `we"ird db`},
			[]string{`ALTER DATABASE "we""ird db" OWNER TO insideout_owner`},
			true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			out := captureStderr(t, func() { explainMissingRoles(tt.p) })
			for _, w := range tt.want {
				if !strings.Contains(out, w) {
					t.Errorf("output lacks %q:\n%s", w, out)
				}
			}
			if got := strings.Contains(out, "CREATE ROLE"); got != tt.wantSQL {
				t.Errorf("prints SQL = %v, want %v:\n%s", got, tt.wantSQL, out)
			}
			if strings.Contains(out, "PASSWORD '") && !strings.Contains(out, "PASSWORD '<") {
				t.Errorf("a real-looking password reached the output:\n%s", out)
			}
		})
	}
}

// The terminal offer: only an explicit yes consents (EOF, enter and anything
// else decline), and the text names what will change before it asks.
func TestConfirmCreateRoles(t *testing.T) {
	old := stdin
	t.Cleanup(func() { stdin = old })
	for _, tt := range []struct {
		in   string
		want bool
	}{
		{"y\n", true}, {"Yes\n", true}, {"n\n", false}, {"\n", false}, {"", false}, {"maybe\n", false},
	} {
		stdin = bufio.NewReader(strings.NewReader(tt.in))
		var got bool
		out := captureStderr(t, func() { got = confirmCreateRoles("appdb") })
		if got != tt.want {
			t.Errorf("answer %q: confirmCreateRoles = %v, want %v", tt.in, got, tt.want)
		}
		for _, w := range []string{"insideout_owner", "insideout_app", "BYPASSRLS", `owner of database "appdb"`, "schema public", "never to the screen", "[y/N]"} {
			if !strings.Contains(out, w) {
				t.Errorf("answer %q: the offer does not mention %q:\n%s", tt.in, w, out)
			}
		}
	}
}

// --create-roles belongs to --local --postgres own, and only there. The
// checkout and .env are pinned away from the real ones: were the flag
// accepted elsewhere, the run still could not touch anything.
func TestCreateRolesFlagIsScopedToLocalOwn(t *testing.T) {
	t.Setenv("INSIDEOUT_SETUP_REPO_ROOT", t.TempDir())
	t.Setenv("INSIDEOUT_ENV_FILE", filepath.Join(t.TempDir(), "scratch.env"))
	for _, tt := range []struct {
		name string
		args []string
	}{
		{"alone", []string{"--create-roles"}},
		{"hosted", []string{"--hosted", "--create-roles"}},
		{"check", []string{"--check", "--create-roles"}},
		{"bundled", []string{"--local", "--postgres", "bundled", "--create-roles"}},
		{"own without a database to touch", []string{"--local", "--postgres", "own", "--no-start", "--create-roles"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var code int
			out := captureStderr(t, func() { _, code = runSetupCommand(tt.args) })
			if code != 2 {
				t.Fatalf("setup %v exited %d, want 2", tt.args, code)
			}
			if !strings.Contains(out, "create-roles") || strings.Contains(out, "not defined") {
				t.Fatalf("setup %v should reject the combination by name, got: %q", tt.args, out)
			}
		})
	}
}

// checkProbeRoles holds probeRoles to the catalog it reads: the connected role
// (always present) is found in whichever slot it is asked about, a name nobody
// has is not, canTry is the documented rule applied to the connected role's
// own attributes, and database is the connected one. The role names are
// parameters, so no insideout role is needed and nothing is created.
func checkProbeRoles(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	var me, db string
	var super, createRole, bypass bool
	if err := pool.QueryRow(ctx, `
		SELECT current_user, current_database(), rolsuper, rolcreaterole, rolbypassrls
		FROM pg_roles WHERE rolname = current_user`).Scan(&me, &db, &super, &createRole, &bypass); err != nil {
		t.Fatalf("read the connected role: %v", err)
	}
	const nobody = "insideout_probe_no_such_role"
	canTry := super || (createRole && bypass)
	for _, tt := range []struct {
		name       string
		owner, app string
		want       rolesProbe
	}{
		{"connected role in the owner slot", me, nobody, rolesProbe{owner: true, canTry: canTry, database: db}},
		{"connected role in the app slot", nobody, me, rolesProbe{app: true, canTry: canTry, database: db}},
		{"neither exists", nobody, nobody + "_2", rolesProbe{canTry: canTry, database: db}},
		{"both exist", me, me, rolesProbe{owner: true, app: true, canTry: canTry, database: db}},
	} {
		got, err := probeRoles(ctx, pool, tt.owner, tt.app)
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		if got != tt.want {
			t.Errorf("%s: probeRoles = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

// Same gate as the store integration tests: unset DATABASE_URL skips, and CI's
// skip gate guarantees it is set there.
func TestProbeRolesReadsTheCatalog(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(pool.Close)
	checkProbeRoles(t, pool)
}
