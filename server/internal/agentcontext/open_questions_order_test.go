package agentcontext

import (
	"slices"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/readiness"
)

// rv4-QUAL-01: the brainstorming context's openQuestions came from a map
// iteration, so the same PRD yielded a different order on every call, and its
// notion of "blank" (== "") disagreed with readiness (whitespace-only and
// absent sections count as blank there). The coach's open questions and the
// readiness gaps must agree on both the rule and the order.

// canonical is the order readiness lists PRD sections in.
var canonical = []string{"background", "constraints", "goals", "nonGoals", "requirements", "risks", "stories", "users"}

func openQuestions(sections map[string]string) []string {
	out := Assemble(Inputs{Mode: "brainstorming", PrdSections: sections})
	return out["openQuestions"].([]string)
}

// written is a PRD with every section filled in, then adjusted: set
// overrides a section's text, drop removes the key altogether.
func written(set map[string]string, drop ...string) map[string]string {
	m := map[string]string{}
	for _, s := range canonical {
		m[s] = "text"
	}
	for k, v := range set {
		m[k] = v
	}
	for _, k := range drop {
		delete(m, k)
	}
	return m
}

// gapSections is every section readiness reports as a gap for some audience,
// in canonical order.
func gapSections(sections map[string]string) []string {
	seen := map[string]bool{}
	for _, ar := range readiness.Assess(sections) {
		for _, g := range ar.Gaps {
			seen[g.Section] = true
		}
	}
	var out []string
	for _, s := range canonical {
		if seen[s] {
			out = append(out, s)
		}
	}
	return out
}

func TestOpenQuestionsOrderIsStable(t *testing.T) {
	allBlank := map[string]string{}
	for _, s := range canonical {
		allBlank[s] = ""
	}
	// The reviewer saw eight distinct orders in twenty calls of the old map
	// iteration; fifty calls make an accidental pass vanishingly unlikely.
	for i := 0; i < 50; i++ {
		if got := openQuestions(allBlank); !slices.Equal(got, canonical) {
			t.Fatalf("call %d: open questions = %v, want the canonical order %v", i, got, canonical)
		}
	}
}

func TestOpenQuestionsUseTheReadinessBlankRule(t *testing.T) {
	cases := []struct {
		name     string
		sections map[string]string
		want     []string
	}{
		{"whitespace-only is blank", written(map[string]string{"goals": " \n\t "}), []string{"goals"}},
		{"an absent section is blank", written(nil, "users"), []string{"users"}},
		{"several blanks keep the canonical order", written(map[string]string{"risks": ""}, "background", "stories"),
			[]string{"background", "risks", "stories"}},
		{"no PRD sections at all: everything is open", nil, canonical},
		{"every section written: nothing is open", written(nil), nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := openQuestions(c.sections)
			if !slices.Equal(got, c.want) {
				t.Errorf("open questions = %v, want %v", got, c.want)
			}
			// The same PRD, seen through readiness: every open question is a
			// gap for some audience and every gap is an open question.
			if gaps := gapSections(c.sections); !slices.Equal(gaps, got) {
				t.Errorf("readiness gaps = %v, open questions = %v, want them to agree", gaps, got)
			}
		})
	}
}
