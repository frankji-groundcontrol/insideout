package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// Ideas are a personal inbox (USER-05). The list is the caller's own; the
// workspace admin can read and drop another member's idea but not edit or
// convert it; a fellow member is told the idea does not exist.

func TestIdeaInboxIsListedAndReadPerAuthor(t *testing.T) {
	w := newCoverageWorld(t)
	idea := w.fileIdea(t, w.member, "  Member's idea  ")
	if idea.Title != "Member's idea" || idea.AuthorID != w.member.ID.String() || idea.Status != "inbox" ||
		idea.PrdID != nil || idea.WorkspaceID != w.project.WorkspaceID.String() {
		t.Fatalf("filed idea = %+v, want a trimmed title, the member as author, in the inbox", idea)
	}
	path := "/api/v1/ideas/" + idea.ID

	mine := decodeAs[[]ideaView](t, w.must(t, w.member, "GET", w.ws()+"/ideas", "", http.StatusOK))
	if len(mine) != 1 || mine[0].ID != idea.ID {
		t.Fatalf("the author's inbox = %+v, want exactly the idea", mine)
	}
	// An admin's inbox is their own, so it is empty rather than the workspace's.
	if body := w.must(t, w.owner, "GET", w.ws()+"/ideas", "", http.StatusOK); strings.TrimSpace(body) != "[]" {
		t.Fatalf("the admin's inbox = %s, want an empty list, not the member's idea", body)
	}
	w.must(t, w.outsider, "GET", w.ws()+"/ideas", "", http.StatusNotFound)
	w.must(t, w.member, "GET", "/api/v1/workspaces/not-a-uuid/ideas", "", http.StatusBadRequest)

	for name, as := range map[string]routeUser{"the author": w.member, "a workspace admin": w.owner} {
		if got := decodeAs[ideaView](t, w.must(t, as, "GET", path, "", http.StatusOK)); got.ID != idea.ID {
			t.Fatalf("%s read %+v, want the idea", name, got)
		}
	}
	other := w.newMember(t, "other")
	w.must(t, other, "GET", path, "", http.StatusNotFound) // existence is not disclosed
	w.must(t, w.outsider, "GET", path, "", http.StatusNotFound)
	w.must(t, w.member, "GET", "/api/v1/ideas/"+uuid.NewString(), "", http.StatusNotFound)
	w.must(t, w.member, "GET", "/api/v1/ideas/not-a-uuid", "", http.StatusBadRequest)
}

func TestIdeasAreCreatedByMembersWithATitle(t *testing.T) {
	w := newCoverageWorld(t)
	w.must(t, w.outsider, "POST", w.ws()+"/ideas", `{"title":"Sneaked in"}`, http.StatusForbidden)
	for name, payload := range map[string]string{
		"blank title":    `{"title":"   ","content":"c"}`,
		"title too long": `{"title":"` + strings.Repeat("x", 201) + `"}`,
		"not json":       `title=T`,
		"unknown field":  `{"title":"T","status":"converted"}`,
	} {
		t.Run(name, func(t *testing.T) { w.must(t, w.member, "POST", w.ws()+"/ideas", payload, http.StatusBadRequest) })
	}
	// Content is optional; the body is not interpreted, so a title that merely
	// looks like markup comes back as it went in.
	made := decodeAs[ideaView](t, w.must(t, w.member, "POST", w.ws()+"/ideas", `{"title":"<b>bold</b>"}`, http.StatusCreated))
	if made.Content != "" || made.Title != "<b>bold</b>" {
		t.Fatalf("idea without content = %+v, want empty content and the title verbatim", made)
	}
	if n := len(decodeAs[[]ideaView](t, w.must(t, w.member, "GET", w.ws()+"/ideas", "", http.StatusOK))); n != 1 {
		t.Fatalf("inbox has %d ideas, want only the accepted one", n)
	}
}

func TestIdeaEditsBelongToItsAuthorAlone(t *testing.T) {
	w := newCoverageWorld(t)
	idea := w.fileIdea(t, w.member, "First draft")
	path := "/api/v1/ideas/" + idea.ID
	other := w.newMember(t, "other")
	const edit = `{"title":"Second draft","content":"more"}`

	w.must(t, w.owner, "PATCH", path, edit, http.StatusForbidden) // an admin can read it, not rewrite it
	w.must(t, other, "PATCH", path, edit, http.StatusNotFound)
	w.must(t, w.outsider, "PATCH", path, edit, http.StatusNotFound)
	w.must(t, w.member, "PATCH", "/api/v1/ideas/"+uuid.NewString(), edit, http.StatusNotFound)
	w.must(t, w.member, "PATCH", "/api/v1/ideas/not-a-uuid", edit, http.StatusBadRequest)
	for name, payload := range map[string]string{
		"blank title":    `{"title":" "}`,
		"title too long": `{"title":"` + strings.Repeat("x", 201) + `"}`,
		"not json":       `title=T`,
	} {
		t.Run(name, func(t *testing.T) { w.must(t, w.member, "PATCH", path, payload, http.StatusBadRequest) })
	}
	if got := decodeAs[ideaView](t, w.must(t, w.member, "GET", path, "", http.StatusOK)); got.Title != "First draft" || got.Status != "inbox" {
		t.Fatalf("refused edits left the idea as %+v, want it untouched", got)
	}

	// Refining an idea moves it out of the inbox state.
	got := decodeAs[ideaView](t, w.must(t, w.member, "PATCH", path, edit, http.StatusOK))
	if got.Title != "Second draft" || got.Content != "more" || got.Status != "refining" || got.AuthorID != w.member.ID.String() {
		t.Fatalf("edited idea = %+v, want the new text, refining, still the member's", got)
	}
}

