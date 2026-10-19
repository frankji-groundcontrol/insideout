package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
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

// rvPrdCapSurfaces renders the PRD sections into each prompt that carries them:
// the coach's system prompt, the critic's and the roadmap planner's.
func rvPrdCapSurfaces() map[string]func(sections map[string]string) string {
	return map[string]func(sections map[string]string) string{
		"coach": func(sections map[string]string) string {
			return systemPrompt(localeEN, StageDraft, "A title", sections, "", "")
		},
		"critic": func(sections map[string]string) string {
			stub := &forcingStub{}
			runCritic(context.Background(), stub, localeEN, "A title", sections, ledgerMeta{}, nil)
			return stub.sawSystem
		},
		"planner": func(sections map[string]string) string {
			stub := &forcingStub{}
			_, _ = NewAnthropicRoadmapPlanner(stub).PlanMVP(context.Background(), localeEN, "A title", sections)
			if len(stub.sawMsgs) == 0 {
				return ""
			}
			return stub.sawMsgs[0].Content
		},
	}
}

// The prompts clip a section at the cap, and the comment on that clip says it
// only bites a row stored before the write cap existed. A section the coach
// itself wrote is rendered back to the coach, the critic and the planner by
// those prompts, so it must never come back cut.
func TestRvPrdCapThePromptNeverClipsWhatTheCoachWrote(t *testing.T) {
	st := newTestStore(t)
	_, ex := seedConversation(t, st)

	// Whether the tool refuses is the other test's business; this one reads what was stored.
	_, _ = rvPrdCapWrite(t, ex, "background", 10_500)
	prd, err := st.GetPrdForMember(context.Background(), ex.prdID, ex.actorID)
	if err != nil {
		t.Fatalf("read the PRD back: %v", err)
	}

	for name, render := range rvPrdCapSurfaces() {
		prompt := render(prd.Sections)
		if i := strings.Index(prompt, "[truncated:"); i >= 0 {
			t.Errorf("%s: the section the coach wrote reaches its own prompt cut (%.60q): the cap that stops this is not on the coach's write path", name, prompt[i:])
		}
	}
}

// rvPrdCapPlant stores text in one section of the executor's PRD the way a row
// written before the cap holds it: a plain UPDATE under the author's row-level
// security context, around the write path that now refuses such text.
func rvPrdCapPlant(t *testing.T, st *store.Store, ex *toolExecutor, section, text string) {
	t.Helper()
	ctx := context.Background()
	tx, err := st.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT set_config('app.user_id', $1, true)`, ex.actorID.String()); err != nil {
		t.Fatalf("set the acting user: %v", err)
	}
	tag, err := tx.Exec(ctx,
		`UPDATE insideout.prds SET sections = jsonb_set(sections, ARRAY[$2::text], to_jsonb($3::text)) WHERE id = $1`,
		ex.prdID, section, text)
	if err != nil {
		t.Fatalf("plant the long section: %v", err)
	}
	if tag.RowsAffected() != 1 {
		t.Fatalf("plant the long section: %d rows updated, want 1", tag.RowsAffected())
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
}

// A row stored before the cap is what the clip in the prompts is for. get_prd
// returns it whole, so the coach can shorten it faithfully; a rewrite that is
// still over the cap is refused like any other, one under it is stored, and from
// then on the prompt carries the section uncut.
func TestRvPrdCapCoachCanShortenAnOversizeStoredSection(t *testing.T) {
	st := newTestStore(t)
	_, ex := seedConversation(t, st)
	ctx := context.Background()
	long := strings.Repeat("字", 15_000)
	rvPrdCapPlant(t, st, ex, "background", long)

	out, err := ex.Execute(ctx, ToolCallRequest{Name: "get_prd", Arguments: "{}"})
	if err != nil {
		t.Fatalf("get_prd: %v", err)
	}
	if !strings.Contains(out, long) {
		t.Error("get_prd cut the stored 15000-character section: the coach cannot read all of what it has to shorten")
	}

	if _, err := rvPrdCapWrite(t, ex, "background", 12_000); err == nil {
		t.Error("a rewrite to 12000 characters, still over the cap, was stored")
	}
	if _, err := rvPrdCapWrite(t, ex, "background", 9_000); err != nil {
		t.Fatalf("a rewrite to 9000 characters: %v, want it stored", err)
	}
	prd, err := st.GetPrdForMember(ctx, ex.prdID, ex.actorID)
	if err != nil {
		t.Fatalf("read the PRD back: %v", err)
	}
	if n := utf8.RuneCountInString(prd.Sections["background"]); n != 9_000 {
		t.Errorf("background holds %d characters, want the 9000 just written", n)
	}
	if strings.Contains(formatSectionsForPrompt(prd.Sections), "truncated") {
		t.Error("the prompt still cuts the section after it was shortened under the cap")
	}
}
