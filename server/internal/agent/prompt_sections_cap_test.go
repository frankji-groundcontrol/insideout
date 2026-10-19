package agent

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"
)

// COMP-04: formatSectionsForPrompt wrote every stored section into the prompt
// whole, so one section saved at the 1 MiB body cap rode into every coach,
// critic and planner call. A section is now cut at contractSectionRunes
// characters with a marker that says how much was cut. The number is the one
// store.UpdateSections refuses at write (store.MaxSectionRunes), written out
// here on purpose: the coach reads exactly what was saved, and the cut only
// bites a row stored before that write cap.
const contractSectionRunes = 10_000

func TestFormatSectionsForPrompt_CutsAHugeSectionAndSaysSo(t *testing.T) {
	huge := strings.Repeat("字", 200_000)
	out := formatSectionsForPrompt(map[string]string{"background": huge, "users": "kept whole"})

	// The eight section lines add under 200 characters around the one section.
	if got := utf8.RuneCountInString(out); got > contractSectionRunes+500 {
		t.Fatalf("a 200000-character section rendered %d characters, want at most %d: the prompt carries it whole", got, contractSectionRunes+500)
	}
	for _, want := range []string{"truncated: 190000 more", "- users: kept whole\n"} {
		if !strings.Contains(out, want) {
			t.Errorf("the rendered sections lack %q:\n%.300s", want, out)
		}
	}
	if !utf8.ValidString(out) {
		t.Error("the cut landed inside a character: the rendered sections are not valid UTF-8")
	}
}

func TestFormatSectionsForPrompt_LeavesASectionAtTheCapWhole(t *testing.T) {
	exact := strings.Repeat("字", contractSectionRunes)
	out := formatSectionsForPrompt(map[string]string{"goals": exact})
	if !strings.Contains(out, "- goals: "+exact+"\n") || strings.Contains(out, "truncated") {
		t.Error("a section of exactly the cap was cut: only longer ones may be")
	}
	if over := formatSectionsForPrompt(map[string]string{"goals": exact + "x"}); !strings.Contains(over, "truncated: 1 more") {
		t.Error("one character over the cap was not cut and marked")
	}
}

// Every prompt that carries the sections routes through the one helper, so
// each surface must grow by no more than the marker when a section goes from
// the cap to 200,000 characters.
func TestPromptSurfaces_DoNotGrowPastTheSectionCap(t *testing.T) {
	surfaces := map[string]func(locale string, sections map[string]string) string{
		"coach": func(locale string, sections map[string]string) string {
			return systemPrompt(locale, StageDraft, "A title", sections, "", "")
		},
		"critic": func(locale string, sections map[string]string) string {
			stub := &forcingStub{}
			runCritic(context.Background(), stub, locale, "A title", sections, ledgerMeta{}, nil)
			return stub.sawSystem
		},
		"planner": func(locale string, sections map[string]string) string {
			stub := &forcingStub{}
			_, _ = NewAnthropicRoadmapPlanner(stub).PlanMVP(context.Background(), locale, "A title", sections)
			if len(stub.sawMsgs) == 0 {
				return ""
			}
			return stub.sawMsgs[0].Content
		},
	}
	for name, render := range surfaces {
		for _, locale := range []string{localeZH, localeEN} {
			at := render(locale, map[string]string{"background": strings.Repeat("字", contractSectionRunes)})
			huge := render(locale, map[string]string{"background": strings.Repeat("字", 200_000)})
			if !strings.Contains(at, strings.Repeat("字", contractSectionRunes)) {
				t.Fatalf("%s/%s: the %d-character section is missing from the prompt, so this test measures nothing", name, locale, contractSectionRunes)
			}
			if extra := utf8.RuneCountInString(huge) - utf8.RuneCountInString(at); extra > 200 {
				t.Errorf("%s/%s: a 200000-character section made the prompt %d characters longer than one at the cap, want only a short marker", name, locale, extra)
			}
		}
	}
}
