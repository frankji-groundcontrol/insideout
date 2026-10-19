package api

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/testlock"
	"github.com/google/uuid"
)

// TEST-13: /api/v1/admin/* is gated on users.system_role, not on anything a
// workspace grants. DB-gated: the role is read from real PostgreSQL on every
// request, and promoting an admin takes the owner pool (RLS blocks the app
// role from touching another user's row), so DATABASE_OWNER_URL is needed too.
func newAdminWorld(t *testing.T) (w *routeWorld, admin routeUser) {
	t.Helper()
	w = newRouteWorld(t)
	ownerURL := os.Getenv("DATABASE_OWNER_URL")
	if ownerURL == "" {
		t.Skip("DATABASE_OWNER_URL not set, skipping: promoting an admin needs the owner role")
	}
	ctx := context.Background()
	if err := w.st.AttachMigratePool(ctx, ownerURL); err != nil {
		t.Fatalf("attach owner pool: %v", err)
	}
	if err := w.st.SetSystemRole(ctx, w.member.Email, "admin"); err != nil {
		t.Fatalf("promote: %v", err)
	}
	return w, w.member
}

func TestAdminRoutesRefuseEveryoneButSystemAdmins(t *testing.T) {
	w, _ := newAdminWorld(t)
	// The workspace's own admin is still not a system admin. The write bodies
	// are ones the handlers would reject (bad mode) or that are harmless, so a
	// missing guard shows up as the wrong status without changing anything.
	routes := []struct{ method, path, body string }{
		{http.MethodGet, "/api/v1/admin/settings", ""},
		{http.MethodPut, "/api/v1/admin/settings", `{"signupMode":"sideways"}`},
		{http.MethodPost, "/api/v1/admin/invitations", `{"email":"refused-` + uuid.NewString() + `@test.local"}`},
		{http.MethodGet, "/api/v1/admin/invitations", ""},
		{http.MethodPost, "/api/v1/admin/invitations/" + uuid.NewString() + "/revoke", ""},
	}
	for _, rt := range routes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			for who, user := range map[string]routeUser{"workspace owner": w.owner, "outsider": w.outsider} {
				status, body := w.call(user, rt.method, rt.path, rt.body)
				if status != http.StatusForbidden || !strings.Contains(body, "admin only") {
					t.Errorf("%s: status %d body %s, want 403 admin only", who, status, body)
				}
			}
		})
	}
}

// Every test that reads or writes the global signup_mode row takes the
// cross-process lock (review-4 TEST-02), because `go test ./...` runs the store
// suite beside this one and its TestSignupModeSetting flips the row to "open".
// A new test that touches the row belongs in this list.
func TestSignupModeTouchersWaitForTheCrossProcessLock(t *testing.T) {
	w, _ := newAdminWorld(t)
	testlock.AssertWaits(t, w.st.Pool, testlock.SignupModeLock,
		"TestAdminCanReadAndWriteSignupMode",
		"TestAnAgentTokenCannotReachTheAdminRoutes",
		"TestInvitationRoundTrip",
		"TestAuthConfigReportsWhetherMailIsDelivered",
	)
}

func TestAdminCanReadAndWriteSignupMode(t *testing.T) {
	w, admin := newAdminWorld(t)
	ctx := context.Background()
	updatedAt := func() time.Time {
		var at time.Time
		if err := w.st.Pool.QueryRow(ctx, `SELECT updated_at FROM insideout.settings WHERE key = 'signup_mode'`).Scan(&at); err != nil {
			t.Fatalf("read signup_mode row: %v", err)
		}
		return at
	}

	status, body := w.call(admin, http.MethodGet, "/api/v1/admin/settings", "")
	expect(t, "GET settings", status, body, http.StatusOK)
	mode := decodeAs[map[string]string](t, body)["signupMode"]
	if mode == "" {
		t.Fatalf("GET settings: no signupMode in %s", body)
	}

	// signup_mode is one global row that store/TestSignupModeSetting also
	// reads, in a package that runs beside this one. So write back the mode it
	// already has instead of flipping it: the write is still proven, by the
	// row's updated_at moving.
	before := updatedAt()
	status, body = w.call(admin, http.MethodPut, "/api/v1/admin/settings", `{"signupMode":"`+mode+`"}`)
	expect(t, "PUT settings", status, body, http.StatusOK)
	if got := decodeAs[map[string]string](t, body)["signupMode"]; got != mode {
		t.Fatalf("PUT settings answered %q, want %q", got, mode)
	}
	if !updatedAt().After(before) {
		t.Fatal("PUT settings answered 200 but did not write the setting")
	}

	status, body = w.call(admin, http.MethodPut, "/api/v1/admin/settings", `{"signupMode":"sideways"}`)
	expect(t, "PUT settings with an unknown mode", status, body, http.StatusBadRequest)
}

func TestAdminInvitationLifecycle(t *testing.T) {
	w, admin := newAdminWorld(t)
	email := "invitee-" + uuid.NewString() + "@test.local"

	status, body := w.call(admin, http.MethodPost, "/api/v1/admin/invitations", `{"email":"`+email+`","ttlHours":1}`)
	expect(t, "create invitation", status, body, http.StatusCreated)
	created := decodeAs[struct {
		Invitation struct{ ID, Email string }
		Code       string
		SignupURL  string `json:"signupUrl"`
	}](t, body)
	if created.Invitation.Email != email || created.Code == "" || !strings.HasSuffix(created.SignupURL, "/register?invite="+created.Code) {
		t.Fatalf("create invitation answered %s, want the email, a one-time code and a signup link carrying it", body)
	}

	type listed struct {
		ID        string
		RevokedAt *string
	}
	find := func() *listed {
		status, body := w.call(admin, http.MethodGet, "/api/v1/admin/invitations", "")
		expect(t, "list invitations", status, body, http.StatusOK)
		for _, inv := range decodeAs[[]listed](t, body) {
			if inv.ID == created.Invitation.ID {
				return &inv
			}
		}
		return nil
	}
	if inv := find(); inv == nil || inv.RevokedAt != nil {
		t.Fatalf("list invitations: want the new invitation, live; got %+v", inv)
	}

	revoke := "/api/v1/admin/invitations/" + created.Invitation.ID + "/revoke"
	status, body = w.call(admin, http.MethodPost, revoke, "")
	expect(t, "revoke invitation", status, body, http.StatusOK)
	if inv := find(); inv == nil || inv.RevokedAt == nil {
		t.Fatalf("list invitations after revoke: want it marked revoked; got %+v", inv)
	}
	status, body = w.call(admin, http.MethodPost, revoke, "")
	expect(t, "revoke an already revoked invitation", status, body, http.StatusNotFound)
}
