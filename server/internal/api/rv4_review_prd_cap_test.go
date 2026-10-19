package api

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"unicode/utf8"
)

// rv4-FR-F127: the 10,000-character section cap checked every section a PATCH
// carried. The Flutter PRD page sends all eight sections back on every save,
// as it loaded them, so a section that is already over the cap (written by the
// coach, whose tool had no cap, or stored before the cap existed) made every
// save of that PRD answer 400, whatever the user had edited.

// rvPrdCapPlant stores text in one section of the owner's PRD the way a row
// written before the cap holds it: a plain UPDATE under the author's row-level
// security context, around the write path that now refuses such text.
func rvPrdCapPlant(t *testing.T, w *routeWorld, prdID, section, text string) {
	t.Helper()
	ctx := context.Background()
	tx, err := w.st.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT set_config('app.user_id', $1, true)`, w.owner.ID.String()); err != nil {
		t.Fatalf("set the acting user: %v", err)
	}
	tag, err := tx.Exec(ctx,
		`UPDATE insideout.prds SET sections = jsonb_set(sections, ARRAY[$2::text], to_jsonb($3::text)) WHERE id = $1::uuid`,
		prdID, section, text)
	if err != nil {
		t.Fatalf("plant the long section: %v", err)
	}
	if tag.RowsAffected() != 1 {
		t.Fatalf("plant the long section: %d rows updated, want 1", tag.RowsAffected())
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
}

// A section already over the cap is the PRD's business, not the save's: the
// page sends it back untouched, and the edit the user made must still land.
func TestRvPrdCapSavingAroundAnUnchangedLongSectionSucceeds(t *testing.T) {
	w := newRouteWorld(t)
	prd := w.prd(t)
	long := cjk(15_000)
	rvPrdCapPlant(t, w, prd, "background", long)

	// What the page does: load the PRD, change one field, send every section back.
	code, body := w.call(w.owner, http.MethodGet, "/api/v1/prds/"+prd, "")
	expect(t, "load the PRD", code, body, http.StatusOK)
	sections := decodeAs[prdView](t, body).Sections
	if sections["background"] != long {
		t.Fatalf("setup: the planted section came back as %d characters, want 15000", utf8.RuneCountInString(sections["background"]))
	}
	sections["goals"] = "a goal typed by hand"

	code, body = w.saveSections(t, prd, sections)
	if code != http.StatusOK {
		t.Errorf("save of every section with the 15000-character one unchanged: status %d, want 200: %s", code, body)
	}

	code, body = w.call(w.owner, http.MethodGet, "/api/v1/prds/"+prd, "")
	expect(t, "read the PRD back", code, body, http.StatusOK)
	got := decodeAs[prdView](t, body).Sections
	if got["goals"] != "a goal typed by hand" {
		t.Errorf("the edit was lost: goals = %q", got["goals"])
	}
	if got["background"] != long {
		t.Errorf("the untouched long section changed: it holds %d characters, want 15000", utf8.RuneCountInString(got["background"]))
	}
}

// Changing a long section is a write like any other: a trim that leaves it over
// the cap is refused, names the section and the limit, and stores nothing from
// the same save; a trim that brings it under the cap lands.
func TestRvPrdCapTrimmingALongSectionIsHeldToTheCap(t *testing.T) {
	w := newRouteWorld(t)
	prd := w.prd(t)
	long := cjk(15_000)
	rvPrdCapPlant(t, w, prd, "background", long)

	code, body := w.saveSections(t, prd, map[string]string{"background": cjk(12_000), "goals": "must not land"})
	if code != http.StatusBadRequest {
		t.Fatalf("a trim to 12000 characters, still over the cap: status %d, want 400: %s", code, body)
	}
	for _, want := range []string{"background", "10000"} {
		if !strings.Contains(body, want) {
			t.Errorf("the answer %q does not name %q", body, want)
		}
	}
	code, body = w.call(w.owner, http.MethodGet, "/api/v1/prds/"+prd, "")
	expect(t, "read the PRD back", code, body, http.StatusOK)
	got := decodeAs[prdView](t, body).Sections
	if got["background"] != long || got["goals"] != "" {
		t.Errorf("the refused save changed the PRD: background holds %d characters, goals = %q", utf8.RuneCountInString(got["background"]), got["goals"])
	}

	code, body = w.saveSections(t, prd, map[string]string{"background": cjk(9_000)})
	expect(t, "a trim to 9000 characters", code, body, http.StatusOK)
}
