package agent

import (
	"encoding/json"
	"testing"
)

func sectionsFixture() map[string]string {
	return map[string]string{
		"background":   "our workshop cohort loses track of old ideas once the thread scrolls past",
		"users":        "everyone who has ever used the internet",
		"goals":        "make things better",
		"nonGoals":     "",
		"stories":      "",
		"requirements": "add a button that saves ideas",
		"constraints":  "",
		"risks":        "",
	}
}

func TestParseCriticFindings_DropsUngroundedDefectQuote(t *testing.T) {
	args, _ := json.Marshal(map[string]any{
		"findings": []map[string]any{
			{"section": "background", "severity": "major", "kind": "defect",
				"quote": "our workshop cohort loses track of old ideas", "issue": "grounded, should survive"},
			{"section": "goals", "severity": "major", "kind": "defect",
				"quote": "this completely fabricated sentence never appeared anywhere", "issue": "fabricated, should be dropped"},
		},
	})
	out, err := parseCriticFindings(string(args), sectionsFixture())
	if err != nil {
		t.Fatalf("parseCriticFindings: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("got %d findings, want 1 (the fabricated-quote one dropped): %+v", len(out), out)
	}
	if out[0].Issue != "grounded, should survive" {
		t.Fatalf("wrong finding survived: %+v", out[0])
	}
}

func TestParseCriticFindings_OmissionNeedsNoQuote(t *testing.T) {
	args, _ := json.Marshal(map[string]any{
		"findings": []map[string]any{
			{"section": "goals", "severity": "blocking", "kind": "omission", "issue": "no measurable objective stated"},
		},
	})
	out, err := parseCriticFindings(string(args), sectionsFixture())
	if err != nil {
		t.Fatalf("parseCriticFindings: %v", err)
	}
	if len(out) != 1 || out[0].Kind != "omission" {
		t.Fatalf("omission finding without a quote should survive: %+v", out)
	}
	if out[0].Status != FindingOpen {
		t.Fatalf("new findings should start open, got %q", out[0].Status)
	}
	if out[0].ID == "" {
		t.Fatal("finding should get an assigned id")
	}
}

func TestParseCriticFindings_RejectsBadSeverityAndKind(t *testing.T) {
	args, _ := json.Marshal(map[string]any{
		"findings": []map[string]any{
			{"section": "goals", "severity": "catastrophic", "kind": "omission", "issue": "bad severity"},
			{"section": "goals", "severity": "minor", "kind": "vibes", "issue": "bad kind"},
			{"section": "goals", "severity": "minor", "kind": "omission", "issue": ""}, // empty issue
		},
	})
	out, err := parseCriticFindings(string(args), sectionsFixture())
	if err != nil {
		t.Fatalf("parseCriticFindings: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("all three findings are malformed and should be dropped, got %+v", out)
	}
}

func TestParseCriticFindings_InvalidJSON(t *testing.T) {
	if _, err := parseCriticFindings("not json", sectionsFixture()); err == nil {
		t.Fatal("invalid JSON should error, not silently return zero findings")
	}
}

func TestHasOpenBlockingFindings(t *testing.T) {
	findings := []CriticFinding{
		{ID: "c1", Severity: "blocking", Status: FindingOpen},
		{ID: "c2", Severity: "minor", Status: FindingOpen},
	}
	if !hasOpenBlockingFindings(findings) {
		t.Fatal("an open blocking finding should gate finalize")
	}

	findings[0].Status = FindingResolved
	if hasOpenBlockingFindings(findings) {
		t.Fatal("a resolved blocking finding should no longer gate finalize")
	}

	findings[0].Status = FindingOverridden
	if hasOpenBlockingFindings(findings) {
		t.Fatal("an overridden blocking finding should no longer gate finalize")
	}
}

func TestCriticState_RoundTrip(t *testing.T) {
	extra := map[string]json.RawMessage{}
	cs := criticState{RoundCount: 1, Skipped: "contention"}
	saveCriticState(extra, cs)

	got := loadCriticState(extra)
	if got.RoundCount != 1 || got.Skipped != "contention" {
		t.Fatalf("round-tripped state = %+v, want %+v", got, cs)
	}
}

func TestCriticFindings_RoundTripThroughLedgerMeta(t *testing.T) {
	// The critic findings and the fact ledger share the same
	// agent_conversations.meta blob via the extra-keys mechanism —
	// verify one doesn't clobber the other.
	lm := ledgerMeta{Facts: []Fact{{ID: "f1", Kind: "problem", Status: FactStatusAttested}}}
	extra := map[string]json.RawMessage{}
	saveCriticFindings(extra, []CriticFinding{{ID: "c1", Severity: "minor", Status: FindingOpen}})

	raw, err := saveLedger(lm, extra)
	if err != nil {
		t.Fatalf("saveLedger: %v", err)
	}
	lm2, extra2 := loadLedger(raw)
	if len(lm2.Facts) != 1 {
		t.Fatalf("facts lost in round trip: %+v", lm2.Facts)
	}
	if findings := loadCriticFindings(extra2); len(findings) != 1 || findings[0].ID != "c1" {
		t.Fatalf("critic findings lost in round trip: %+v", findings)
	}
}
