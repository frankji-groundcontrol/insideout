package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// The numbers in trust_facts.go are tunables I chose, not measured limits.
// These tests pin them and the pairs they were calibrated on, so moving either
// is a decision made in a diff rather than by accident.

func TestCheckFactEvidence_TunablesArePinned(t *testing.T) {
	if minAttestedQuoteRunes != 12 || minTextSupport != 0.2 || minLongTextSupport != 0.6 || longTextRatio != 2 {
		t.Fatalf("minAttestedQuoteRunes = %d, minTextSupport = %v, minLongTextSupport = %v, longTextRatio = %v: re-run the calibration pairs below (and the long-text pairs in rv4_review_agent_facts_test.go) and update the changelog before moving them",
			minAttestedQuoteRunes, minTextSupport, minLongTextSupport, longTextRatio)
	}
}

// A rune is not a unit of content: a Han rune counts as two Latin letters.
func TestCheckFactEvidence_QuoteFloor(t *testing.T) {
	cases := []struct {
		name, text, quote, want string // want "" = accepted
	}{
		{"11 letters is a bare phrase", "wanted it now", "we want it now", "too short"},
		{"12 letters is a clause", "wanted it soon", "we want it soon", ""},
		{"spaces and punctuation do not count", "wanted it now", " ...we  want, it: now!!! ", "too short"},
		{"5 Chinese runes are a bare noun phrase", "独立设计师", "独立设计师", "too short"},
		{"6 Chinese runes are a clause", "现在就要", "我们现在就要", ""},
		{"one word", "find", "find", "too short"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkFactEvidence(tc.text, tc.quote)
			switch {
			case tc.want == "" && err != nil:
				t.Fatalf("checkFactEvidence(%q, %q) = %v, want accepted", tc.text, tc.quote, err)
			case tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)):
				t.Fatalf("checkFactEvidence(%q, %q) = %v, want a %q refusal", tc.text, tc.quote, err, tc.want)
			}
		})
	}
}

// The calibration pairs: texts a coach would write for the quote beside them,
// and texts about something else. Support is lexical, so a heavy paraphrase that
// shares few words with its quote is refused too; the model restates closer.
func TestCheckFactEvidence_TextSupport(t *testing.T) {
	type pair struct{ text, quote string }
	supported := []pair{
		{"users lose track of old notes", "nobody can find their old notes once the workshop moves on"},
		{"a 15-person designer workshop cohort", "our workshop cohort of 15 designers, they're the ones stuck with this"},
		{"nothing today", "they just forget, there's no real alternative today"},
		{"customers churn when onboarding is slow", "honestly half of our trial users never finish onboarding, they just drift away after the first day"},
		{"the team has no budget for new tools", "we have basically zero budget for tooling this year, finance already said no"},
		{"launch by the end of Q1", "the board wants something shipped by the end of March at the latest"},
		{"笔记丢失", "我们的设计师团队每周都会丢失大量会议笔记，找不回来"},
		{"目标用户是独立设计师", "我们主要想做给独立设计师用的，他们一个人要同时对接好几个客户"},
		{"现在用微信群来同步进度", "目前大家都是在微信群里互相同步进度，很容易漏掉消息"},
		{"团队在用 Notion 记录需求", "我们现在用 Notion 记需求，但是经常找不到最新版本"},
	}
	unrelated := []pair{
		{"customers want dark mode and offline sync across their devices", "nobody can find their old notes once the workshop moves on"},
		{"enterprise buyers require SOC 2 compliance before purchase", "we have basically zero budget for tooling this year, finance already said no"},
		{"80% of users churn weekly", "we lose a few customers here and there, nothing dramatic"},
		{"market size is 4 billion dollars", "we are building this for freelance designers who juggle five clients at once"},
		{"competitors charge twice as much", "it takes our reviewers three or four days to get back to anyone"},
		{"客户想要深色模式和离线同步功能", "我们的设计师团队每周都会丢失大量会议笔记"},
		{"大型企业要求通过安全合规认证", "我们现在没什么预算，也没有专职的开发人员，只能靠兼职"},
		{"市场规模超过四十亿", "目前大家都是在微信群里互相同步进度，很容易漏掉消息"},
	}
	for _, p := range supported {
		if err := checkFactEvidence(p.text, p.quote); err != nil {
			t.Errorf("text %q under quote %q = %v, want accepted", p.text, p.quote, err)
		}
	}
	for _, p := range unrelated {
		if err := checkFactEvidence(p.text, p.quote); err == nil || !strings.Contains(err.Error(), "not supported") {
			t.Errorf("text %q under quote %q = %v, want a \"not supported\" refusal", p.text, p.quote, err)
		}
	}
}

// Text too short to hold a single n-gram has nothing to compare, so it is not
// refused for that; the quote floor still applies.
func TestCheckFactEvidence_TextTooShortToJudgeIsNotRefused(t *testing.T) {
	if err := checkFactEvidence("ok", "nobody can find their old notes once the workshop moves on"); err != nil {
		t.Fatalf("a two-letter text = %v, want it left to the quote floor", err)
	}
}

// The quote is stored cut at maxQuoteLen, so support is measured against the cut
// quote. Text that only the cut-off tail supports must not pass on its strength.
func TestRecordFact_MeasuresSupportAgainstTheStoredQuote(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	_, ex := seedConversation(t, st)
	quote := strings.Repeat("kiln booking notes get lost ", 11) + "dark mode and offline sync"
	if n := len([]rune(quote)); n <= maxQuoteLen {
		t.Fatalf("fixture premise broken: the quote is %d runes, it must exceed the %d-rune cap", n, maxQuoteLen)
	}
	ex.userMessages = []string{quote}

	args, _ := json.Marshal(map[string]string{"kind": "problem", "text": "dark mode and offline sync", "quote": quote})
	if _, err := ex.recordFact(ctx, string(args)); err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("record_fact whose text only the cut-off tail of the quote supports = %v, want a \"not supported\" refusal", err)
	}
}
