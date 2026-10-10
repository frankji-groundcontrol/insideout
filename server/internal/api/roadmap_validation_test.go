package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/agent"
	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// FEAT-04 (review 3), the HTTP half: a human records a validation
// observation on a leaf, and a done leaf reaches 'validated' only through
// the confirm endpoint carrying a reason — never a plain status PATCH.

// validationServer builds the full server plus one project with a root
// and one leaf, and mints the creator's token.
func validationServer(t *testing.T) (*httptest.Server, store.RoadmapNode, string) {
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
	ctx := context.Background()

	user, err := st.CreateUser(ctx, "validation-"+uuid.NewString()+"@test.local", "x", "validation")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	ws, err := st.CreateWorkspace(ctx, user.ID, "Validation API WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	proj, err := st.CreateProject(ctx, user.ID, ws.ID, "Validation API project", "")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	root, err := st.CreateRoadmapNode(ctx, user.ID, proj.ID, nil, "Ship", "")
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	leaf, err := st.CreateRoadmapNode(ctx, user.ID, proj.ID, &root.ID, "Tour", "")
	if err != nil {
		t.Fatalf("create leaf: %v", err)
	}

	tokens := auth.NewTokenIssuer("a-32-byte-or-longer-test-secret!", time.Hour)
	token, err := tokens.MintAccessToken(user.ID)
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	srv := NewServer(st, tokens, &config.Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, agent.NewTemplateRoadmapPlanner())
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts, *leaf, token
}

func validationStatus(t *testing.T, ts *httptest.Server, token string, code int, method, path, body string) {
	t.Helper()
	resp := doJSON(t, ts, token, method, path, body)
	defer resp.Body.Close()
	if resp.StatusCode != code {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("%s %s: status %d, want %d; body %s", method, path, resp.StatusCode, code, raw)
	}
}

func TestValidationObservationEndpoint(t *testing.T) {
	ts, leaf, token := validationServer(t)

	body := `{"detail":"4 of 5 pilot users finished the tour without help","sourceUrl":"pilot check-in thread","observedAt":"2026-10-02"}`
	validationStatus(t, ts, token, http.StatusCreated, http.MethodPost, "/api/v1/roadmap/"+leaf.ID.String()+"/evidence", body)

	// The observation reads back as evidence of the validation kind.
	resp := doJSON(t, ts, token, http.MethodGet, "/api/v1/roadmap/"+leaf.ID.String()+"/evidence", "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list evidence: status %d", resp.StatusCode)
	}
	var rows []store.RoadmapEvidence
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		t.Fatalf("decode evidence: %v", err)
	}
	if len(rows) != 1 || rows[0].Kind != "validation" {
		t.Fatalf("rows = %+v, want one validation row", rows)
	}
	if rows[0].ObservedAt == nil || rows[0].ObservedAt.Format("2006-01-02") != "2026-10-02" {
		t.Fatalf("ObservedAt = %v, want 2026-10-02", rows[0].ObservedAt)
	}

	// An observation with nothing observed, or an undated one, is refused
	// before the store sees it.
	validationStatus(t, ts, token, http.StatusBadRequest, http.MethodPost,
		"/api/v1/roadmap/"+leaf.ID.String()+"/evidence", `{"detail":"   ","observedAt":"2026-10-02"}`)
	validationStatus(t, ts, token, http.StatusBadRequest, http.MethodPost,
		"/api/v1/roadmap/"+leaf.ID.String()+"/evidence", `{"detail":"seen","observedAt":"Oct 2"}`)
}

func TestValidateNodeConfirmEndpoint(t *testing.T) {
	ts, leaf, token := validationServer(t)

	// Walk the leaf to done over the public API, like the canvas does.
	for _, status := range []string{"in_progress", "done"} {
		validationStatus(t, ts, token, http.StatusOK, http.MethodPatch,
			"/api/v1/roadmap/"+leaf.ID.String(), `{"status":"`+status+`"}`)
	}

	// The plain status PATCH cannot reach validated: it is not a human
	// confirm and carries no reason.
	validationStatus(t, ts, token, http.StatusBadRequest, http.MethodPatch,
		"/api/v1/roadmap/"+leaf.ID.String(), `{"status":"validated"}`)

	// The confirm without a reason is refused.
	validationStatus(t, ts, token, http.StatusBadRequest, http.MethodPost,
		"/api/v1/roadmap/"+leaf.ID.String()+"/validate", `{"reason":"  "}`)

	// The confirm with a reason lands, and the reason rides the node.
	validationStatus(t, ts, token, http.StatusOK, http.MethodPost,
		"/api/v1/roadmap/"+leaf.ID.String()+"/validate", `{"reason":"7-day retention rose from 40% to 68%"}`)

	list := doJSON(t, ts, token, http.MethodGet, "/api/v1/projects/"+leaf.ProjectID.String()+"/roadmap", "")
	defer list.Body.Close()
	if list.StatusCode != http.StatusOK {
		t.Fatalf("list roadmap: status %d", list.StatusCode)
	}
	var views []struct {
		ID               string `json:"id"`
		Status           string `json:"status"`
		ValidationReason string `json:"validationReason"`
	}
	if err := json.NewDecoder(list.Body).Decode(&views); err != nil {
		t.Fatalf("decode roadmap: %v", err)
	}
	for _, v := range views {
		if v.ID == leaf.ID.String() {
			if v.Status != "validated" {
				t.Fatalf("status = %q, want validated", v.Status)
			}
			if v.ValidationReason != "7-day retention rose from 40% to 68%" {
				t.Fatalf("validationReason = %q", v.ValidationReason)
			}
			return
		}
	}
	t.Fatal("leaf missing from roadmap list")
}
