// Package audienceview defines the audience projections of one PRD
// core (PRODUCT.md "One PRD core, multiple audience views"): each
// audience reads a chosen, ordered subset of the same sections — a
// projection, never a separately maintained document.
package audienceview

import "fmt"

// SectionPick is one section in an audience's reading order, with why
// that audience reads it.
type SectionPick struct {
	Key string `json:"section"`
	Why string `json:"why"`
}

// Projection is an audience's lens over the PRD core.
type Projection struct {
	Audience string        `json:"audience"`
	Title    string        `json:"title"`
	Purpose  string        `json:"purpose"`
	Sections []SectionPick `json:"sections"`
}

var projections = map[string]Projection{
	"decision": {
		Audience: "decision",
		Title:    "Decision view",
		Purpose:  "For a boss or decision maker: why, why now, investment and return, risks, and the decisions required.",
		Sections: []SectionPick{
			{"background", "the why-this, why-now argument"},
			{"goals", "the investment and the return being decided on"},
			{"nonGoals", "what this bet deliberately does not include"},
			{"risks", "the decisions the answer requires now"},
		},
	},
	"management": {
		Audience: "management",
		Title:    "Management view",
		Purpose:  "For a product manager and team: personas, paths, priorities, scope, dependencies, assumptions, and sequence.",
		Sections: []SectionPick{
			{"users", "concrete people and their paths"},
			{"goals", "priorities against stated goals"},
			{"requirements", "scope and sequence"},
			{"nonGoals", "the scope boundary"},
			{"constraints", "dependencies and assumptions"},
		},
	},
	"delivery": {
		Audience: "delivery",
		Title:    "Delivery view",
		Purpose:  "For engineering and QA: behavior, states, rules, boundaries, acceptance, dependencies, and evidence.",
		Sections: []SectionPick{
			{"requirements", "behavior, rules, and acceptance"},
			{"constraints", "boundaries and dependencies"},
			{"stories", "scenarios to build and test against"},
		},
	},
	"validation": {
		Audience: "validation",
		Title:    "Co-creation & validation view",
		Purpose:  "For early users, friends, and advisers: the thesis, usable material, uncertain assumptions, and requested feedback.",
		Sections: []SectionPick{
			{"background", "the thesis and usable material to react to"},
			{"users", "who this is for"},
			{"stories", "what they can try"},
			{"risks", "uncertain assumptions and requested feedback"},
		},
	},
}

// Audiences lists the valid audience keys.
func Audiences() []string { return []string{"decision", "management", "delivery", "validation"} }

// Valid reports whether the audience key exists.
func Valid(audience string) bool { _, ok := projections[audience]; return ok }

// Get returns an audience's projection.
func Get(audience string) (Projection, error) {
	p, ok := projections[audience]
	if !ok {
		return Projection{}, fmt.Errorf("audienceview: unknown audience %q", audience)
	}
	return p, nil
}
