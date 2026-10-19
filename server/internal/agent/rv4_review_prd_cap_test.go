package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"
)

// rv4-FR-F85 and rv4-FR-F95: the PRD section cap (10,000 characters) bound only
// the human PATCH. The coach's update_prd_section went straight to the store and
// wrote a section of any length, which every later prompt then clipped while
// get_prd returned it whole. The number is spelled out on purpose: it is the
// documented contract.

// rvPrdCapWrite is one update_prd_section call writing n three-byte characters.
func rvPrdCapWrite(t *testing.T, ex *toolExecutor, section string, n int) (string, error) {
	t.Helper()
	raw, err := json.Marshal(map[string]string{"section": section, "markdown": strings.Repeat("字", n)})
	if err != nil {
		t.Fatal(err)
	}
	return ex.Execute(context.Background(), ToolCallRequest{Name: "update_prd_section", Arguments: string(raw)})
}

// The tool result is the only channel the model reads, so a write over the cap
// must come back as a tool error that says what is wrong and how to proceed,
// and must store nothing.
func TestRvPrdCapCoachWriteOverTheCapIsRefusedAndNamesTheWayOut(t *testing.T) {
	st := newTestStore(t)
	_, ex := seedConversation(t, st)

	// 10,000 three-byte characters are 30,000 bytes: only a character count takes them.
	if _, err := rvPrdCapWrite(t, ex, "background", 10_000); err != nil {
		t.Fatalf("a section of exactly 10000 characters: %v, want it written", err)
	}
	_, err := rvPrdCapWrite(t, ex, "background", 10_001)
	if err == nil {
		t.Error("a 10001-character section was written: the coach can store text its own prompts then clip")
	} else {
		msg := strings.ToLower(err.Error())
		for _, want := range []string{"background", "10000", "10001", "shorten"} {
			if !strings.Contains(msg, want) {
				t.Errorf("the tool error %q does not mention %q: the model is not told what to change", err, want)
			}
		}
	}

	prd, err := st.GetPrdForMember(context.Background(), ex.prdID, ex.actorID)
	if err != nil {
		t.Fatalf("read the PRD back: %v", err)
	}
	if n := utf8.RuneCountInString(prd.Sections["background"]); n != 10_000 {
		t.Errorf("background holds %d characters after the refused write, want the 10000 written before it", n)
	}
}
