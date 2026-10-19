package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// COMP-04: a PRD save checked only that the section key exists, so one section
// could carry the whole 1 MiB body and every coach turn then repeated it. A
// section is capped at 10,000 characters (characters, not bytes, like every
// other field cap); the save that goes over is refused whole, and the answer
// names the section and the limit. wantSectionCap is the documented contract,
// written out here on purpose: raising it is a decision that belongs in the docs.
const wantSectionCap = 10_000

func (w *routeWorld) saveSections(t *testing.T, prdID string, sections map[string]string) (int, string) {
	t.Helper()
	return w.send(t, w.owner, http.MethodPatch, "/api/v1/prds/"+prdID, map[string]any{"sections": sections})
}

func TestPrdSectionCap_RefusesAnOversizeSectionAndNamesIt(t *testing.T) {
	w := newRouteWorld(t)
	prd := w.prd(t)
	// 200,000 characters is the review's case: 600 KB, under the 1 MiB body cap,
	// so only a text cap can stop it.
	for _, n := range []int{wantSectionCap + 1, 200_000} {
		code, body := w.saveSections(t, prd, map[string]string{"background": cjk(n)})
		if code != http.StatusBadRequest {
			t.Fatalf("a %d-character section: status %d, want 400: the save stored it and every coach turn will carry it", n, code)
		}
		for _, want := range []string{"background", "10000"} {
			if !strings.Contains(body, want) {
				t.Errorf("a %d-character section: the answer %q does not name %q", n, body, want)
			}
		}
	}
}

func TestPrdSectionCap_EveryKeyTakesExactlyTheCapInCharacters(t *testing.T) {
	w := newRouteWorld(t)
	prd := w.prd(t)
	for _, key := range store.PrdSectionKeys {
		// 10,000 three-byte characters are 30,000 bytes: a byte count would refuse them.
		if code, body := w.saveSections(t, prd, map[string]string{key: cjk(wantSectionCap)}); code != http.StatusOK {
			t.Errorf("%s at exactly %d characters: status %d, want 200: %s", key, wantSectionCap, code, body)
		}
		code, body := w.saveSections(t, prd, map[string]string{key: cjk(wantSectionCap + 1)})
		if code != http.StatusBadRequest || !strings.Contains(body, key) {
			t.Errorf("%s at %d characters: status %d body %s, want 400 naming the section", key, wantSectionCap+1, code, body)
		}
	}
}

// One bad section refuses the whole save: nothing in the request is stored.
func TestPrdSectionCap_ARefusedSaveStoresNothing(t *testing.T) {
	w := newRouteWorld(t)
	prd := w.prd(t)
	code, body := w.saveSections(t, prd, map[string]string{"goals": "a fine goal", "risks": cjk(wantSectionCap + 1)})
	if code != http.StatusBadRequest {
		t.Fatalf("status %d, want 400: %s", code, body)
	}
	code, body = w.call(w.owner, http.MethodGet, "/api/v1/prds/"+prd, "")
	expect(t, "read the PRD back", code, body, http.StatusOK)
	if got := decodeAs[prdView](t, body).Sections["goals"]; got != "" {
		t.Errorf("the refused save stored goals = %q, want it untouched", got)
	}
}
