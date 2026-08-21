package readiness

import "testing"

func TestAssessEmptyPRD(t *testing.T) {
	out := Assess(nil)
	for _, audience := range []string{"validation", "decision", "management", "delivery"} {
		ar := out[audience]
		if ar.Ready {
			t.Errorf("%s: empty PRD should not be ready", audience)
		}
		if len(ar.CarryIntoCommit) == 0 {
			t.Errorf("%s: gaps must be carryable into a commit", audience)
		}
	}
	// With nothing written at all, the product argument is a must-now gap.
	v := out["validation"]
	found := false
	for _, g := range v.Gaps {
		if g.Section == "background" && g.Priority == MustNow {
			found = true
		}
	}
	if !found {
		t.Errorf("empty PRD should mark background must_clarify_now: %v", v.Gaps)
	}
}

func TestAssessCompletePRD(t *testing.T) {
	full := map[string]string{}
	for _, s := range allSections() {
		full[s] = "written content"
	}
	out := Assess(full)
	for audience, ar := range out {
		if !ar.Ready {
			t.Errorf("%s: complete PRD should be ready, gaps %v", audience, ar.Gaps)
		}
	}
}

func TestBlankCountsAsMissing(t *testing.T) {
	sections := map[string]string{"background": "ok", "users": "   \n"}
	out := Assess(sections)
	v := out["validation"]
	if v.Ready {
		t.Error("blank users should not be ready for validation")
	}
	sawUsers := false
	for _, g := range v.Gaps {
		if g.Section == "users" && g.Priority == ShouldNow {
			sawUsers = true
		}
	}
	if !sawUsers {
		t.Errorf("blank users should be a should-clarify gap: %v", v.Gaps)
	}
}

func TestValidateLaterNeverBlocks(t *testing.T) {
	sections := map[string]string{"background": "x", "users": "y"}
	for _, s := range []string{"goals", "requirements", "risks", "nonGoals", "constraints"} {
		sections[s] = "x"
	}
	// stories missing: validate-later only.
	out := Assess(sections)
	v := out["validation"]
	if !v.Ready {
		t.Errorf("validate-later gaps must not block readiness: %v", v.Gaps)
	}
	foundLate := false
	for _, g := range v.Gaps {
		if g.Section == "stories" {
			if g.Priority != ValidateLate {
				t.Errorf("stories should be validate_later, got %s", g.Priority)
			}
			foundLate = true
		}
	}
	if !foundLate {
		t.Errorf("missing stories should appear as validate-later: %v", v.Gaps)
	}
}

func TestAudienceCoverage(t *testing.T) {
	out := Assess(map[string]string{})
	for _, want := range []string{"validation", "decision", "management", "delivery"} {
		if _, ok := out[want]; !ok {
			t.Errorf("audience %s missing", want)
		}
	}
}
