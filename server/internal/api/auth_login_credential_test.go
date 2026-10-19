package api

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

// rv4-DATA-03 (read half): login is the one flow that must hold the stored
// credential, and it has to get it without its database role being able to
// SELECT password_hash from the table (the grant is revoked; the credential
// comes from a SECURITY DEFINER function behind store.GetUserByEmail). These
// tests drive the real login route over real PostgreSQL: it still answers a
// right password, a wrong one and an unknown email as before, a legacy bcrypt
// hash still upgrades in place, and the role those requests run as can no
// longer read the column.

func credentialWorld(t *testing.T) (*store.Store, http.Handler) {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	st, err := store.Open(context.Background(), url)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(st.Close)
	s := NewServer(st, auth.NewTokenIssuer(routeTestSecret, time.Hour),
		&config.Config{AccessTTL: 15 * time.Minute, RefreshTTL: 720 * time.Hour},
		slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil)
	return st, s.Handler()
}

func postLogin(h http.Handler, email, password string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"`+email+`","password":"`+password+`"}`))
	r.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func TestLoginCredential_RouteNeedsNoTableRead(t *testing.T) {
	st, h := credentialWorld(t)
	ctx := context.Background()
	const password = "correct horse battery staple"
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	u, err := st.CreateUser(ctx, "login-credential-"+uuid.NewString()+"@example.com", hash, "credential-tester")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	if rec := postLogin(h, u.Email, password); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"accessToken"`) {
		t.Errorf("login with the right password: status %d, want 200 with a session", rec.Code)
	}
	wrong := postLogin(h, u.Email, "not the password")
	unknown := postLogin(h, "no-such-user-"+uuid.NewString()+"@example.com", "not the password")
	if wrong.Code != http.StatusUnauthorized || unknown.Code != wrong.Code || unknown.Body.String() != wrong.Body.String() {
		t.Errorf("a wrong password and an unknown email must answer alike with a 401: wrong = %d %q, unknown = %d %q",
			wrong.Code, wrong.Body.String(), unknown.Code, unknown.Body.String())
	}

	// The route above ran as the pool role; that role must not read the
	// column it verified against.
	var pgErr *pgconn.PgError
	_, err = st.Pool.Exec(ctx, `SELECT password_hash FROM insideout.users WHERE email = $1`, u.Email)
	if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
		t.Errorf("SELECT password_hash as the pool role: want permission denied (42501), got %v", err)
	}
}

// Control: UpdatePasswordHash writes the column and never reads it, so the
// transparent bcrypt-to-argon2id upgrade survives the revoked SELECT.
func TestLoginCredential_LegacyBcryptUpgradesInPlace(t *testing.T) {
	st, h := credentialWorld(t)
	ctx := context.Background()
	const password = "old-password"
	legacy, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	u, err := st.CreateUser(ctx, "login-legacy-"+uuid.NewString()+"@example.com", string(legacy), "legacy-tester")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	if rec := postLogin(h, u.Email, password); rec.Code != http.StatusOK {
		t.Fatalf("login with a legacy bcrypt hash: status %d, want 200", rec.Code)
	}
	got, err := st.GetUserByEmail(ctx, u.Email)
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	if auth.IsBcryptHash(got.PasswordHash) {
		t.Error("the bcrypt hash was not upgraded to argon2id after a successful login")
	}
	if ok, err := auth.VerifyPassword(got.PasswordHash, password); err != nil || !ok {
		t.Errorf("the upgraded hash must verify the same password: ok=%v err=%v", ok, err)
	}
}
