package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// FEAT-07: PRODUCT.md promises "Reverts and failed deployments append
// evidence; they do not erase history", but GitHub reports a deployment's
// outcome in a deployment_status delivery fired after the creation event —
// which the webhook drops today.

const deploymentStatusPayload = `{
  "action": "created",
  "deployment_status": {"state": "failure", "id": 1},
  "deployment": {"ref": "refs/heads/main", "environment": "production", "sha": "abc1234567"},
  "repository": {"full_name": "own/repo"}
}`

func mustEvent(t *testing.T, payload string) *webhookEvent {
	t.Helper()
	var ev webhookEvent
	if err := json.Unmarshal([]byte(payload), &ev); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return &ev
}

// A failed deployment appends its own activity row ("deploy failed: …"),
// keyed by the (node, kind, detail) idempotency index — the release row
// from the creation event is never rewritten.
func TestPlanGuideEvidenceDeploymentStatusFailure(t *testing.T) {
	p, ok := planGuideEvidence("deployment_status", mustEvent(t, deploymentStatusPayload), "own/repo")
	if !ok {
		t.Fatal("ok = false, want a failure status to append evidence")
	}
	if p.kind != "activity" {
		t.Errorf("kind = %q, want activity", p.kind)
	}
	if want := "deploy failed: production @ abc1234"; p.detail != want {
		t.Errorf("detail = %q, want %q", p.detail, want)
	}
	if want := "https://github.com/own/repo/commit/abc1234567"; p.source != want {
		t.Errorf("source = %q, want %q", p.source, want)
	}
	if p.branch != "main" {
		t.Errorf("branch = %q, want main (the deployment's ref)", p.branch)
	}
}

// Success (and in-flight) statuses refine nothing: the release row the
// creation event appended already stands, so no second row may appear.
func TestPlanGuideEvidenceDeploymentStatusNonFailureAppendsNothing(t *testing.T) {
	for _, state := range []string{"success", "queued", "in_progress"} {
		payload := `{"deployment_status":{"state":"` + state + `"},"deployment":{"ref":"refs/heads/main","environment":"production","sha":"abc1234567"}}`
		if _, ok := planGuideEvidence("deployment_status", mustEvent(t, payload), "own/repo"); ok {
			t.Errorf("state %q: ok = true, want no evidence for a non-failure status", state)
		}
	}
}

// The creation event keeps mapping to the release row exactly as before
// (CC-01) — guards the extraction that made the status mapping testable.
func TestPlanGuideEvidenceDeploymentCreationStillRelease(t *testing.T) {
	payload := `{"deployment":{"ref":"refs/heads/main","environment":"production","sha":"abc1234567"}}`
	p, ok := planGuideEvidence("deployment", mustEvent(t, payload), "own/repo")
	if !ok {
		t.Fatal("ok = false, want the creation event to append evidence")
	}
	if p.kind != "release" || p.detail != "deploy production @ abc1234" {
		t.Errorf("creation event: kind=%q detail=%q, want release / deploy production @ abc1234", p.kind, p.detail)
	}
}

// End to end through the handler: a signed deployment_status delivery is
// routed into the evidence pipeline, not answered as an ignored event.
// The repo is deliberately unlinked, so no guide fetch happens — the
// assertion is on routing, which is the part the finding says is dropped.
func TestDeploymentStatusDeliveryIsRoutedNotIgnored(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	st, err := store.Open(context.Background(), url)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(st.Close)
	s := NewServer(st, nil, &config.Config{GithubWebhookSecret: hookSecret},
		slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil)

	w := deliver(s, "deployment_status", deploymentStatusPayload, hmacHeader(hookSecret, deploymentStatusPayload))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d, want 200; body %s", w.Code, w.Body)
	}
	got := decodeAs[map[string]any](t, w.Body.String())
	if _, ignored := got["ignored"]; ignored {
		t.Fatalf("body %s, want the delivery routed (event field), not ignored", w.Body)
	}
	if got["event"] != "deployment_status" {
		t.Fatalf("body %s, want event:deployment_status", w.Body)
	}
}
