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
// outcome in a deployment_status delivery fired after the creation event.
// The creation and success mappings live in github_webhook_release_test.go
// (rv4-FEAT-07); this file keeps the failure mapping and the routing.

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
// keyed by the (node, kind, detail) idempotency index — nothing already
// recorded is rewritten.
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
