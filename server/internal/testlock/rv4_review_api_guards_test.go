package testlock

import (
	"os"
	"strings"
	"testing"
)

// rvGuardsRefusal runs RequireLocalURL against a recording TB (its Fatalf ends
// the goroutine) and returns what it failed with, "" when it passed. It never
// connects, so nothing here needs a database.
func rvGuardsRefusal(t *testing.T, dsn string) string {
	t.Helper()
	rec := &recordTB{TB: t}
	done := make(chan struct{})
	go func() {
		defer close(done)
		RequireLocalURL(rec, dsn)
	}()
	<-done
	return rec.msg
}

// rv4-FR-F195: the helpers that open the store themselves ask about the DSN
// before they connect, because store.Open pings.
func TestRvGuardsRequireLocalURL(t *testing.T) {
	for _, tc := range []struct {
		name, strict, dsn string
		refuse            bool
	}{
		{"strict, a remote host", "1", remoteDSN, true},
		{"strict, a remote host in a keyword DSN", "1", "host=db.example.invalid user=app dbname=app", true},
		{"strict, a remote fallback behind a local host", "1", "postgres://app:secret@localhost:5432,db.example.invalid:5432/app", true},
		{"strict, localhost", "1", "postgres://app:secret@localhost:5432/app", false},
		{"strict, loopback v4", "1", "postgres://app:secret@127.0.0.1:55454/app", false},
		{"strict, loopback v6", "1", "postgres://app:secret@[::1]:5432/app", false},
		{"strict, a unix socket", "1", "postgres:///app?host=/var/run/postgresql", false},
		{"off, a remote host", "0", remoteDSN, false},
		{"strict, a DSN that does not parse", "1", "postgres://app:secret@db.example.invalid:notaport/app", false}, // Open reports it before it connects
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(strictEnv, tc.strict)
			msg := rvGuardsRefusal(t, tc.dsn)
			if (msg != "") != tc.refuse {
				t.Fatalf("RequireLocalURL(%s=%s) failed with %q, want refused=%v", strictEnv, tc.strict, msg, tc.refuse)
			}
			if tc.refuse && (!strings.Contains(msg, strictEnv) || strings.Contains(msg, "example.invalid")) {
				t.Fatalf("refusal = %q, want it to name %s and keep the host out of the log", msg, strictEnv)
			}
		})
	}
}

// AssertRefuses is worth its name only if it tells a test that asks the guard
// from one that does not, and a name that matches nothing from either.
func TestRvGuardsAssertRefusesTellsGuardedFromUnguarded(t *testing.T) {
	if err := refuses("TestRvGuardsProbeGuarded"); err != nil {
		t.Errorf("a test that calls RequireLocalURL was not seen to refuse: %v", err)
	}
	if err := refuses("TestRvGuardsProbeUnguarded"); err == nil {
		t.Error("a test that never asks the guard was seen to refuse")
	}
	if err := refuses("TestRvGuardsNoSuchProbe"); err == nil {
		t.Error("a test name that matches nothing was seen to refuse")
	}
}

// The two probes refuses runs in a child process; on their own they pass.
func TestRvGuardsProbeGuarded(t *testing.T) { RequireLocalURL(t, os.Getenv("DATABASE_URL")) }

func TestRvGuardsProbeUnguarded(t *testing.T) { t.Log("never asks the guard, on purpose") }
