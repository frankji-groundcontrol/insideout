package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// rv5 wave 2 verifier follow-ups on the task tree (TREEX-08, the agent root
// check). Real PostgreSQL.

// An opened node's ancestors name their owners, so a client can tell that the
// caller owns the opened tree through an ancestor.
func TestAnOpenedNodesAncestorsCarryTheirOwners(t *testing.T) {
	w := newTreeWorld(t)
	got := decodeAs[struct {
		Ancestors []map[string]any `json:"ancestors"`
	}](t, w.must(t, w.member, "GET", nodePath(w.card.ID), "", http.StatusOK))
	if len(got.Ancestors) != 2 {
		t.Fatalf("ancestors of Card: %v, want Root and Payments", got.Ancestors)
	}
	root, payments := got.Ancestors[0], got.Ancestors[1]
	if v, has := root["ownerId"]; !has || v != nil {
		t.Errorf("Root ancestor: ownerId %v (present %v), want an explicit null", v, has)
	}
	if payments["ownerId"] != w.member.ID.String() {
		t.Errorf("Payments ancestor: ownerId %v, want the member", payments["ownerId"])
	}
}

// feedContent returns the content of the project feed rows of kind about node.
func (w *treeWorld) feedContent(t *testing.T, kind string, node uuid.UUID) []string {
	t.Helper()
	got := decodeAs[struct {
		Updates []map[string]any `json:"updates"`
	}](t, w.must(t, w.owner, "GET", w.proj(), "", http.StatusOK))
	var out []string
	for _, u := range got.Updates {
		if u["kind"] == kind && u["nodeId"] == node.String() {
			out = append(out, u["content"].(string))
		}
	}
	return out
}

// Starting an ownerless leaf makes the actor its owner. The feed says they
// took it on, not that it was delegated.
func TestStartingAnOwnerlessLeafReadsAsTakingItOn(t *testing.T) {
	w := newTreeWorld(t)
	w.must(t, w.owner, "PATCH", nodePath(w.shipping.ID), `{"status":"in_progress"}`, http.StatusOK)
	got := w.feedContent(t, "node_delegated", w.shipping.ID)
	if len(got) != 1 || !strings.HasPrefix(got[0], `Took on "Shipping"`) {
		t.Errorf("feed for starting Shipping: %q, want one node_delegated row reading Took on \"Shipping\"", got)
	}
	if got := w.feedContent(t, "node_delegated", w.payments.ID); len(got) != 1 || !strings.HasPrefix(got[0], "Delegated") {
		t.Errorf("feed for delegating Payments: %q, want it still reading Delegated", got)
	}
}
