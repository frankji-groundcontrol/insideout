package store

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
)

// newTestStore connects to DATABASE_URL and skips the test if it's unset
// — these are real integration tests against a real Postgres (RLS
// policies can't be verified against a mock), consistent with this
// project's no-mocks testing policy. Run with:
//
//	DATABASE_URL=... go test ./internal/store/ -run TestAuthz -v
func newTestStore(t *testing.T) *Store {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	st, err := Open(context.Background(), url)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(st.Close)
	return st
}

// mkUser creates a throwaway user for one test, bypassing withUserContext
// (CreateUser is the pre-auth path — see users.go) with a unique email so
// parallel/rerun test runs never collide.
func mkUser(t *testing.T, st *Store) *User {
	t.Helper()
	u, err := st.CreateUser(context.Background(), "authz-"+uuid.NewString()+"@test.local", "x", "authz-tester")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return u
}

func TestAuthz_WorkspaceMembershipLifecycle(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	admin := mkUser(t, st)
	member := mkUser(t, st)
	stranger := mkUser(t, st)

	ws, err := st.CreateWorkspace(ctx, admin.ID, "Authz Test WS", "desc")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}

	// Deny: a stranger who never joined cannot read the workspace, even
	// with its real ID — RLS hides the row entirely (existence not
	// disclosed), matching the pre-RLS app-layer contract.
	if _, err := st.GetWorkspaceForMember(ctx, ws.ID, stranger.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("stranger GetWorkspaceForMember: want ErrNotFound, got %v", err)
	}

	// Deny: a stranger cannot join with a wrong/guessed code.
	if _, err := st.JoinWorkspace(ctx, stranger.ID, "000000"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("join with wrong code: want ErrNotFound, got %v", err)
	}

	// Allow: joining with the real code works — this is the one path that
	// legitimately reads a workspace row before membership exists (see
	// the workspaces_select RLS policy's app.join_code carve-out).
	if _, err := st.JoinWorkspace(ctx, member.ID, ws.Code); err != nil {
		t.Fatalf("join with real code: %v", err)
	}

	// Deny: joining twice is a conflict, not a silent success.
	if _, err := st.JoinWorkspace(ctx, member.ID, ws.Code); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate join: want ErrConflict, got %v", err)
	}

	// Allow: the new member can now read the workspace.
	if _, err := st.GetWorkspaceForMember(ctx, ws.ID, member.ID); err != nil {
		t.Fatalf("member GetWorkspaceForMember: %v", err)
	}

	// Deny: a plain member (not admin, not creator) cannot update the
	// workspace.
	if _, err := st.UpdateWorkspace(ctx, member.ID, ws.ID, "renamed", "desc", nil); !errors.Is(err, ErrForbidden) {
		t.Fatalf("member UpdateWorkspace: want ErrForbidden, got %v", err)
	}

	// Deny: a plain member cannot change another member's role.
	if err := st.UpdateMemberRole(ctx, member.ID, ws.ID, admin.ID, "member"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("member UpdateMemberRole: want ErrForbidden, got %v", err)
	}

	// Allow: the admin (creator) can promote the member.
	if err := st.UpdateMemberRole(ctx, admin.ID, ws.ID, member.ID, "admin"); err != nil {
		t.Fatalf("admin UpdateMemberRole: %v", err)
	}

	// Deny: a stranger still can't see the member list.
	if _, err := st.ListMembers(ctx, stranger.ID, ws.ID); err != nil {
		t.Fatalf("stranger ListMembers unexpectedly errored: %v", err)
	} else {
		list, _ := st.ListMembers(ctx, stranger.ID, ws.ID)
		if len(list) != 0 {
			t.Fatalf("stranger ListMembers: want empty (RLS-filtered), got %d rows", len(list))
		}
	}

	// Deny: only the creator can delete, not a (now-)admin member.
	if err := st.DeleteWorkspace(ctx, member.ID, ws.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("non-creator DeleteWorkspace: want ErrForbidden, got %v", err)
	}
}

