package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/testlock"
	"github.com/google/uuid"
)

// Real-Postgres integration tests for invitation-only signup, same gate
// and conventions as auth_tokens_test.go (uuid-derived hashes so reruns
// never collide).

func freshInvite(t *testing.T) string {
	t.Helper()
	return "inv-" + uuid.NewString()
}

func TestSignupModeSetting(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	// signup_mode is shared global state on a persistent database — save
	// and restore it so a test run can never leave the deployment in a
	// different mode than it started.
	prior, priorErr := st.GetSetting(ctx, "signup_mode")
	t.Cleanup(func() {
		if priorErr == nil {
			_ = st.SetSetting(ctx, "signup_mode", prior)
		}
	})

	mode, err := st.GetSetting(ctx, "signup_mode")
	if err != nil {
		t.Fatalf("get seeded signup_mode: %v", err)
	}
	if mode != `"invitation_only"` {
		t.Fatalf("seeded signup_mode = %q, want \"invitation_only\" (fail closed)", mode)
	}
	if err := st.SetSetting(ctx, "signup_mode", `"open"`); err != nil {
		t.Fatalf("set signup_mode: %v", err)
	}
	if mode, _ := st.GetSetting(ctx, "signup_mode"); mode != `"open"` {
		t.Fatalf("signup_mode after set = %q, want open", mode)
	}
	_ = mode
}

// The test that writes the global signup_mode row takes the cross-process
// lock (review-4 TEST-02): `go test ./...` runs the api suite beside this one,
// and an unlocked flip to "open" fails its invitation tests.
func TestSignupModeSettingWaitsForTheCrossProcessLock(t *testing.T) {
	st := newTestStore(t)
	testlock.AssertWaits(t, st.Pool, testlock.SignupModeLock, "TestSignupModeSetting")
}

func TestInvitationLifecycle(t *testing.T) {
	st := newTestStore(t)
	admin := mkUser(t, st)
	ctx := context.Background()

	email := "invite-" + uuid.NewString() + "@test.local"
	h := freshInvite(t)
	if _, err := st.CreateInvitation(ctx, admin.ID, email, h, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("create invitation: %v", err)
	}

	// Wrong email: not consumable.
	if err := st.ConsumeInvitation(ctx, "other-"+uuid.NewString()+"@test.local", h, admin.ID); err != ErrNotFound {
		t.Fatalf("wrong-email consume err = %v, want ErrNotFound", err)
	}
	// Right email: consumed once.
	if err := st.ConsumeInvitation(ctx, email, h, admin.ID); err != nil {
		t.Fatalf("consume invitation: %v", err)
	}
	if err := st.ConsumeInvitation(ctx, email, h, admin.ID); err != ErrNotFound {
		t.Fatalf("second consume err = %v, want ErrNotFound (single use)", err)
	}
}

func TestInvitationExpiryAndRevoke(t *testing.T) {
	st := newTestStore(t)
	admin := mkUser(t, st)
	ctx := context.Background()

	expiredEmail := "inv-" + uuid.NewString() + "@test.local"
	eh := freshInvite(t)
	if _, err := st.CreateInvitation(ctx, admin.ID, expiredEmail, eh, time.Now().Add(-time.Minute)); err != nil {
		t.Fatalf("create expired invitation: %v", err)
	}
	if err := st.ConsumeInvitation(ctx, expiredEmail, eh, admin.ID); err != ErrNotFound {
		t.Fatalf("expired consume err = %v, want ErrNotFound", err)
	}

	revokedEmail := "inv-" + uuid.NewString() + "@test.local"
	rh := freshInvite(t)
	inv, err := st.CreateInvitation(ctx, admin.ID, revokedEmail, rh, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("create revocable invitation: %v", err)
	}
	if err := st.RevokeInvitation(ctx, inv.ID); err != nil {
		t.Fatalf("revoke invitation: %v", err)
	}
	if err := st.ConsumeInvitation(ctx, revokedEmail, rh, admin.ID); err != ErrNotFound {
		t.Fatalf("revoked consume err = %v, want ErrNotFound", err)
	}
}

func TestInvitationCreateReplacesPriorUnused(t *testing.T) {
	st := newTestStore(t)
	admin := mkUser(t, st)
	ctx := context.Background()

	email := "inv-" + uuid.NewString() + "@test.local"
	oldH, newH := freshInvite(t), freshInvite(t)
	if _, err := st.CreateInvitation(ctx, admin.ID, email, oldH, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("create old invitation: %v", err)
	}
	if _, err := st.CreateInvitation(ctx, admin.ID, email, newH, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("create new invitation: %v", err)
	}
	if err := st.ConsumeInvitation(ctx, email, oldH, admin.ID); err != ErrNotFound {
		t.Fatalf("superseded consume err = %v, want ErrNotFound", err)
	}
	if err := st.ConsumeInvitation(ctx, email, newH, admin.ID); err != nil {
		t.Fatalf("replacement consume: %v", err)
	}
}

func TestSetSystemRole(t *testing.T) {
	st := newTestStore(t)
	if owner := osGetenv("DATABASE_OWNER_URL"); owner != "" {
		if err := st.AttachMigratePool(context.Background(), owner); err != nil {
			t.Fatalf("attach owner pool: %v", err)
		}
	}
	u := mkUser(t, st)

	if err := st.SetSystemRole(context.Background(), u.Email, "admin"); err != nil {
		t.Fatalf("promote: %v", err)
	}
	got, err := st.GetUserByEmail(context.Background(), u.Email)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got.SystemRole != "admin" {
		t.Fatalf("system_role = %q, want admin", got.SystemRole)
	}
}

func osGetenv(k string) string { return os.Getenv(k) }
