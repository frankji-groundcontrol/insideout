package api

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

// rv5-PARC-10: the evidence read answers 404 for a node that does not exist
// or that the caller cannot see, like its sibling roadmap routes, and writes
// its times in the API's layouts. Real PostgreSQL.

func TestEvidenceReadRefusesAnInvisibleNodeAndUsesTheAPITimeLayout(t *testing.T) {
	w := newTreeWorld(t)
	path := "/api/v1/roadmap/" + w.card.ID.String() + "/evidence"
	w.must(t, w.owner, "POST", path, `{"detail":"five people paid","observedAt":"2026-11-20"}`, http.StatusCreated)

	rows := decodeAs[[]map[string]any](t, w.must(t, w.member, "GET", path, "", http.StatusOK))
	if len(rows) != 1 {
		t.Fatalf("evidence rows: %v, want the one observation", rows)
	}
	created, _ := rows[0]["createdAt"].(string)
	if at, err := time.Parse(timeLayout, created); err != nil || at.Format(timeLayout) != created {
		t.Errorf("createdAt %q is not in the API's time layout", created)
	}
	if rows[0]["observedAt"] != "2026-11-20" {
		t.Errorf("observedAt = %v, want the date as it was entered", rows[0]["observedAt"])
	}

	w.must(t, w.outsider, "GET", path, "", http.StatusNotFound)
	w.must(t, w.member, "GET", "/api/v1/roadmap/"+uuid.NewString()+"/evidence", "", http.StatusNotFound)
}
