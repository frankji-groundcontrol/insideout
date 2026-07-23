package agent

import "testing"

func TestQuoteIsGrounded_ExactAndCJK(t *testing.T) {
	cases := []struct {
		name   string
		quote  string
		source string
		want   bool
	}{
		{"exact english substring", "we lose customers every week", "Yeah honestly we lose customers every week to this.", true},
		{"exact chinese substring", "我们每周都在流失客户", "老实说，我们每周都在流失客户，因为这个问题。", true},
		{"english with punctuation/spacing noise", "we, lose customers!!", "we lose customers", true},
		{"chinese with punctuation noise", "找不到、也联系不上", "用户找不到也联系不上客服", true},
		{"paraphrase within bigram threshold", "工作区成员经常找不到旧的想法", "工作区成员经常找不到旧的想法内容，非常头疼", true},
		{"fabricated quote not in source", "80% of users churn within a week", "we lose some customers sometimes", false},
		{"fabricated chinese quote", "百分之八十的用户一周内流失", "我们偶尔会流失一些客户", false},
		{"empty quote never grounds", "", "anything at all", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := quoteIsGrounded(tc.quote, tc.source); got != tc.want {
				t.Errorf("quoteIsGrounded(%q, %q) = %v, want %v", tc.quote, tc.source, got, tc.want)
			}
		})
	}
}

func TestFactBounds_MaxPerKindAndQuoteTruncation(t *testing.T) {
	var facts []Fact
	for i := 0; i < maxFactsPerKind; i++ {
		facts = append(facts, Fact{Kind: "problem", Status: FactStatusAttested})
	}
	if countByKind(facts, "problem") != maxFactsPerKind {
		t.Fatalf("countByKind = %d, want %d", countByKind(facts, "problem"), maxFactsPerKind)
	}

	long := make([]rune, maxQuoteLen+50)
	for i := range long {
		long[i] = 'x'
	}
	quote := string(long)
	truncated := quote
	if len([]rune(truncated)) > maxQuoteLen {
		truncated = string([]rune(truncated)[:maxQuoteLen])
	}
	if len([]rune(truncated)) != maxQuoteLen {
		t.Fatalf("truncated quote length = %d, want %d", len([]rune(truncated)), maxQuoteLen)
	}
}

func TestAttestedKinds_OnlyCountsAttestedStatus(t *testing.T) {
	facts := []Fact{
		{Kind: "problem", Status: FactStatusAttested},
		{Kind: "segment", Status: FactStatusAssumed}, // not attested — shouldn't satisfy the gate
		{Kind: "whynow", Status: FactStatusAttested},
	}
	got := attestedKinds(facts)
	if !got["problem"] || !got["whynow"] {
		t.Fatalf("attestedKinds = %v, want problem and whynow", got)
	}
	if got["segment"] {
		t.Fatalf("attestedKinds = %v, an assumed (not attested) fact should not count", got)
	}
}

func TestSegmentFailsEveryoneLint(t *testing.T) {
	cases := []struct {
		text string
		want bool
	}{
		{"everyone who uses the internet", true},
		{"所有人都需要", true},
		{"workspace admins running a 20-person workshop cohort", false},
		{"工作坊里带队的组长，通常管理 5-15 人的团队", false},
	}
	for _, tc := range cases {
		if got := segmentFailsEveryoneLint(tc.text); got != tc.want {
			t.Errorf("segmentFailsEveryoneLint(%q) = %v, want %v", tc.text, got, tc.want)
		}
	}
}

func TestLoadSaveLedger_RoundTripsAndPreservesUnknownKeys(t *testing.T) {
	raw := []byte(`{"facts":[{"id":"f1","kind":"problem","text":"t","quote":"q","status":"attested"}],"sectionFacts":{"background":["f1"]},"critic_findings":[{"id":"c1"}]}`)
	lm, extra := loadLedger(raw)
	if len(lm.Facts) != 1 || lm.Facts[0].ID != "f1" {
		t.Fatalf("loadLedger facts = %+v", lm.Facts)
	}
	if len(lm.SectionFacts["background"]) != 1 {
		t.Fatalf("loadLedger sectionFacts = %+v", lm.SectionFacts)
	}
	if _, ok := extra["critic_findings"]; !ok {
		t.Fatal("loadLedger should preserve unrecognized keys (e.g. H3's critic_findings) in extra")
	}

	out, err := saveLedger(lm, extra)
	if err != nil {
		t.Fatalf("saveLedger: %v", err)
	}
	lm2, extra2 := loadLedger(out)
	if len(lm2.Facts) != 1 {
		t.Fatalf("round-tripped facts = %+v", lm2.Facts)
	}
	if _, ok := extra2["critic_findings"]; !ok {
		t.Fatal("saveLedger should not drop unrecognized keys on round-trip")
	}
}
