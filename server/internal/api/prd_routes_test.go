package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// convertedPrd has `as` file an idea and convert it, which is how a PRD comes
// to exist: the PRD is theirs and so is its coach conversation.
func (w *routeWorld) convertedPrd(t *testing.T, as routeUser, title string) (prdID, conversationID string) {
	t.Helper()
	idea := w.fileIdea(t, as, title)
	made := decodeAs[map[string]string](t, w.must(t, as, "POST", "/api/v1/ideas/"+idea.ID+"/convert", "", http.StatusCreated))
	return made["prdId"], made["conversationId"]
}

func TestPrdIsReadByMembersAndSavedByItsAuthorOrAnAdmin(t *testing.T) {
	w := newCoverageWorld(t)
	other := w.newMember(t, "other")
	id, _ := w.convertedPrd(t, w.member, "Parser")
	path := "/api/v1/prds/" + id
	read := func(as routeUser) prdView {
		t.Helper()
		return decodeAs[prdView](t, w.must(t, as, "GET", path, "", http.StatusOK))
	}
	save := func(as routeUser, payload string, want int) string {
		t.Helper()
		return w.must(t, as, "PATCH", path, payload, want)
	}

	for _, as := range []routeUser{w.member, w.owner, other} {
		got := read(as)
		if got.Title != "Parser" || got.Status != "draft" || got.AuthorID != w.member.ID.String() || len(got.Sections) != len(store.PrdSectionKeys) {
			t.Fatalf("PRD = %+v, want a draft by the member with all %d sections", got, len(store.PrdSectionKeys))
		}
	}
	w.must(t, w.outsider, "GET", path, "", http.StatusNotFound)
	w.must(t, w.member, "GET", "/api/v1/prds/"+uuid.NewString(), "", http.StatusNotFound)
	w.must(t, w.member, "GET", "/api/v1/prds/not-a-uuid", "", http.StatusBadRequest)

	save(other, `{"sections":{"background":"hijacked"}}`, http.StatusForbidden)
	save(w.outsider, `{"sections":{"background":"hijacked"}}`, http.StatusNotFound)
	for name, payload := range map[string]string{
		"unknown section": `{"sections":{"appendix":"x"}}`,
		"blank title":     `{"title":"   "}`,
		"title too long":  `{"title":"` + strings.Repeat("x", 201) + `"}`,
		"unknown field":   `{"status":"approved"}`,
		"not json":        `background=x`,
	} {
		t.Run(name, func(t *testing.T) { save(w.member, payload, http.StatusBadRequest) })
	}
	if got := read(w.member); got.Sections["background"] != "" || got.Title != "Parser" || got.Status != "draft" {
		t.Fatalf("refused saves left the PRD as %+v, want it untouched", got)
	}

	// A save names the sections it changes and leaves the others, and the title,
	// alone; the admin may save too.
	got := decodeAs[prdView](t, save(w.member, `{"sections":{"background":"Why now"}}`, http.StatusOK))
	if got.Sections["background"] != "Why now" || got.Title != "Parser" {
		t.Fatalf("saved PRD = %+v, want the background written and the title kept", got)
	}
	got = decodeAs[prdView](t, save(w.owner, `{"title":"  Parser v2  ","sections":{"goals":"Ship it"}}`, http.StatusOK))
	if got.Title != "Parser v2" || got.Sections["goals"] != "Ship it" || got.Sections["background"] != "Why now" {
		t.Fatalf("admin's save = %+v, want the trimmed title, goals added, background kept", got)
	}
}

