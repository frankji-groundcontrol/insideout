package store

import (
	"context"
	"net/url"
	"os"
	"strings"
	"testing"
)

// rv4-DATA-04: CheckServeRole against real PostgreSQL. The request-pool role
// (insideout_app) passes; the owner pool (BYPASSRLS) is refused, and the error
// carries neither its DSN, its password nor its host. SUPERUSER cannot be
// exercised on the shared database without creating a role outside the
// insideout schema, so the message builder is checked on its own.

func TestServeRoleAcceptsTheAppRole(t *testing.T) {
	st := newTestStore(t)
	if err := st.CheckServeRole(context.Background()); err != nil {
		t.Fatalf("the request-pool role (insideout_app) must be fit to serve: %v", err)
	}
}

func TestServeRoleRefusesABypassRLSPool(t *testing.T) {
	ownerURL := os.Getenv("DATABASE_OWNER_URL")
	if ownerURL == "" {
		t.Skip("DATABASE_OWNER_URL not set, skipping the BYPASSRLS refusal check")
	}
	ctx := context.Background()
	owner, err := Open(ctx, ownerURL)
	if err != nil {
		t.Fatalf("open owner store: %v", err)
	}
	t.Cleanup(owner.Close)

	err = owner.CheckServeRole(ctx)
	if err == nil {
		t.Fatal("a pool role with BYPASSRLS must be refused: it skips every policy")
	}
	msg := err.Error()
	if !strings.Contains(msg, "BYPASSRLS") || !strings.Contains(msg, "DATABASE_URL") {
		t.Errorf("the refusal must name the attribute and DATABASE_URL, got %q", msg)
	}
	u, perr := url.Parse(ownerURL)
	if perr != nil {
		t.Fatalf("parse owner url: %v", perr)
	}
	password, _ := u.User.Password()
	for _, secret := range []string{ownerURL, password, u.Host} {
		if secret != "" && strings.Contains(msg, secret) {
			t.Errorf("the refusal printed part of the DSN: %q", msg)
		}
	}
}

func TestServeRoleNamesEveryUnsafeAttribute(t *testing.T) {
	for _, c := range []struct {
		name          string
		super, bypass bool
		want, absent  []string
	}{
		{name: "ordinary role"},
		{name: "bypassrls", bypass: true, want: []string{"BYPASSRLS"}, absent: []string{"SUPERUSER"}},
		{name: "superuser", super: true, want: []string{"SUPERUSER"}, absent: []string{"BYPASSRLS"}},
		{name: "both", super: true, bypass: true, want: []string{"SUPERUSER", "BYPASSRLS"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := servePrivilegeError(c.super, c.bypass)
			if len(c.want) == 0 {
				if err != nil {
					t.Fatalf("an ordinary role must pass, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("want a refusal")
			}
			for _, w := range c.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("refusal %q does not name %s", err, w)
				}
			}
			for _, a := range c.absent {
				if strings.Contains(err.Error(), a) {
					t.Errorf("refusal %q names %s, which the role does not have", err, a)
				}
			}
		})
	}
}
