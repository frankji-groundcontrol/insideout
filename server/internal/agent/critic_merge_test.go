package agent

import "testing"

func TestMergeRankedCritique(t *testing.T) {
	stored := []CriticFinding{
		{ID: "c1", Section: "goals", Severity: "blocking", Issue: "no measurable objective", Status: FindingOpen},
		{ID: "c2", Section: "users", Severity: "minor", Issue: "audience too broad", Status: FindingOverridden},
	}

	t.Run("matched line keeps stored id and status", func(t *testing.T) {
		restated := []CriticFinding{
			{ID: "cnew1", Section: "users", Severity: "minor", Issue: "audience too broad。", Status: FindingOpen},
		}
		merged := mergeRankedCritique(stored, restated)
		if len(merged) != 2 {
			t.Fatalf("got %d findings, want 2: %+v", len(merged), merged)
		}
		if merged[1].ID != "c2" || merged[1].Status != FindingOverridden {
			t.Fatalf("restated line must keep the stored id and status, got %+v", merged[1])
		}
	})

	t.Run("omitted stored finding survives", func(t *testing.T) {
		restated := []CriticFinding{
			{ID: "cnew1", Section: "users", Severity: "minor", Issue: "audience too broad", Status: FindingOpen},
		}
		merged := mergeRankedCritique(stored, restated)
		if merged[0].ID != "c1" || merged[0].Status != FindingOpen {
			t.Fatalf("omitted open blocking finding must survive, got %+v", merged[0])
		}
	})

	t.Run("genuinely new line is added", func(t *testing.T) {
		restated := []CriticFinding{
			{ID: "cnew1", Section: "risks", Severity: "major", Issue: "no risks listed", Status: FindingOpen},
		}
		merged := mergeRankedCritique(stored, restated)
		if len(merged) != 3 || merged[2].ID != "cnew1" {
			t.Fatalf("new line should be appended as-is, got %+v", merged)
		}
	})

	t.Run("same issue in another section or severity is new", func(t *testing.T) {
		restated := []CriticFinding{
			{ID: "cnew1", Section: "goals", Severity: "minor", Issue: "no measurable objective", Status: FindingOpen},
			{ID: "cnew2", Section: "users", Severity: "blocking", Issue: "audience too broad", Status: FindingOpen},
		}
		merged := mergeRankedCritique(stored, restated)
		if len(merged) != 4 {
			t.Fatalf("section or severity change means a different finding, got %+v", merged)
		}
	})
}
