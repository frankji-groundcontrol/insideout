package api

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// rv5-STRS-11: the handlers that answered store errors through their own
// ladders now go through writeStoreError. These pin what a client sees on each
// of those paths (status, code and message), on the real router and real
// PostgreSQL, so the conversion keeps every answer.

type errCase struct {
	name               string
	as                 routeUser
	method, path, body string
	status             int
	code, msg          string
}

func (w *routeWorld) expectErrors(t *testing.T, cases []errCase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := decodeAs[struct{ Error, Code string }](t, w.must(t, c.as, c.method, c.path, c.body, c.status))
			if got.Code != c.code || got.Error != c.msg {
				t.Fatalf("%s %s: code %q, message %q; want %q, %q", c.method, c.path, got.Code, got.Error, c.code, c.msg)
			}
		})
	}
}

func TestStoreErrorAnswersOfTheMemberAndWorkspaceRoutes(t *testing.T) {
	w := newCoverageWorld(t)
	other := w.newMember(t, "other")
	member := func(u routeUser) string { return w.ws() + "/members/" + u.ID.String() }
	noWs := "/api/v1/workspaces/" + uuid.NewString()
	w.expectErrors(t, []errCase{
		{"roster to an outsider", w.outsider, "GET", w.ws() + "/members", "", http.StatusNotFound, "not_found", "workspace not found"},
		{"role by a plain member", other, "PATCH", member(w.member), `{"role":"admin"}`, http.StatusForbidden, "forbidden", "only a workspace admin can change member roles"},
		{"role of a non-member", w.owner, "PATCH", member(w.outsider), `{"role":"member"}`, http.StatusNotFound, "not_found", "member not found"},
		{"remove by a plain member", other, "DELETE", member(w.member), "", http.StatusForbidden, "forbidden", "only a workspace admin can remove other members"},
		{"remove a non-member", w.owner, "DELETE", member(w.outsider), "", http.StatusNotFound, "not_found", "member not found"},
		{"join an unknown code", w.outsider, "POST", "/api/v1/workspaces/join", `{"code":"no-such-code"}`, http.StatusNotFound, "not_found", "no active workspace with this code"},
		{"workspace to an outsider", w.outsider, "GET", w.ws(), "", http.StatusNotFound, "not_found", "workspace not found"},
		{"edit by a plain member", w.member, "PATCH", w.ws(), `{"title":"x"}`, http.StatusForbidden, "forbidden", "only the workspace admin or creator can do this"},
		{"edit an unknown workspace", w.member, "PATCH", noWs, `{"title":"x"}`, http.StatusNotFound, "not_found", "workspace not found"},
		{"delete by a plain member", w.member, "DELETE", w.ws(), "", http.StatusForbidden, "forbidden", "only the workspace creator can delete it"},
	})
	ws := decodeAs[workspaceView](t, w.must(t, w.owner, "GET", w.ws(), "", http.StatusOK))
	w.expectErrors(t, []errCase{
		{"join twice", w.member, "POST", "/api/v1/workspaces/join", `{"code":` + js(ws.Code) + `}`, http.StatusConflict, "conflict", "already a member of this workspace"},
	})
}

// rv5-STRS-11: the last admin's refusal carries the store's own code, as it
// does on every other route that meets ErrLastAdmin.
func TestLastAdminRefusalCarriesItsCode(t *testing.T) {
	w := newCoverageWorld(t)
	owner := w.ws() + "/members/" + w.owner.ID.String()
	const why = "a workspace needs at least one admin"
	w.expectErrors(t, []errCase{
		{"step down", w.owner, "PATCH", owner, `{"role":"member"}`, http.StatusConflict, "last_admin", why},
		{"leave", w.owner, "DELETE", owner, "", http.StatusConflict, "last_admin", why},
	})
}

