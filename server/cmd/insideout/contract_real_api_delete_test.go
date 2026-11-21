package main

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
	"github.com/frankji-groundcontrol/insideout/server/internal/clientconfig"
)

// rv5-TREEB-10 follow-up: DELETE /roadmap/{nid} refuses a branch holding
// evidence or finished work unless ?force=true. The CLI shows the counts from
// that 409, names --force, and `roadmap delete --force` goes through.
// DATABASE_URL-gated.
func TestCLIDeletesABranchWithHistoryOnlyWithForce(t *testing.T) {
	w := newContractWorld(t)
	owner := apiclient.New(w.base)
	if err := owner.Login(w.email, w.password); err != nil {
		t.Fatalf("login: %v", err)
	}
	claims, _ := apiclient.ParseTokenClaims(owner.Token())
	writeClientConfig(t, clientconfig.Config{APIBase: w.base, Token: owner.Token(), RefreshToken: owner.RefreshToken()})

	sfx := uuid.NewString()[:8]
	ws := cliJSON[doc](t, "workspaces", "create", "--title", "Delete "+sfx)
	pr := cliJSON[doc](t, "projects", "create", "--title", "Board "+sfx, ws.ID)
	node := cliJSON[doc](t, "roadmap", "add", "--title", "Branch "+sfx, pr.ID)
	if err := w.st.AddRoadmapEvidence(context.Background(), uuid.MustParse(claims.Subject), uuid.MustParse(node.ID), "commit", "abc1234 branch", ""); err != nil {
		t.Fatalf("evidence: %v", err)
	}

	refused := runCLI(t, cliEnv(), "", "roadmap", "delete", node.ID)
	if refused.code == 0 || !strings.Contains(refused.stderr, "1 evidence") || !strings.Contains(refused.stderr, "--force") {
		t.Fatalf("roadmap delete over evidence: exit %d\n%s\nwant a non-zero exit naming 1 evidence row and --force", refused.code, refused.output())
	}
	if out := cliOut(t, "roadmap", "delete", "--force", node.ID); strings.TrimSpace(out) != "deleted" {
		t.Fatalf("roadmap delete --force: %q, want deleted", out)
	}
}