// Dropping is the author's, or an admin's; a dropped idea leaves the inbox but
// is kept (soft delete), so the author can still open it.
func TestIdeaDropsAreForTheAuthorOrAnAdmin(t *testing.T) {
	w := newCoverageWorld(t)
	idea := w.fileIdea(t, w.member, "To drop")
	path := "/api/v1/ideas/" + idea.ID
	other := w.newMember(t, "other")
	inbox := func(as routeUser) int {
		t.Helper()
		return len(decodeAs[[]ideaView](t, w.must(t, as, "GET", w.ws()+"/ideas", "", http.StatusOK)))
	}

	w.must(t, other, "DELETE", path, "", http.StatusNotFound)
	w.must(t, w.outsider, "DELETE", path, "", http.StatusNotFound)
	w.must(t, w.member, "DELETE", "/api/v1/ideas/"+uuid.NewString(), "", http.StatusNotFound)
	w.must(t, w.member, "DELETE", "/api/v1/ideas/not-a-uuid", "", http.StatusBadRequest)
	if inbox(w.member) != 1 {
		t.Fatal("a refused drop took the idea out of the inbox")
	}

	w.must(t, w.member, "DELETE", path, "", http.StatusOK)
	if inbox(w.member) != 0 {
		t.Fatal("a dropped idea is still in the inbox")
	}
	if got := decodeAs[ideaView](t, w.must(t, w.member, "GET", path, "", http.StatusOK)); got.Status != "dropped" {
		t.Fatalf("dropped idea reads as %q, want it kept with status dropped", got.Status)
	}

	again := w.fileIdea(t, w.member, "Dropped by an admin")
	w.must(t, w.owner, "DELETE", "/api/v1/ideas/"+again.ID, "", http.StatusOK)
	if inbox(w.member) != 0 {
		t.Fatal("an admin's drop did not take the idea out of the author's inbox")
	}
}

// Converting turns an idea into a PRD and its coach conversation, once, and
// only for the author.
func TestIdeaConversionMakesOnePrdForItsAuthor(t *testing.T) {
	w := newCoverageWorld(t)
	idea := w.fileIdea(t, w.member, "To convert")
	path := "/api/v1/ideas/" + idea.ID + "/convert"
	other := w.newMember(t, "other")

	w.must(t, w.owner, "POST", path, "", http.StatusForbidden) // an admin may see it, not convert it
	w.must(t, other, "POST", path, "", http.StatusNotFound)
	w.must(t, w.outsider, "POST", path, "", http.StatusNotFound)
	w.must(t, w.member, "POST", "/api/v1/ideas/"+uuid.NewString()+"/convert", "", http.StatusNotFound)
	w.must(t, w.member, "POST", "/api/v1/ideas/not-a-uuid/convert", "", http.StatusBadRequest)
	w.must(t, w.member, "POST", path, `{"locale":`, http.StatusBadRequest)
	if got := decodeAs[ideaView](t, w.must(t, w.member, "GET", "/api/v1/ideas/"+idea.ID, "", http.StatusOK)); got.Status != "inbox" || got.PrdID != nil {
		t.Fatalf("refused conversions left the idea as %+v, want it unconverted", got)
	}

	made := decodeAs[map[string]string](t, w.must(t, w.member, "POST", path, `{"locale":"en-US"}`, http.StatusCreated))
	if made["prdId"] == "" || made["conversationId"] == "" {
		t.Fatalf("conversion answered %v, want a PRD and a conversation", made)
	}
	conv := decodeAs[conversationView](t, w.must(t, w.member, "GET", "/api/v1/conversations/"+made["conversationId"], "", http.StatusOK))
	if conv.PrdID != made["prdId"] || conv.Locale != "en" {
		t.Fatalf("conversation = %+v, want it on the new PRD, coaching in the locale asked for", conv)
	}
	got := decodeAs[ideaView](t, w.must(t, w.member, "GET", "/api/v1/ideas/"+idea.ID, "", http.StatusOK))
	if got.Status != "converted" || got.PrdID == nil || *got.PrdID != made["prdId"] {
		t.Fatalf("converted idea = %+v, want status converted and the new PRD", got)
	}
	w.must(t, w.member, "POST", path, "", http.StatusConflict)

	// No body at all is a valid conversion too, and coaches in the default locale.
	plain := w.fileIdea(t, w.member, "Converted without a body")
	made = decodeAs[map[string]string](t, w.must(t, w.member, "POST", "/api/v1/ideas/"+plain.ID+"/convert", "", http.StatusCreated))
	conv = decodeAs[conversationView](t, w.must(t, w.member, "GET", "/api/v1/conversations/"+made["conversationId"], "", http.StatusOK))
	if conv.Locale != "zh" {
		t.Fatalf("locale of a conversion without a body = %q, want the zh default", conv.Locale)
	}
}
