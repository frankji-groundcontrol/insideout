package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestProjectUpdatesAreFiledByMembersWithValidContent(t *testing.T) {
	w := newCoverageWorld(t)
	u := w.note(t, w.member, "  shipped the parser  ")
	if u.AuthorID != w.member.ID.String() || u.Kind != "progress" || u.Content != "shipped the parser" || u.ProjectID != w.project.ID.String() {
		t.Fatalf("filed update = %+v, want the member's trimmed progress note on the project", u)
	}

	// The note reaches both the project's feed and its card on the board.
	feed := decodeAs[projectDetail](t, w.must(t, w.owner, "GET", w.proj(), "", http.StatusOK))
	if len(feed.Updates) != 1 || feed.Updates[0].ID != u.ID {
		t.Fatalf("feed = %+v, want the one note", feed.Updates)
	}
	card := decodeAs[[]projectView](t, w.must(t, w.owner, "GET", w.ws()+"/projects", "", http.StatusOK))[0]
	if card.LatestUpdateKind == nil || *card.LatestUpdateKind != "progress" ||
		card.LatestUpdateContent == nil || *card.LatestUpdateContent != "shipped the parser" || card.LatestUpdateAt == nil {
		t.Fatalf("board card = %+v, want the note as its latest update", card)
	}

	post := func(as routeUser, payload string, want int) {
		t.Helper()
		w.must(t, as, "POST", w.proj()+"/updates", payload, want)
	}
	post(w.outsider, `{"kind":"note","content":"x"}`, http.StatusNotFound) // the project is invisible to them
	post(w.member, `{"kind":"agent_proposal","content":"x"}`, http.StatusBadRequest)
	post(w.member, `{"kind":"","content":"x"}`, http.StatusBadRequest)
	post(w.member, `{"kind":"note","content":"   "}`, http.StatusBadRequest)
	post(w.member, `{"kind":"note"`, http.StatusBadRequest)
	// The bound is 5000 runes, not bytes: 5000 two-byte runes fit, one more does not.
	post(w.member, `{"kind":"note","content":"`+strings.Repeat("é", 5001)+`"}`, http.StatusBadRequest)
	post(w.member, `{"kind":"blocker","content":"`+strings.Repeat("é", 5000)+`"}`, http.StatusCreated)
	w.must(t, w.member, "POST", "/api/v1/projects/"+uuid.NewString()+"/updates", `{"kind":"note","content":"x"}`, http.StatusNotFound)
	w.must(t, w.member, "POST", "/api/v1/projects/not-a-uuid/updates", `{"kind":"note","content":"x"}`, http.StatusBadRequest)

	if feed = decodeAs[projectDetail](t, w.must(t, w.owner, "GET", w.proj(), "", http.StatusOK)); len(feed.Updates) != 2 {
		t.Fatalf("feed has %d updates, want the two accepted ones", len(feed.Updates))
	}
}

// An update is its author's, and a workspace admin may step in; another
// member may not, and a refused attempt leaves it as it was.
func TestProjectUpdateEditsAndDeletesBelongToTheAuthorOrAnAdmin(t *testing.T) {
	w := newCoverageWorld(t)
	other := w.newMember(t, "other")
	mine := w.note(t, w.member, "first draft")
	content := func() (string, bool) {
		t.Helper()
		for _, u := range decodeAs[projectDetail](t, w.must(t, w.owner, "GET", w.proj(), "", http.StatusOK)).Updates {
			if u.ID == mine.ID {
				return u.Content, true
			}
		}
		return "", false
	}
	edit := func(as routeUser, id, payload string, want int) string {
		t.Helper()
		return w.must(t, as, "PATCH", "/api/v1/updates/"+id, payload, want)
	}
	remove := func(as routeUser, id string, want int) {
		t.Helper()
		w.must(t, as, "DELETE", "/api/v1/updates/"+id, "", want)
	}

	edit(other, mine.ID, `{"content":"hijacked"}`, http.StatusForbidden)
	edit(w.outsider, mine.ID, `{"content":"hijacked"}`, http.StatusNotFound)
	edit(w.member, mine.ID, `{"content":"   "}`, http.StatusBadRequest)
	edit(w.member, mine.ID, `{"content":"`+strings.Repeat("x", 5001)+`"}`, http.StatusBadRequest)
	edit(w.member, mine.ID, `{"content":"x","kind":"note"}`, http.StatusBadRequest)
	edit(w.member, uuid.NewString(), `{"content":"x"}`, http.StatusNotFound)
	edit(w.member, "not-a-uuid", `{"content":"x"}`, http.StatusBadRequest)
	if got, _ := content(); got != "first draft" {
		t.Fatalf("refused edits left the update as %q, want it untouched", got)
	}

	got := decodeAs[projectUpdateView](t, edit(w.member, mine.ID, `{"content":" second draft "}`, http.StatusOK))
	if got.Content != "second draft" || got.AuthorID != w.member.ID.String() {
		t.Fatalf("author's edit = %+v, want the trimmed text, still theirs", got)
	}
	edit(w.owner, mine.ID, `{"content":"edited by the admin"}`, http.StatusOK)
	if got, _ := content(); got != "edited by the admin" {
		t.Fatalf("update reads %q after the admin's edit", got)
	}

	remove(other, mine.ID, http.StatusForbidden)
	remove(w.outsider, mine.ID, http.StatusNotFound)
	remove(w.member, uuid.NewString(), http.StatusNotFound)
	remove(w.member, "not-a-uuid", http.StatusBadRequest)
	if _, there := content(); !there {
		t.Fatal("a refused delete removed the update")
	}
	remove(w.member, mine.ID, http.StatusOK)
	remove(w.member, mine.ID, http.StatusNotFound)

	theirs := w.note(t, other, "someone else's note")
	remove(w.owner, theirs.ID, http.StatusOK) // an admin may remove another member's
	if feed := decodeAs[projectDetail](t, w.must(t, w.owner, "GET", w.proj(), "", http.StatusOK)); len(feed.Updates) != 0 {
		t.Fatalf("feed = %+v after both deletes, want it empty", feed.Updates)
	}
}
