package export

import (
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
