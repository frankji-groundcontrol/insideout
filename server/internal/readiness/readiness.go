// Package readiness discloses per-audience gaps in a working PRD
// (PRODUCT.md: "Readiness is also audience-specific … there is no
// misleading universal '100% complete' score"). Readiness never blocks
// a Commit — its gaps are exactly what a "form a version now" Commit
// carries as unresolved items.
package readiness

// Gap priorities follow PRODUCT.md's coaching contract: must clarify
// now / should clarify this version / validate later.
const (
	MustNow      = "must_clarify_now"
	ShouldNow    = "should_clarify_this_version"
	ValidateLate = "validate_later"
)

type Gap struct {
	Section  string `json:"section"`
	Priority string `json:"priority"`
	// Reason explains who the gap serves and why it matters now
	// (principle 7: explain the question).
	Reason string `json:"reason"`
}

type AudienceReadiness struct {
	Ready           bool     `json:"ready"`
	Gaps            []Gap    `json:"gaps"`
	CarryIntoCommit []string `json:"carryIntoCommit"`
}

// audienceRules maps each audience to the sections it depends on, with
// the reader-facing reason a missing section matters. Deliberately
// small: these are critical questions, not a form schema.
var audienceRules = map[string][]struct {
	Section string
	Reason  string
}{
	"validation": {
		{"background", "the thesis and usable material early readers react to"},
		{"users", "concrete people make feedback specific instead of polite"},
	},
	"decision": {
		{"background", "the why-this, why-now argument a decision maker needs"},
		{"goals", "the investment and return being decided on"},
		{"risks", "the decisions the answer requires now"},
	},
	"management": {
		{"users", "persona paths set priorities"},
		{"goals", "priorities need stated goals"},
		{"requirements", "scope and sequence depend on requirements"},
		{"nonGoals", "scope is only real against non-goals"},
	},
	"delivery": {
		{"requirements", "behavior, rules, and acceptance for engineering and QA"},
		{"constraints", "boundaries and dependencies shape the work"},
	},
}

// validateLater lists sections no audience strictly requires but every
// audience eventually benefits from.
var validateLater = []string{"stories", "constraints", "risks", "nonGoals"}

// Assess discloses each audience's readiness for a section map. A
// section present but blank counts as missing. `ready` means no
// must/should gaps for that audience; validate-later gaps never block.
func Assess(sections map[string]string) map[string]AudienceReadiness {
	out := map[string]AudienceReadiness{}
	blank := blankSections(sections)
	nothingWritten := len(blank) == len(allSections())
	for audience, rules := range audienceRules {
		ar := AudienceReadiness{Ready: true}
		for _, r := range rules {
			if !blank[r.Section] {
				continue
			}
			priority := ShouldNow
			if nothingWritten && r.Section == "background" {
				priority = MustNow
			}
			ar.Gaps = append(ar.Gaps, Gap{Section: r.Section, Priority: priority, Reason: r.Reason})
			ar.Ready = false
			ar.CarryIntoCommit = append(ar.CarryIntoCommit,
				r.Section+": "+r.Reason+" (carried as open question)")
		}
		for _, s := range validateLater {
			if blank[s] && !requiredBy(audience, s) {
				ar.Gaps = append(ar.Gaps, Gap{
					Section: s, Priority: ValidateLate,
					Reason: "useful eventually; not required for this audience",
				})
			}
		}
		if ar.Gaps == nil {
			ar.Gaps = []Gap{}
		}
		if ar.CarryIntoCommit == nil {
			ar.CarryIntoCommit = []string{}
		}
		out[audience] = ar
	}
	return out
}

func blankSections(sections map[string]string) map[string]bool {
	blank := map[string]bool{}
	for _, s := range allSections() {
		v, ok := sections[s]
		blank[s] = !ok || trimLen(v) == 0
	}
	return blank
}

func allSections() []string {
	return []string{"background", "constraints", "goals", "nonGoals", "requirements", "risks", "stories", "users"}
}

func requiredBy(audience, section string) bool {
	for _, r := range audienceRules[audience] {
		if r.Section == section {
			return true
		}
	}
	return false
}

func trimLen(s string) int {
	n := 0
	for _, r := range s {
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			n++
		}
	}
	return n
}
