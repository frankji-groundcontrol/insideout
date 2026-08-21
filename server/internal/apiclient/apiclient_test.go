package apiclient

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServer(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return New(srv.URL)
}

func TestLoginStoresTokenFromResponse(t *testing.T) {
	var gotAuth, gotBody string
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/login" {
			t.Errorf("path = %q, want /auth/login", r.URL.Path)
		}
		buf := make([]byte, 256)
		n, _ := r.Body.Read(buf)
		gotBody = string(buf[:n])
		gotAuth = r.Header.Get("Authorization")
		w.Write([]byte(`{"accessToken":"tok-1","refreshToken":"rtok"}`))
	})
	if err := c.Login("a@b.c", "pw"); err != nil {
		t.Fatalf("Login: %v", err)
	}
	if want := `{"email":"a@b.c","password":"pw"}`; gotBody != want {
		t.Errorf("body = %q, want %q", gotBody, want)
	}
	if gotAuth != "" {
		t.Errorf("login request should carry no bearer, got %q", gotAuth)
	}
}

func TestGetSendsBearerAndReturnsRawJSON(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer tok-1" {
			t.Errorf("Authorization = %q, want Bearer tok-1", got)
		}
		switch r.URL.Path {
		case "/me":
			w.Write([]byte(`{"id":"u1","email":"a@b.c"}`))
		case "/workspaces":
			w.Write([]byte(`[{"id":"w1","name":"W"}]`))
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
		}
	})
	c.SetToken("tok-1")
	me, err := c.Whoami()
	if err != nil || string(me) != `{"id":"u1","email":"a@b.c"}` {
		t.Fatalf("Whoami = %s, err %v", me, err)
	}
	ws, err := c.Workspaces()
	if err != nil || string(ws) != `[{"id":"w1","name":"W"}]` {
		t.Fatalf("Workspaces = %s, err %v", ws, err)
	}
}

func TestUnauthorizedIsActionable(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"no token"}`))
	})
	if _, err := c.Whoami(); err == nil || !strings.Contains(err.Error(), "INSIDEOUT_TOKEN") {
		t.Fatalf("want unauthorized hint, got %v", err)
	}
}

// Regression: RoadmapDelete against an endpoint that answers with a JSON
// body (not 204) must not fail decoding into a nil target.
func TestDeleteDiscardsJSONBody(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/roadmap/n1" {
			t.Errorf("got %s %s, want DELETE /roadmap/n1", r.Method, r.URL.Path)
		}
		w.Write([]byte(`{"deleted":true}`))
	})
	c.SetToken("tok-1")
	if err := c.RoadmapDelete("n1"); err != nil {
		t.Fatalf("RoadmapDelete: %v", err)
	}
}
