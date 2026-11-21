package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// rv5-PARB-09, PARA-11: SKILL.md showed `projects create <ws-id> --title T`,
// which exits 2 — Go stops reading flags at the first positional argument. Every
// CLI example in the pages agents follow must put its flags before the ids. A
// "<placeholder>" right after a flag is that flag's value, not a positional.
func TestDocumentedCommandsPutFlagsBeforeIDs(t *testing.T) {
	verbs := map[string]bool{}
	for _, v := range cliVerbs {
		verbs[v.name] = true
	}
	span := regexp.MustCompile("`([^`]+)`")
	for _, doc := range []string{"SKILL.md", "docs/usage/cli-mcp-and-guidance.md"} {
		raw, err := os.ReadFile(filepath.Join("..", "..", "..", doc))
		if err != nil {
			t.Fatal(err)
		}
		var spans [][]string // per line: a fence's three backticks would pair spans across lines
		for _, line := range strings.Split(string(raw), "\n") {
			spans = append(spans, span.FindAllStringSubmatch(line, -1)...)
		}
		for _, m := range spans {
			words := strings.Fields(strings.TrimPrefix(m[1], "insideout "))
			if len(words) < 2 || !verbs[words[0]] {
				continue
			}
			positional := false
			for i, w := range words {
				isValue := i > 0 && strings.HasPrefix(strings.TrimLeft(words[i-1], "["), "-")
				if strings.HasPrefix(w, "<") && !isValue {
					positional = true
				}
				if positional && strings.HasPrefix(strings.TrimLeft(w, "["), "--") {
					t.Errorf("%s: `%s` puts %s after an id; the CLI exits 2", doc, m[1], w)
					break
				}
			}
		}
	}
}

// rv5-PARA-11: the help footer names the saved login, which every client verb
// reads after INSIDEOUT_TOKEN, not INSIDEOUT_TOKEN alone.
func TestHelpFooterNamesTheSavedLogin(t *testing.T) {
	var b strings.Builder
	printHelp(&b)
	if !strings.Contains(b.String(), "insideout login") {
		t.Errorf("help ends %q, want it to name the saved `insideout login`", b.String()[max(0, b.Len()-200):])
	}
}
