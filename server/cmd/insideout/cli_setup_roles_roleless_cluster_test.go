//go:build roleless

package main

// The roleless suite's harness (scenarios: cli_setup_roles_roleless_test.go):
// a handle on a throwaway superuser-accessible cluster that has no insideout
// roles, plus a driver that runs `setup --local --postgres own` against it
// with a piped DSN.

import (
	"bytes"
	"context"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const rolelessDefaultURL = "postgres://postgres@127.0.0.1:5612/postgres?sslmode=disable"

type rlCluster struct {
	admin *pgxpool.Pool
	base  *url.URL
}

func rlSuffix() string { return strings.ReplaceAll(uuid.NewString(), "-", "")[:10] }

func newRolelessCluster(t *testing.T) *rlCluster {
	t.Helper()
	raw := os.Getenv("INSIDEOUT_ROLELESS_URL")
	if raw == "" {
		raw = rolelessDefaultURL
	}
	base, err := url.Parse(raw)
	if err != nil {
		t.Fatal("INSIDEOUT_ROLELESS_URL is not a URL") // never echo it: it may carry a password
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, raw)
	if err != nil {
		t.Fatalf("open roleless cluster: %v", err)
	}
	t.Cleanup(admin.Close) // registered first, so it runs after every other cleanup
	c := &rlCluster{admin: admin, base: base}

	var super bool
	if err := admin.QueryRow(ctx, `SELECT rolsuper FROM pg_roles WHERE rolname = current_user`).Scan(&super); err != nil || !super {
		t.Fatalf("INSIDEOUT_ROLELESS_URL must be a superuser connection (err=%v)", err)
	}
	var have int
	if err := admin.QueryRow(ctx, `SELECT count(*) FROM pg_roles WHERE rolname IN ('insideout_owner','insideout_app')`).Scan(&have); err != nil {
		t.Fatal(err)
	}
	if have != 0 {
		t.Fatal("cluster already has the insideout roles: point INSIDEOUT_ROLELESS_URL at a throwaway cluster")
	}
	t.Cleanup(func() {
		for _, r := range []string{"insideout_app", "insideout_owner"} {
			admin.Exec(ctx, `DROP ROLE IF EXISTS `+r)
		}
	})
	return c
}

// dsn builds a DSN for db; user "" keeps the superuser credentials.
func (c *rlCluster) dsn(db, user, password string) string {
	u := *c.base
	u.Path = "/" + db
	if user != "" {
		u.User = url.UserPassword(user, password)
	}
	return u.String()
}

func (c *rlCluster) exec(t *testing.T, sql string) {
	t.Helper()
	if _, err := c.admin.Exec(context.Background(), sql); err != nil {
		t.Fatalf("%s: %v", strings.SplitN(sql, " ", 3)[0], err)
	}
}

// newDatabase creates a throwaway database (owned by owner when set).
func (c *rlCluster) newDatabase(t *testing.T, owner string) string {
	t.Helper()
	name := "io_rl_" + rlSuffix()
	sql := `CREATE DATABASE ` + name
	if owner != "" {
		sql += ` OWNER ` + owner
	}
	c.exec(t, sql)
	t.Cleanup(func() {
		c.admin.Exec(context.Background(), `DROP DATABASE IF EXISTS `+name+` WITH (FORCE)`)
	})
	return name
}

// newLoginRole creates a throwaway login role with the given extra attributes.
func (c *rlCluster) newLoginRole(t *testing.T, attrs string) (name, password string) {
	t.Helper()
	name, password = "io_rl_"+rlSuffix(), "pw"+rlSuffix()
	c.exec(t, `CREATE ROLE `+name+` LOGIN `+attrs+` PASSWORD '`+password+`'`)
	t.Cleanup(func() { c.admin.Exec(context.Background(), `DROP ROLE IF EXISTS `+name) })
	return name, password
}

func (c *rlCluster) roleExists(t *testing.T, name string) bool {
	t.Helper()
	var ok bool
	if err := c.admin.QueryRow(context.Background(), `SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)`, name).Scan(&ok); err != nil {
		t.Fatal(err)
	}
	return ok
}

func (c *rlCluster) schemaExists(t *testing.T, db string) bool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), c.dsn(db, "", ""))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var ok bool
	if err := pool.QueryRow(context.Background(), `SELECT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'insideout')`).Scan(&ok); err != nil {
		t.Fatal(err)
	}
	return ok
}