func TestAuthz_ProjectsAndIdeasAndPrds(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	admin := mkUser(t, st)
	author := mkUser(t, st)
	stranger := mkUser(t, st)

	ws, err := st.CreateWorkspace(ctx, admin.ID, "Authz Test WS 2", "desc")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if _, err := st.JoinWorkspace(ctx, author.ID, ws.Code); err != nil {
		t.Fatalf("join: %v", err)
	}

	// --- projects ---
	proj, err := st.CreateProject(ctx, author.ID, ws.ID, "Proj", "desc")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if _, err := st.GetProjectForMember(ctx, proj.ID, stranger.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("stranger GetProjectForMember: want ErrNotFound, got %v", err)
	}
	// Deny: the project's author is not its owner by default here and is
	// not workspace admin either — a non-owner, non-admin member cannot
	// update it. (author == created_by, but owner_id defaults to creator
	// too per CreateProject, so use a third member to get a genuine
	// non-owner/non-admin actor.)
	other := mkUser(t, st)
	if _, err := st.JoinWorkspace(ctx, other.ID, ws.Code); err != nil {
		t.Fatalf("join: %v", err)
	}
	if _, err := st.UpdateProject(ctx, other.ID, proj.ID, ProjectUpdateFields{Title: "x", Description: "y", Status: "active"}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("non-owner/non-admin UpdateProject: want ErrForbidden, got %v", err)
	}
	// Allow: workspace admin can update any project.
	if _, err := st.UpdateProject(ctx, admin.ID, proj.ID, ProjectUpdateFields{Title: "renamed", Description: "y", Status: "active"}); err != nil {
		t.Fatalf("admin UpdateProject: %v", err)
	}

	// --- ideas ---
	idea, err := st.CreateIdea(ctx, author.ID, ws.ID, "Idea", "content")
	if err != nil {
		t.Fatalf("create idea: %v", err)
	}
	// Deny: only the author can edit an idea, not another member.
	if _, err := st.UpdateIdea(ctx, other.ID, idea.ID, "hijacked", "x"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("non-author UpdateIdea: want ErrForbidden, got %v", err)
	}
	// Deny: a stranger can't even see the idea exists.
	if _, err := st.GetIdeaForMember(ctx, idea.ID, stranger.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("stranger GetIdeaForMember: want ErrNotFound, got %v", err)
	}

	// --- convert idea -> PRD, then the review lifecycle's author-vs-admin split ---
	prd, conv, err := st.ConvertIdea(ctx, author.ID, idea.ID)
	if err != nil {
		t.Fatalf("convert idea: %v", err)
	}
	if conv.UserID != author.ID {
		t.Fatalf("conversation owner mismatch: got %s want %s", conv.UserID, author.ID)
	}

	// Deny: a stranger can't read the PRD.
	if _, err := st.GetPrdForMember(ctx, prd.ID, stranger.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("stranger GetPrdForMember: want ErrNotFound, got %v", err)
	}

	// Allow: author submits draft -> reviewing.
	if _, err := st.UpdatePrdStatus(ctx, author.ID, prd.ID, "reviewing"); err != nil {
		t.Fatalf("author submit for review: %v", err)
	}

	// Deny: the author cannot approve their own PRD (no self-review).
	if _, err := st.UpdatePrdStatus(ctx, author.ID, prd.ID, "approved"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("author self-approve: want ErrForbidden, got %v", err)
	}

	// Deny: a non-admin member cannot approve either.
	if _, err := st.UpdatePrdStatus(ctx, other.ID, prd.ID, "approved"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("non-admin approve: want ErrForbidden, got %v", err)
	}

	// Allow: the workspace admin approves.
	if _, err := st.UpdatePrdStatus(ctx, admin.ID, prd.ID, "approved"); err != nil {
		t.Fatalf("admin approve: %v", err)
	}

	// Deny: approved is terminal — even the admin can't skip back to draft.
	if _, err := st.UpdatePrdStatus(ctx, admin.ID, prd.ID, "draft"); !errors.Is(err, ErrValidation) {
		t.Fatalf("approved->draft: want ErrValidation, got %v", err)
	}
}

func TestAuthz_ConversationsAreOwnerOnly(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	admin := mkUser(t, st)
	author := mkUser(t, st)
	other := mkUser(t, st)

	ws, err := st.CreateWorkspace(ctx, admin.ID, "Authz Test WS 3", "desc")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if _, err := st.JoinWorkspace(ctx, author.ID, ws.Code); err != nil {
		t.Fatalf("join: %v", err)
	}
	if _, err := st.JoinWorkspace(ctx, other.ID, ws.Code); err != nil {
		t.Fatalf("join: %v", err)
	}

	idea, err := st.CreateIdea(ctx, author.ID, ws.ID, "Idea", "content")
	if err != nil {
		t.Fatalf("create idea: %v", err)
	}
	_, conv, err := st.ConvertIdea(ctx, author.ID, idea.ID)
	if err != nil {
		t.Fatalf("convert idea: %v", err)
	}

	// Deny: a fellow workspace member (not the conversation owner) cannot
	// read it — conversations are strictly owner-only, unlike everything
	// else in the workspace.
	if _, err := st.GetConversationForOwner(ctx, conv.ID, other.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("non-owner GetConversationForOwner: want ErrNotFound, got %v", err)
	}
	if _, err := st.GetConversationForOwner(ctx, conv.ID, author.ID); err != nil {
		t.Fatalf("owner GetConversationForOwner: %v", err)
	}

	msg, err := st.InsertAgentMessage(ctx, author.ID, conv.ID, "user", "hello", nil, nil, nil)
	if err != nil {
		t.Fatalf("insert message: %v", err)
	}
	// Deny: another member cannot list or finalize messages in someone
	// else's conversation.
	otherMsgs, err := st.ListAgentMessages(ctx, other.ID, conv.ID)
	if err != nil {
		t.Fatalf("other ListAgentMessages unexpectedly errored: %v", err)
	}
	if len(otherMsgs) != 0 {
		t.Fatalf("other ListAgentMessages: want empty (RLS-filtered), got %d", len(otherMsgs))
	}
	ownMsgs, err := st.ListAgentMessages(ctx, author.ID, conv.ID)
	if err != nil {
		t.Fatalf("owner ListAgentMessages: %v", err)
	}
	if len(ownMsgs) != 1 {
		t.Fatalf("owner ListAgentMessages: want 1, got %d", len(ownMsgs))
	}
	if err := st.UpdateAgentMessageContent(ctx, other.ID, msg.ID, "hijacked", nil); err != nil {
		t.Fatalf("other UpdateAgentMessageContent unexpectedly errored: %v", err)
	}
	// Verify the hijack attempt silently affected zero rows (RLS-filtered
	// UPDATE matches nothing), not that it errored — Postgres UPDATE ...
	// WHERE matching zero rows is not itself an error.
	ownMsgs, err = st.ListAgentMessages(ctx, author.ID, conv.ID)
	if err != nil {
		t.Fatalf("re-list: %v", err)
	}
	if ownMsgs[0].Content == "hijacked" {
		t.Fatalf("non-owner UpdateAgentMessageContent: RLS should have blocked this write, but content was changed")
	}
}
