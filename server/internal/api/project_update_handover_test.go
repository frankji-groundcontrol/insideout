package api

import (
	"net/http"
	"strings"
	"testing"
)

// rv4-TEST-03: who may change a project's owner. The web client's edit always
// sends the owner it was shown (project_page.dart sends data['ownerId'] back),
// so an owner who is not a workspace admin edits with their own id; only a
// change of the owner itself is for an admin. Real PostgreSQL, real router.

// rvHandoverPatch is a PATCH body; owner is the raw ownerId member (with its
// leading comma) or "" to leave the key out.
func rvHandoverPatch(title, status, owner string) string {
	return `{"title":` + js(title) + `,"description":"d","status":"` + status + `"` + owner + `}`
}

// rvHandoverOwner is the ownerId member of a PATCH body naming id.
func rvHandoverOwner(id string) string { return `,"ownerId":"` + id + `"` }

// rvHandoverGiveTo has the world's admin make who the project's owner.
func rvHandoverGiveTo(t *testing.T, w *routeWorld, who routeUser) {
	t.Helper()
	w.must(t, w.owner, "PATCH", w.proj(), rvHandoverPatch("Handed down", "active", rvHandoverOwner(who.ID.String())), http.StatusOK)
}

// rvHandoverOwnerOf reads the project back and returns its owner and title.
func rvHandoverOwnerOf(t *testing.T, w *routeWorld) (owner *string, title string) {
	t.Helper()
	got := decodeAs[projectDetail](t, w.must(t, w.owner, "GET", w.proj(), "", http.StatusOK))
	return got.OwnerID, got.Title
}

// An edit that keeps the owner is the owner's to make, admin or not. These
// pass without the fix and must keep passing with it.
func TestRvHandoverAPlainEditByANonAdminOwnerStaysAllowed(t *testing.T) {
	w := newCoverageWorld(t)
	rvHandoverGiveTo(t, w, w.member)
	own := w.member.ID.String()

	for name, body := range map[string]string{
		"title only":             rvHandoverPatch("Retitled", "active", rvHandoverOwner(own)),
		"status only":            rvHandoverPatch("Handed down", "paused", rvHandoverOwner(own)),
		"description only":       `{"title":"Handed down","description":"a new description","status":"active","ownerId":"` + own + `"}`,
		"all three":              `{"title":"All three","description":"changed","status":"done","ownerId":"` + own + `"}`,
		"own id spelled in caps": rvHandoverPatch("Caps", "active", rvHandoverOwner(strings.ToUpper(own))),
	} {
		t.Run(name, func(t *testing.T) {
			got := decodeAs[projectView](t, w.must(t, w.member, "PATCH", w.proj(), body, http.StatusOK))
			if got.OwnerID == nil || *got.OwnerID != own {
				t.Fatalf("owner after a plain edit = %v, want the editor, unchanged", got.OwnerID)
			}
		})
	}
}

// An admin may hand the project to a member and take it away from one, or
// leave it ownerless, however the empty owner is spelled. These pass without
// the fix and must keep passing with it.
func TestRvHandoverAnAdminCanStillHandTheProjectOverOrClearIt(t *testing.T) {
	w := newCoverageWorld(t)
	other := w.newMember(t, "other")
	rvHandoverGiveTo(t, w, w.member)

	w.must(t, w.owner, "PATCH", w.proj(), rvHandoverPatch("To another", "active", rvHandoverOwner(other.ID.String())), http.StatusOK)
	if owner, _ := rvHandoverOwnerOf(t, w); owner == nil || *owner != other.ID.String() {
		t.Fatalf("owner after the admin's hand-over = %v, want the other member", owner)
	}
	w.must(t, w.owner, "PATCH", w.proj(), rvHandoverPatch("Taken back", "active", rvHandoverOwner(w.owner.ID.String())), http.StatusOK)

	for name, fragment := range map[string]string{
		"an empty string": `,"ownerId":""`,
		"null":            `,"ownerId":null`,
		"left out":        "",
	} {
		t.Run("clear with "+name, func(t *testing.T) {
			rvHandoverGiveTo(t, w, w.member)
			w.must(t, w.owner, "PATCH", w.proj(), rvHandoverPatch("Cleared", "active", fragment), http.StatusOK)
			if owner, title := rvHandoverOwnerOf(t, w); owner != nil || title != "Cleared" {
				t.Fatalf("after the admin cleared the owner (%s): owner %v, title %q, want no owner", name, owner, title)
			}
		})
	}
}

// The 403 says what is needed. A caller who only left ownerId out (which, in
// this API, clears it) learns the way out; a member who is not the owner keeps
// the plain answer.
func TestRvHandoverTheRefusalSaysAnAdminIsNeeded(t *testing.T) {
	w := newCoverageWorld(t)
	other := w.newMember(t, "other")
	rvHandoverGiveTo(t, w, w.member)

	refusal := func(t *testing.T, as routeUser, payload string) string {
		t.Helper()
		return decodeAs[struct{ Error string }](t, w.must(t, as, "PATCH", w.proj(), payload, http.StatusForbidden)).Error
	}
	for name, payload := range map[string]string{
		"to another member": rvHandoverPatch("Sneaked", "active", rvHandoverOwner(other.ID.String())),
		"to nobody":         rvHandoverPatch("Sneaked", "active", ""),
	} {
		t.Run(name, func(t *testing.T) {
			msg := refusal(t, w.member, payload)
			for _, want := range []string{"workspace admin", "someone else", "ownerId"} {
				if !strings.Contains(msg, want) {
					t.Fatalf("refusal %q does not mention %q", msg, want)
				}
			}
		})
	}

	// A member who owns nothing here is told the older, plainer thing.
	msg := refusal(t, other, rvHandoverPatch("Sneaked", "active", rvHandoverOwner(other.ID.String())))
	if strings.Contains(msg, "someone else") || !strings.Contains(msg, "project owner or a workspace admin") {
		t.Fatalf("refusal for a member who is not the owner = %q, want the plain owner-or-admin answer", msg)
	}
	if _, title := rvHandoverOwnerOf(t, w); title != "Handed down" {
		t.Fatalf("title = %q after the refused edits, want it untouched", title)
	}
}
