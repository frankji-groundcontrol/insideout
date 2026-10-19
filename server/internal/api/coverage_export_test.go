package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// exportWorld is a world with a PRD whose every section carries a marker of its
// own, so an export shows which sections it carried.
func exportWorld(t *testing.T) (w *routeWorld, prdID string, marker map[string]string) {
	t.Helper()
	w = newCoverageWorld(t)
	prdID, _ = w.convertedPrd(t, w.member, "Export <Me>")
	marker = map[string]string{}
	sections := map[string]string{}
	for _, key := range store.PrdSectionKeys {
		marker[key] = "MARK-" + strings.ToUpper(key) + "-" + uuid.NewString()[:8]
		sections[key] = marker[key]
	}
	sections["risks"] += "\n\n<script>alert(1)</script>"
	w.must(t, w.member, "PATCH", "/api/v1/prds/"+prdID, jsonBody(t, map[string]any{"sections": sections}), http.StatusOK)
	return w, prdID, marker
}

func TestPrdExportRendersMarkdownWithAllSections(t *testing.T) {
	w, id, marker := exportWorld(t)
	path := "/api/v1/prds/" + id + "/export?format=markdown"

	for _, as := range []routeUser{w.member, w.owner} {
		rec := w.callRaw(as, "GET", path)
		if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "text/markdown; charset=utf-8" ||
			rec.Header().Get("Content-Disposition") != `attachment; filename="prd.md"` {
			t.Fatalf("markdown export = %d, type %q, disposition %q", rec.Code, rec.Header().Get("Content-Type"), rec.Header().Get("Content-Disposition"))
		}
		body := rec.Body.String()
		if !strings.HasPrefix(body, "# Export <Me>\n") {
			t.Fatalf("markdown starts %q, want the title as a heading", body[:min(len(body), 40)])
		}
		for key, m := range marker {
			if !strings.Contains(body, m) {
				t.Fatalf("markdown export lacks the %s section (%s)", key, m)
			}
		}
	}
	w.must(t, w.outsider, "GET", path, "", http.StatusNotFound)
	w.must(t, w.member, "GET", "/api/v1/prds/"+uuid.NewString()+"/export?format=markdown", "", http.StatusNotFound)
	w.must(t, w.member, "GET", "/api/v1/prds/not-a-uuid/export?format=markdown", "", http.StatusBadRequest)
}

func TestPrdExportFormatsAreChecked(t *testing.T) {
	w, id, _ := exportWorld(t)
	base := "/api/v1/prds/" + id + "/export"
	for _, query := range []string{"", "?format=pdf", "?format=Markdown", "?audience=decision"} {
		w.must(t, w.member, "GET", base+query, "", http.StatusBadRequest)
	}
	// The format is judged before the PRD is looked up.
	w.must(t, w.outsider, "GET", base+"?format=pdf", "", http.StatusBadRequest)
}

func TestPrdExportPrintIsEscapedHTML(t *testing.T) {
	w, id, marker := exportWorld(t)
	rec := w.callRaw(w.member, "GET", "/api/v1/prds/"+id+"/export?format=print")
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("print export = %d, type %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	if rec.Header().Get("Content-Disposition") != "" {
		t.Fatalf("print export names a download (%q); it is meant to open in the browser", rec.Header().Get("Content-Disposition"))
	}
	body := rec.Body.String()
	for _, want := range []string{"<h1>Export &lt;Me&gt;</h1>", "<title>Export &lt;Me&gt;</title>", marker["background"], "&lt;script&gt;alert(1)&lt;/script&gt;"} {
		if !strings.Contains(body, want) {
			t.Fatalf("print export lacks %q", want)
		}
	}
	if strings.Contains(body, "<script>") {
		t.Fatal("print export carries a live <script> from a PRD section")
	}
	// An audience projection exists as markdown only.
	w.must(t, w.member, "GET", "/api/v1/prds/"+id+"/export?format=print&audience=decision", "", http.StatusBadRequest)
}

// Each audience reads a chosen subset of the one PRD, with its own file name.
func TestPrdExportAudienceViewsProjectTheirSections(t *testing.T) {
	w, id, marker := exportWorld(t)
	reads := map[string][]string{
		"decision":   {"background", "goals", "nonGoals", "risks"},
		"management": {"users", "goals", "requirements", "nonGoals", "constraints"},
		"delivery":   {"requirements", "constraints", "stories"},
		"validation": {"background", "users", "stories", "risks"},
	}
	for audience, keys := range reads {
		t.Run(audience, func(t *testing.T) {
			rec := w.callRaw(w.owner, "GET", "/api/v1/prds/"+id+"/export?format=markdown&audience="+audience)
			if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "text/markdown; charset=utf-8" ||
				rec.Header().Get("Content-Disposition") != `attachment; filename="prd-`+audience+`-view.md"` {
				t.Fatalf("%s export = %d, type %q, disposition %q", audience, rec.Code, rec.Header().Get("Content-Type"), rec.Header().Get("Content-Disposition"))
			}
			body := rec.Body.String()
			carried := map[string]bool{}
			for _, key := range keys {
				carried[key] = true
			}
			for key, m := range marker {
				if has := strings.Contains(body, m); has != carried[key] {
					t.Fatalf("%s view carries the %s section = %v, want %v", audience, key, has, carried[key])
				}
			}
		})
	}
	w.must(t, w.owner, "GET", "/api/v1/prds/"+id+"/export?format=markdown&audience=everyone", "", http.StatusBadRequest)
	w.must(t, w.outsider, "GET", "/api/v1/prds/"+id+"/export?format=markdown&audience=decision", "", http.StatusNotFound)
}