func TestPrdRevisionsAreSnapshotsTakenByTheAuthor(t *testing.T) {
	w := newCoverageWorld(t)
	id, _ := w.convertedPrd(t, w.member, "Snapshots")
	path := "/api/v1/prds/" + id + "/revisions"
	w.must(t, w.member, "PATCH", "/api/v1/prds/"+id, `{"sections":{"background":"first words"}}`, http.StatusOK)

	if body := w.must(t, w.owner, "GET", path, "", http.StatusOK); strings.TrimSpace(body) != "[]" {
		t.Fatalf("revisions of a PRD never snapshotted = %s, want an empty list", body)
	}
	w.must(t, w.owner, "POST", path, `{"note":"by an admin"}`, http.StatusForbidden) // only the author snapshots
	w.must(t, w.outsider, "POST", path, "", http.StatusNotFound)
	w.must(t, w.outsider, "GET", path, "", http.StatusNotFound)
	w.must(t, w.member, "POST", path, `{"note":`, http.StatusBadRequest)
	w.must(t, w.member, "POST", "/api/v1/prds/"+uuid.NewString()+"/revisions", "", http.StatusNotFound)
	w.must(t, w.member, "GET", "/api/v1/prds/not-a-uuid/revisions", "", http.StatusBadRequest)

	// A PRD is born holding revision 1 as its working version, so a snapshot
	// takes the number after the one the PRD is on.
	on := decodeAs[prdView](t, w.must(t, w.member, "GET", "/api/v1/prds/"+id, "", http.StatusOK)).CurrentRevision
	first := decodeAs[revisionView](t, w.must(t, w.member, "POST", path, `{"note":"first cut"}`, http.StatusCreated))
	if first.Revision != on+1 || first.Note == nil || *first.Note != "first cut" || first.CreatedBy != w.member.ID.String() || first.Sections["background"] != "first words" {
		t.Fatalf("first revision = %+v, want the next number, the note, the member, and the sections as saved", first)
	}
	w.must(t, w.member, "PATCH", "/api/v1/prds/"+id, `{"sections":{"background":"second words"}}`, http.StatusOK)
	second := decodeAs[revisionView](t, w.must(t, w.member, "POST", path, "", http.StatusCreated)) // body-less is fine
	if second.Revision != on+2 || second.Note != nil || second.Sections["background"] != "second words" {
		t.Fatalf("second revision = %+v, want the number after, no note, the later sections", second)
	}

	// Newest first, and each keeps the words it was taken with.
	for _, as := range []routeUser{w.member, w.owner} {
		list := decodeAs[[]revisionView](t, w.must(t, as, "GET", path, "", http.StatusOK))
		if len(list) != 2 || list[0].Revision != on+2 || list[1].Revision != on+1 || list[1].Sections["background"] != "first words" {
			t.Fatalf("revisions = %+v, want the newer then the older, the older still holding its own words", list)
		}
	}
	if got := decodeAs[prdView](t, w.must(t, w.member, "GET", "/api/v1/prds/"+id, "", http.StatusOK)); got.CurrentRevision != on+2 {
		t.Fatalf("currentRevision = %d, want %d", got.CurrentRevision, on+2)
	}
}