func TestStoreErrorAnswersOfTheConversationRoutes(t *testing.T) {
	w := newCoverageWorld(t)
	other := w.newMember(t, "other")
	prd, conv := w.convertedPrd(t, w.member, "Pinned")
	noConv := "/api/v1/conversations/" + uuid.NewString()
	w.expectErrors(t, []errCase{
		{"read someone else's", other, "GET", "/api/v1/conversations/" + conv, "", http.StatusNotFound, "not_found", "conversation not found"},
		{"messages of an unknown one", w.member, "GET", noConv + "/messages", "", http.StatusNotFound, "not_found", "conversation not found"},
		{"send to an unknown one", w.member, "POST", noConv + "/messages", `{"content":"hi"}`, http.StatusNotFound, "not_found", "conversation not found"},
		{"start on an unknown PRD", w.member, "POST", "/api/v1/prds/" + uuid.NewString() + "/conversation", "", http.StatusNotFound, "not_found", "PRD not found"},
		{"start on someone else's PRD", other, "POST", "/api/v1/prds/" + prd + "/conversation", "", http.StatusForbidden, "forbidden", "only the author or a workspace admin can coach this PRD"},
		{"decline a stage not waiting", w.member, "POST", "/api/v1/conversations/" + conv + "/stage/decline", `{"stage":"draft"}`, http.StatusConflict, "conflict", "that stage is not waiting for confirmation"},
	})
}

func TestStoreErrorAnswersOfTheProjectSideRoutes(t *testing.T) {
	w := newCoverageWorld(t)
	noProj := "/api/v1/projects/" + uuid.NewString()
	repo := `{"repoUrl":"https://github.com/example/pinned"}`
	w.expectErrors(t, []errCase{
		{"guide of an unknown project", w.member, "GET", noProj + "/guide", "", http.StatusNotFound, "not_found", "project not found"},
		{"guide to an outsider", w.outsider, "GET", w.proj() + "/guide", "", http.StatusNotFound, "not_found", "project not found"},
		{"presence of an unknown project", w.member, "GET", noProj + "/presence", "", http.StatusNotFound, "not_found", "project not found"},
		{"presence to an outsider", w.outsider, "GET", w.proj() + "/presence", "", http.StatusNotFound, "not_found", "project not found"},
		{"link to an unknown project", w.member, "PUT", noProj + "/repo", repo, http.StatusNotFound, "not_found", "project not found"},
		{"link by a plain member", w.member, "PUT", w.proj() + "/repo", repo, http.StatusForbidden, "forbidden", "only the project owner or a workspace admin can link a repo"},
		{"sync an unknown project", w.member, "POST", noProj + "/sync-github", "", http.StatusNotFound, "not_found", "project not found"},
		{"sync by a plain member", w.member, "POST", w.proj() + "/sync-github", "", http.StatusForbidden, "forbidden", "only the project owner or a workspace admin can sync"},
	})
	node := decodeAs[struct{ ID string }](t, w.must(t, w.owner, "POST", w.proj()+"/roadmap", `{"title":"Pinned"}`, http.StatusCreated))
	nodePath := "/api/v1/roadmap/" + node.ID
	w.expectErrors(t, []errCase{
		{"hand back by a non-owner", w.member, "POST", nodePath + "/handback", `{}`, http.StatusForbidden, "forbidden", "only the node's current owner can hand it back"},
		{"delegate by a plain member", w.member, "POST", nodePath + "/delegate", `{"toUserId":` + js(w.member.ID.String()) + `}`, http.StatusForbidden, "forbidden",
			"only the project owner, a workspace admin, or the owner of the branch above can change this node's owner"},
		{"hand back an unknown node", w.member, "POST", "/api/v1/roadmap/" + uuid.NewString() + "/handback", `{}`, http.StatusNotFound, "not_found", "node not found"},
	})
}

func TestStoreErrorAnswerOfInvitationRevoke(t *testing.T) {
	w, admin := newAdminWorld(t)
	w.expectErrors(t, []errCase{
		{"revoke an unknown invitation", admin, "POST", "/api/v1/admin/invitations/" + uuid.NewString() + "/revoke", "", http.StatusNotFound, "not_found", "invitation not found or already used"},
	})
}
