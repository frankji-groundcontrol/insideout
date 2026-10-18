package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// rv4-FEAT-07: GitHub fires `deployment` when a deployment is CREATED, before
// anyone knows whether it worked, and the webhook recorded that as `release`
// evidence. A deployment that then failed (or never finished) had already put
// "release" on the roadmap node. PRODUCT.md: a deployment "may show release" -
// so release is recorded when a deployment_status says the deployment
// succeeded, creation stays plain activity, and a failure still appends its
// own row without erasing anything.

const deployRef = `"deployment":{"ref":"refs/heads/main","environment":"production","sha":"abc1234567"}`

func TestDeploymentCreationIsActivityNotRelease(t *testing.T) {
	p, ok := planGuideEvidence("deployment", mustEvent(t, `{`+deployRef+`}`), "own/repo")
	if !ok {
		t.Fatal("ok = false, want the creation event to leave a trace")
	}
	if p.kind != "activity" {
		t.Errorf("kind = %q, want activity: nothing is known about the outcome yet", p.kind)
	}
}

func TestDeploymentStatusMapsToEvidenceByOutcome(t *testing.T) {
	cases := []struct {
		state    string
		wantKind string // "" = no evidence
	}{
		{"success", "release"},
		{"failure", "activity"},
		{"error", "activity"},
		{"queued", ""},
		{"pending", ""},
		{"in_progress", ""},
		{"inactive", ""},
	}
	for _, tc := range cases {
		t.Run(tc.state, func(t *testing.T) {
			payload := `{"deployment_status":{"state":"` + tc.state + `"},` + deployRef + `}`
			p, ok := planGuideEvidence("deployment_status", mustEvent(t, payload), "own/repo")
			if tc.wantKind == "" {
				if ok {
					t.Fatalf("ok = true (kind %q), want no evidence for %q", p.kind, tc.state)
				}
				return
			}
			if !ok || p.kind != tc.wantKind {
				t.Fatalf("state %q: ok=%v kind=%q, want kind %q", tc.state, ok, p.kind, tc.wantKind)
			}
			if !strings.Contains(p.detail, "production") || !strings.Contains(p.detail, "abc1234") {
				t.Errorf("detail = %q, want the environment and short sha", p.detail)
			}
			if want := "https://github.com/own/repo/commit/abc1234567"; p.source != want {
				t.Errorf("source = %q, want %q", p.source, want)
			}
		})
	}
}

// The whole life of one deployment, through the handler and the database: the
// node shows release only once a success status has arrived, a redelivery adds
// nothing, and a later failure appends its own row.
func TestDeploymentLifecycleRecordsReleaseOnlyAfterSuccess(t *testing.T) {
	f := newFakeGitHub(t)
	w := appServerWorld(t)
	ctx := context.Background()
	node, err := w.st.CreateRoadmapNode(ctx, w.owner.ID, w.project.ID, nil, "Ship it", "")
	if err != nil {
		t.Fatalf("create node: %v", err)
	}
	name := uniqueRepo()
	f.addRepo(name, fakeRepo{guide: fmt.Sprintf("version: 1\nnodes:\n  %s:\n    branches: [main]\n", node.ID)})
	code, body := putRepo(t, w, name)
	expect(t, "link the repo", code, body, http.StatusOK)

	send := func(event, extra string) {
		t.Helper()
		payload := fmt.Sprintf(`{%s%s,"repository":{"full_name":%q}}`, extra, deployRef, name)
		rec := deliver(w.srv, event, payload, hmacHeader(hookSecret, payload))
		expect(t, event+" delivery", rec.Code, rec.Body.String(), http.StatusOK)
	}
	kinds := func() map[string]int {
		t.Helper()
		rows, err := w.st.ListRoadmapEvidence(ctx, w.owner.ID, node.ID)
		if err != nil {
			t.Fatalf("list evidence: %v", err)
		}
		out := map[string]int{}
		for _, e := range rows {
			out[e.Kind]++
		}
		return out
	}
	status := func(state string) {
		t.Helper()
		send("deployment_status", fmt.Sprintf(`"deployment_status":{"state":%q},`, state))
	}

	send("deployment", `"action":"created",`)
	status("queued")
	status("in_progress")
	if got := kinds(); got["release"] != 0 {
		t.Fatalf("evidence %v before any outcome: want no release row", got)
	}

	status("success")
	status("success") // GitHub redelivers: the (node, kind, detail) key makes it free
	if got := kinds(); got["release"] != 1 {
		t.Fatalf("evidence %v after success: want exactly one release row", got)
	}

	status("failure")
	got := kinds()
	if got["release"] != 1 || got["activity"] < 2 {
		t.Errorf("evidence %v after a later failure: want the release row kept and the failure appended as activity", got)
	}
}