// operatorProvision applies, as an operator would by hand, the contract of
// docker/postgres-init/001-create-app-role.sh to db. Deliberately a second,
// independent statement of that contract rather than a call into the code
// under test.
func (c *rlCluster) operatorProvision(t *testing.T, db, ownerPW, appPW string) {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), c.dsn(db, "", ""))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	for _, s := range []string{
		`CREATE ROLE insideout_owner LOGIN NOSUPERUSER BYPASSRLS PASSWORD '` + ownerPW + `'`,
		`CREATE ROLE insideout_app LOGIN NOSUPERUSER NOBYPASSRLS PASSWORD '` + appPW + `'`,
		`ALTER DATABASE ` + db + ` OWNER TO insideout_owner`,
		`GRANT CONNECT ON DATABASE ` + db + ` TO insideout_app`,
		`REVOKE CREATE ON SCHEMA public FROM PUBLIC`,
		`REVOKE ALL ON SCHEMA public FROM insideout_app`,
	} {
		if _, err := pool.Exec(context.Background(), s); err != nil {
			t.Fatalf("operator provisioning %q: %v", strings.SplitN(s, " ", 3)[0], err)
		}
	}
}

// rlRun is one fake checkout + .env that successive setup runs share.
type rlRun struct{ env string }

func newRlRun(t *testing.T) *rlRun {
	t.Helper()
	repo := t.TempDir()
	skeleton := "DATABASE_URL=change_me_db\nINSIDEOUT_JWT_SECRET=change_me_to_a_long_random_string\n#DATABASE_OWNER_URL=\n#POSTGRES_OWNER_PASSWORD=\n#POSTGRES_APP_PASSWORD=\n"
	for _, name := range []string{"docker-compose.yml", ".env.example"} {
		if err := os.WriteFile(filepath.Join(repo, name), []byte(skeleton), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(repo, "server"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "server", "go.mod"), []byte("module stub\n"), 0644); err != nil {
		t.Fatal(err)
	}
	env := filepath.Join(repo, "scratch.env")
	t.Setenv("INSIDEOUT_SETUP_REPO_ROOT", repo)
	t.Setenv("INSIDEOUT_ENV_FILE", env)
	return &rlRun{env: env}
}

// run drives `setup --local --postgres own <args>` with dsn piped on stdin
// (the agent path: stdin is not a terminal, so any prompt would hang) and
// returns the exit code plus everything written to stdout and stderr.
func (r *rlRun) run(t *testing.T, dsn string, args ...string) (int, string) {
	t.Helper()
	in, inW, _ := os.Pipe()
	inW.WriteString(dsn + "\n")
	inW.Close()
	outR, outW, _ := os.Pipe()
	oldIn, oldOut, oldErr := os.Stdin, os.Stdout, os.Stderr
	os.Stdin, os.Stdout, os.Stderr = in, outW, outW
	var buf bytes.Buffer
	copied := make(chan struct{})
	go func() { io.Copy(&buf, outR); close(copied) }()
	done := make(chan int, 1)
	go func() {
		_, code := runSetupCommand(append([]string{"--local", "--postgres", "own"}, args...))
		done <- code
	}()
	code := -1
	select {
	case code = <-done:
	case <-time.After(120 * time.Second):
		os.Stdin, os.Stdout, os.Stderr = oldIn, oldOut, oldErr
		t.Fatal("setup did not return in 120s (blocked on a prompt on a piped stdin?)")
	}
	outW.Close()
	<-copied
	os.Stdin, os.Stdout, os.Stderr = oldIn, oldOut, oldErr
	return code, buf.String()
}

func rlNoSecrets(t *testing.T, out string, secrets ...string) {
	t.Helper()
	for _, s := range secrets {
		if s != "" && strings.Contains(out, s) {
			t.Fatalf("setup output leaked a secret value (%d chars):\n%s", len(s), out)
		}
	}
}
