package api

import (
	"net/http"
	"testing"
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