// The review lifecycle splits the roles: the author submits and reopens, only
// an admin decides, and an approved PRD is final.
func TestPrdStatusLifecycleSplitsAuthorAndAdmin(t *testing.T) {
	w := newCoverageWorld(t)
	other := w.newMember(t, "other")
	id, _ := w.convertedPrd(t, w.member, "Lifecycle")
	path := "/api/v1/prds/" + id + "/status"
	move := func(as routeUser, status string, want int) string {
		t.Helper()
		return w.must(t, as, "POST", path, `{"status":"`+status+`"}`, want)
	}
	status := func() string {
		t.Helper()
		return decodeAs[prdView](t, w.must(t, w.member, "GET", "/api/v1/prds/"+id, "", http.StatusOK)).Status
	}

	move(w.member, "published", http.StatusBadRequest) // not a status at all
	w.must(t, w.member, "POST", path, `{"status":`, http.StatusBadRequest)
	w.must(t, w.member, "POST", path, `{"status":"reviewing","reason":"x"}`, http.StatusBadRequest)
	move(w.member, "approved", http.StatusBadRequest) // a draft cannot skip review
	move(w.outsider, "reviewing", http.StatusNotFound)
	w.must(t, w.member, "POST", "/api/v1/prds/"+uuid.NewString()+"/status", `{"status":"reviewing"}`, http.StatusNotFound)
	w.must(t, w.member, "POST", "/api/v1/prds/not-a-uuid/status", `{"status":"reviewing"}`, http.StatusBadRequest)
	move(w.owner, "reviewing", http.StatusForbidden) // submitting is the author's, even for an admin
	move(other, "reviewing", http.StatusForbidden)
	if got := status(); got != "draft" {
		t.Fatalf("refused moves left the PRD %q, want draft", got)
	}

	if got := decodeAs[prdView](t, move(w.member, "reviewing", http.StatusOK)); got.Status != "reviewing" {
		t.Fatalf("after submitting = %q, want reviewing", got.Status)
	}
	move(w.member, "approved", http.StatusForbidden) // never the author alone
	move(other, "rejected", http.StatusForbidden)
	move(w.member, "draft", http.StatusBadRequest) // reviewing does not go back to draft
	if got := status(); got != "reviewing" {
		t.Fatalf("PRD = %q after refused decisions, want reviewing", got)
	}

	move(w.owner, "rejected", http.StatusOK)
	move(w.owner, "draft", http.StatusForbidden) // reopening is the author's
	move(w.member, "draft", http.StatusOK)
	move(w.member, "reviewing", http.StatusOK)
	move(w.owner, "approved", http.StatusOK)
	for _, next := range []string{"draft", "reviewing", "rejected", "approved"} {
		move(w.owner, next, http.StatusBadRequest) // approved is final
	}
	if got := status(); got != "approved" {
		t.Fatalf("PRD = %q at the end, want approved", got)
	}
}

// A coach conversation belongs to the one who started it; conversion starts the
// converter's, and an admin starts their own on someone else's PRD.
func TestPrdConversationIsStartedAndFoundPerPerson(t *testing.T) {
	w := newCoverageWorld(t)
	other := w.newMember(t, "other")
	id, convID := w.convertedPrd(t, w.member, "Coaching")
	path := "/api/v1/prds/" + id + "/conversation"

	if got := decodeAs[conversationView](t, w.must(t, w.member, "GET", path, "", http.StatusOK)); got.ID != convID || got.PrdID != id {
		t.Fatalf("converter's conversation = %+v, want the one made by the conversion", got)
	}
	w.must(t, w.owner, "GET", path, "", http.StatusNotFound) // an admin has none of their own yet
	w.must(t, w.outsider, "GET", path, "", http.StatusNotFound)
	w.must(t, w.member, "GET", "/api/v1/prds/"+uuid.NewString()+"/conversation", "", http.StatusNotFound)
	w.must(t, w.member, "GET", "/api/v1/prds/not-a-uuid/conversation", "", http.StatusBadRequest)

	w.must(t, other, "POST", path, "", http.StatusForbidden) // neither the author nor an admin
	w.must(t, w.outsider, "POST", path, "", http.StatusNotFound)
	w.must(t, w.owner, "POST", path, `{"locale":`, http.StatusBadRequest)
	started := decodeAs[conversationView](t, w.must(t, w.owner, "POST", path, `{"locale":"en-US"}`, http.StatusCreated))
	if started.ID == convID || started.PrdID != id || started.Locale != "en" {
		t.Fatalf("admin's new conversation = %+v, want their own on the PRD, in English", started)
	}
	again := decodeAs[conversationView](t, w.must(t, w.owner, "POST", path, "", http.StatusOK))
	if again.ID != started.ID {
		t.Fatalf("a second start made conversation %s, want the first (%s) again", again.ID, started.ID)
	}
	if got := decodeAs[conversationView](t, w.must(t, w.owner, "GET", path, "", http.StatusOK)); got.ID != started.ID {
		t.Fatalf("admin finds %s, want their own conversation %s", got.ID, started.ID)
	}
	if got := decodeAs[conversationView](t, w.must(t, w.member, "GET", path, "", http.StatusOK)); got.ID != convID {
		t.Fatalf("converter finds %s, want still the one from the conversion (%s)", got.ID, convID)
	}
	w.must(t, other, "GET", "/api/v1/conversations/"+started.ID, "", http.StatusNotFound)
}
