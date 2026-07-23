// Package export renders a PRD's sections to markdown or print-ready HTML
// on demand — no job model, no object storage (D8 in
// docs/plans/2026-07-20-go-rewrite/README.md), unlike the old edge
// function's synchronous-but-job-shaped export.
package export

import (
	"fmt"
	"html"
	"strings"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// sectionLabels gives each fixed PRD section key a human-readable
// heading, in the same order as store.PrdSectionKeys.
var sectionLabels = map[string]string{
	"background":   "Background & Problem Statement",
	"users":        "Target Users & Personas",
	"goals":        "Goals & Success Metrics",
	"nonGoals":     "Non-Goals",
	"stories":      "User Stories & Scenarios",
	"requirements": "Functional Requirements",
	"constraints":  "Constraints & Dependencies",
	"risks":        "Risks & Open Questions",
}

func Markdown(title string, sections map[string]string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", title)
	for _, key := range store.PrdSectionKeys {
		content := strings.TrimSpace(sections[key])
		fmt.Fprintf(&b, "## %s\n\n", sectionLabels[key])
		if content == "" {
			b.WriteString("_(not yet written)_\n\n")
			continue
		}
		fmt.Fprintf(&b, "%s\n\n", content)
	}
	return strings.TrimRight(b.String(), "\n") + "\n"
}

// PrintHTML renders real (escaped, paragraph-wrapped) HTML for
// browser print-to-PDF — unlike the old export function, this is not
// escaped-markdown dumped into a single <pre> block.
func PrintHTML(title string, sections map[string]string) string {
	var body strings.Builder
	fmt.Fprintf(&body, "<h1>%s</h1>\n", html.EscapeString(title))
	for _, key := range store.PrdSectionKeys {
		fmt.Fprintf(&body, "<h2>%s</h2>\n", html.EscapeString(sectionLabels[key]))
		content := strings.TrimSpace(sections[key])
		if content == "" {
			body.WriteString("<p><em>(not yet written)</em></p>\n")
			continue
		}
		for _, para := range strings.Split(content, "\n\n") {
			para = strings.TrimSpace(para)
			if para == "" {
				continue
			}
			escaped := html.EscapeString(para)
			escaped = strings.ReplaceAll(escaped, "\n", "<br>")
			fmt.Fprintf(&body, "<p>%s</p>\n", escaped)
		}
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>%s</title>
<style>
  body { font-family: -apple-system, "Noto Sans SC", sans-serif; max-width: 800px; margin: 2rem auto; padding: 0 1rem; line-height: 1.6; }
  h1 { font-size: 1.75rem; }
  h2 { font-size: 1.25rem; margin-top: 2rem; border-bottom: 1px solid #ddd; padding-bottom: 0.25rem; }
  @media print { body { margin: 0; max-width: none; } }
</style>
</head>
<body>
%s</body>
</html>
`, html.EscapeString(title), body.String())
}
