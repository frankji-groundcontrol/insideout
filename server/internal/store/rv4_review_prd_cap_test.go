package store

import (
	"context"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"
)

// rv4-FR-F85 and rv4-FR-F95: the 10,000-character PRD section cap (rv4-COMP-04)
// was checked only by the human PATCH handler. UpdateSections is the one writer
// of prds.sections and had no limit of its own, so the coach's update_prd_section
// tool stored a section of any length, the next prompt then clipped what the
// coach had just written, and get_prd still returned it whole.
//
// The number is spelled out in every test on purpose: it is the documented
// contract, and raising it is a decision for the docs, not for a constant.

// rvPrdCapRunes is n copies of one three-byte character, so a byte count would
// refuse text that a character count takes.
func rvPrdCapRunes(n int) string { return strings.Repeat("字", n) }

// rvPrdCapDraft is an author and a fresh PRD of theirs, on real PostgreSQL.
func rvPrdCapDraft(t *testing.T, st *Store) (*User, *Prd) {
	t.Helper()
	ctx := context.Background()
	author := mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, author.ID, "Section cap WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	idea, err := st.CreateIdea(ctx, author.ID, ws.ID, "an idea", "content")
	if err != nil {
		t.Fatalf("create idea: %v", err)
	}
	prd, _, err := st.ConvertIdea(ctx, author.ID, idea.ID)
	if err != nil {
		t.Fatalf("convert idea: %v", err)
	}
	return author, prd
}

// Every writer of a section goes through UpdateSections, so the store is where
// the cap has to live. The write at exactly the cap is stored, the one a
// character over is refused as a validation error that names the section and
// the limit, and the refused write leaves the stored text as it was.
func TestRvPrdCapUpdateSectionsRefusesAnOverlongSection(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	author, prd := rvPrdCapDraft(t, st)

	for _, key := range PrdSectionKeys {
		// 10,000 three-byte characters are 30,000 bytes: only a character count takes them.
		if _, err := st.UpdateSections(ctx, author.ID, prd.ID, nil, map[string]string{key: rvPrdCapRunes(10_000)}, nil); err != nil {
			t.Errorf("%s at exactly 10000 characters: %v, want it stored", key, err)
			continue
		}
		_, err := st.UpdateSections(ctx, author.ID, prd.ID, nil, map[string]string{key: rvPrdCapRunes(10_001)}, nil)
		switch {
		case !errors.Is(err, ErrValidation):
			t.Errorf("%s at 10001 characters: err = %v, want a validation error: the store accepted it", key, err)
		case !strings.Contains(err.Error(), key) || !strings.Contains(err.Error(), "10000"):
			t.Errorf("%s at 10001 characters: %q names neither the section nor the limit", key, err)
		}
		got, err := st.GetPrdForMember(ctx, prd.ID, author.ID)
		if err != nil {
			t.Fatalf("read the PRD back: %v", err)
		}
		if n := utf8.RuneCountInString(got.Sections[key]); n != 10_000 {
			t.Errorf("%s holds %d characters after the 10001-character write, want the 10000 stored before it", key, n)
		}
	}
}

// One section over the cap refuses the whole call: the other sections and the
// title in the same call are not stored either.
func TestRvPrdCapARefusedPatchStoresNothing(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	author, prd := rvPrdCapDraft(t, st)
	renamed := "renamed in the same call"

	_, err := st.UpdateSections(ctx, author.ID, prd.ID, &renamed,
		map[string]string{"goals": "a fine goal", "risks": rvPrdCapRunes(10_001)}, nil)
	if !errors.Is(err, ErrValidation) {
		t.Errorf("a patch with a 10001-character section: err = %v, want a validation error", err)
	}
	got, err := st.GetPrdForMember(ctx, prd.ID, author.ID)
	if err != nil {
		t.Fatalf("read the PRD back: %v", err)
	}
	if got.Sections["goals"] != "" || got.Sections["risks"] != "" || got.Title != prd.Title {
		t.Errorf("the refused call stored goals=%q risks=%d characters title=%q, want none of it",
			got.Sections["goals"], utf8.RuneCountInString(got.Sections["risks"]), got.Title)
	}
}

// The coach calls UpdateSections with the revision it read as the CAS guard.
// A fresh revision must not be a way around the cap.
func TestRvPrdCapTheCoachCallShapeIsCappedToo(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	author, prd := rvPrdCapDraft(t, st)

	cur, err := st.GetPrdForMember(ctx, prd.ID, author.ID)
	if err != nil {
		t.Fatalf("read the PRD: %v", err)
	}
	_, err = st.UpdateSections(ctx, author.ID, prd.ID, nil, map[string]string{"background": rvPrdCapRunes(10_001)}, &cur.UpdatedAt)
	if !errors.Is(err, ErrValidation) {
		t.Errorf("a 10001-character write with a fresh revision: err = %v, want a validation error", err)
	}
}

// The documented cap, spelled out. Raising it is a decision for the docs.
func TestRvPrdCapTheCapIsTenThousandCharacters(t *testing.T) {
	if got := MaxSectionRunes; got != 10_000 {
		t.Fatalf("MaxSectionRunes = %d, want 10000: the cap is the documented contract", got)
	}
}

// The refusal is typed: the api answers 400 with its text, which is what the
// PRD page shows the user, and the coach tool builds its message from the
// section it names. With two sections over the cap the answer must not change
// with map order.
func TestRvPrdCapRefusalIsTypedAndNamesTheSection(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	author, prd := rvPrdCapDraft(t, st)
	patch := map[string]string{"risks": rvPrdCapRunes(10_001), "goals": rvPrdCapRunes(10_001), "users": "fine"}

	first := ""
	for i := 0; i < 20; i++ {
		_, err := st.UpdateSections(ctx, author.ID, prd.ID, nil, patch, nil)
		var tooLong *SectionTooLongError
		if !errors.As(err, &tooLong) || !errors.Is(err, ErrValidation) {
			t.Fatalf("err = %v, want a *SectionTooLongError that is an ErrValidation", err)
		}
		if want := "section " + tooLong.Section + " is too long (max 10000 characters)"; err.Error() != want {
			t.Fatalf("message = %q, want %q", err.Error(), want)
		}
		if first == "" {
			first = tooLong.Section
		}
		if tooLong.Section != first || (first != "goals" && first != "risks") {
			t.Fatalf("call %d names %q, the first call named %q: the answer changes with map order or names a section that fits", i, tooLong.Section, first)
		}
	}
}
