package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// LEAD-FENCE-DEDUP: the coach's <prd_data> fence and the MCP tools' fence are one
// rule, and it lived twice: a regexp in this package copied from agentcontext,
// held together by TestFenceDataAgreesWithTheMCPFence over nine fixed strings.
// A rule change made in one copy, on an input outside those nine, would split
// the two fences without a test noticing. The rule has one home now.
func TestFenceRuleHasOneSource(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("glob the package sources: %v (%d files)", err, len(files))
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(src), "\n") {
			if strings.Contains(line, "regexp.MustCompile(") && strings.Contains(line, "prd_data") {
				t.Errorf("%s:%d compiles its own <prd_data> fence pattern; fence through agentcontext.FenceData so there is one rule", f, i+1)
			}
		}
	}
}
