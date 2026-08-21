package export

import (
	"github.com/frankji-groundcontrol/insideout/server/internal/audienceview"
	"strings"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

func TestMarkdown_IncludesTitleAndAllSections(t *testing.T) {
	sections := map[string]string{"background": "Users lose track of updates."}
	md := Markdown("My PRD", sections)

	if !strings.Contains(md, "# My PRD") {
		t.Fatalf("markdown missing title heading:\n%s", md)
	}
	if !strings.Contains(md, "Users lose track of updates.") {
		t.Fatalf("markdown missing background content:\n%s", md)
	}
	for _, key := range store.PrdSectionKeys {
		if !strings.Contains(md, sectionLabels[key]) {
			t.Fatalf("markdown missing heading for section %q:\n%s", key, md)
		}
	}
	if !strings.Contains(md, "not yet written") {
		t.Fatalf("markdown should mark empty sections as not yet written:\n%s", md)
	}
}

func TestPrintHTML_EscapesContent(t *testing.T) {
	sections := map[string]string{"background": "<script>alert(1)</script>"}
	html := PrintHTML("Title", sections)

	if strings.Contains(html, "<script>alert(1)</script>") {
		t.Fatalf("PrintHTML must escape section content, got raw script tag:\n%s", html)
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Fatalf("PrintHTML should contain the escaped form:\n%s", html)
	}
}

func TestMarkdownAudience_ProjectsOrderedSubset(t *testing.T) {
	proj := audienceview.Projection{
		Title:   "Decision view",
		Purpose: "For a boss.",
		Sections: []audienceview.SectionPick{
			{Key: "goals", Why: "the return"},
			{Key: "background", Why: "the why"},
		},
	}
	out := MarkdownAudience("Widget", map[string]string{"goals": "G", "background": "B", "users": "U"}, proj)
	if !strings.Contains(out, "Widget — Decision view") {
		t.Errorf("title missing: %s", out)
	}
	if !strings.Contains(out, "Projected from the working version") {
		t.Errorf("projection disclosure missing")
	}
	if strings.Contains(out, "Target Users") {
		t.Errorf("non-projected section leaked: %s", out)
	}
	if strings.Index(out, "Goals") > strings.Index(out, "Background") {
		t.Errorf("projection order not respected: %s", out)
	}
	if !strings.Contains(out, "Why you read this: the return") {
		t.Errorf("why annotation missing")
	}
}

func TestMarkdownAudience_MissingSectionCarriesOpenQuestion(t *testing.T) {
	proj, _ := audienceview.Get("decision")
	out := MarkdownAudience("X", map[string]string{}, proj)
	if !strings.Contains(out, "carried as an open question") {
		t.Errorf("blank section should carry an open question: %s", out)
	}
}
