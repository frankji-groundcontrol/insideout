package probe

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServiceHealthRequiresHealthzAndTheAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/api/v1/me":
			w.WriteHeader(http.StatusUnauthorized)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	checks, err := ServiceHealth(t.Context(), srv.Client(), srv.URL+"/api/v1")
	if err != nil {
		t.Fatalf("ServiceHealth: %v", err)
	}
	if len(checks) != 3 || checks[1].Detail != "/healthz ok" || checks[2].Detail != "authentication required" {
		t.Fatalf("checks = %+v", checks)
	}
}

func TestServiceHealthFailsWhenTheDatabaseIsDown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			w.WriteHeader(http.StatusServiceUnavailable)
		case "/api/v1/me":
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer srv.Close()

	checks, err := ServiceHealth(t.Context(), srv.Client(), srv.URL+"/api/v1")
	if err == nil || !strings.Contains(err.Error(), "/healthz answered 503 with an empty body") {
		t.Fatalf("err = %v", err)
	}
	if checks[1].Status != "fail" {
		t.Fatalf("health check = %+v", checks[1])
	}
}

func TestServiceHealthAcceptsPlainOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("ok\n"))
		case "/api/v1/me":
			w.WriteHeader(http.StatusUnauthorized)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	checks, err := ServiceHealth(t.Context(), srv.Client(), srv.URL+"/api/v1")
	if err != nil {
		t.Fatalf("ServiceHealth: %v", err)
	}
	if checks[1].Status != "ok" {
		t.Fatalf("health = %+v", checks[1])
	}
}

func TestServiceHealthNamesTheBadBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":"database unreachable"}`))
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	checks, err := ServiceHealth(t.Context(), srv.Client(), srv.URL+"/api/v1")
	if err == nil || checks[1].Detail != `/healthz answered 503: {"error":"database unreachable"}` {
		t.Fatalf("health = %+v, err = %v", checks[1], err)
	}
}

func TestServiceHealthRejectsABareHost(t *testing.T) {
	_, err := ServiceHealth(t.Context(), nil, "not a url")
	if err == nil {
		t.Fatal("expected a bad address to fail")
	}
}
