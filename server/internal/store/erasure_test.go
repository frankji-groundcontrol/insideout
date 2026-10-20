package store

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/testlock"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// TestErase_IdleConversations is the retention half of rv4-DATA-06: a coach
// conversation lives 365 days after its newest message, or after its own
// creation when it has none. A newer message keeps the conversation.
func TestErase_IdleConversations(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	user := mkUser(t, st)
	ws, err := st.CreateWorkspace(ctx, user.ID, "Idle "+uuid.NewString()[:8], "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}

	old := mustConversation(t, st, user.ID, ws.ID, "old", "old body")
	ageConversation(t, st, user.ID, old, 400, true)
	fresh := mustConversation(t, st, user.ID, ws.ID, "fresh", "fresh body")
	empty := mustConversation(t, st, user.ID, ws.ID, "empty", "")
	ageConversation(t, st, user.ID, empty, 400, false)
	mixed := mustConversation(t, st, user.ID, ws.ID, "mixed", "first turn")
	ageConversation(t, st, user.ID, mixed, 400, true)
	if _, err := st.InsertAgentMessage(ctx, user.ID, mixed, "user", "still talking", nil, nil, nil); err != nil {
		t.Fatalf("fresh message on an old conversation: %v", err)
	}
	linked, err := st.CreateAIRun(ctx, ws.ID, &old, user.ID, "linked prompt")
	if err != nil {
		t.Fatalf("create linked run: %v", err)
	}

	if _, err := st.ReapStaleAIRuns(ctx, 10*time.Minute); err != nil {
		t.Fatalf("reap: %v", err)
	}

	if conversationVisible(t, st, user.ID, old) {
		t.Error("a conversation whose newest message is older than 365 days was not deleted")
	}
	if !conversationVisible(t, st, user.ID, fresh) {
		t.Error("a newer conversation was deleted")
	}
	if conversationVisible(t, st, user.ID, empty) {
		t.Error("a conversation with no messages, older than 365 days by its creation time, was not deleted")
	}
	if !conversationVisible(t, st, user.ID, mixed) {
		t.Error("a conversation whose newest message is inside the window was deleted")
	}
	if messageVisible(t, st, user.ID, old) {
		t.Error("messages of a purged conversation were not deleted")
	}
	var convID *uuid.UUID
	if err := st.Pool.QueryRow(ctx, `SELECT conversation_id FROM insideout.ai_runs WHERE id = $1`, linked.ID).Scan(&convID); err != nil {
		t.Fatalf("linked run: %v", err)
	}
	if convID != nil {
		t.Error("purging a conversation did not set ai_runs.conversation_id null")
	}
}

// TestErase_Account is the tombstone half: private rows go, the account row
// stays under placeholders, and another member's PRD and commit do not change.
func TestErase_Account(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	ownerURL := os.Getenv("DATABASE_OWNER_URL")
	if ownerURL == "" {
		t.Fatal("DATABASE_OWNER_URL is required so the audit can see through row-level security")
	}
	testlock.RequireLocalURL(t, ownerURL)
	audit, err := Open(ctx, ownerURL)
	if err != nil {
		t.Fatalf("open owner: %v", err)
	}
	t.Cleanup(audit.Close)

	erased := mkUser(t, st)
	other := mkUser(t, st)
	address := erased.Email
	shared, err := st.CreateWorkspace(ctx, erased.ID, "Shared "+uuid.NewString()[:8], "")
	if err != nil {
		t.Fatalf("shared workspace: %v", err)
	}
	if _, err := st.JoinWorkspace(ctx, other.ID, shared.Code); err != nil {
		t.Fatalf("join: %v", err)
	}
	keptIdea, err := st.CreateIdea(ctx, other.ID, shared.ID, "kept prd", "other member's text")
	if err != nil {
		t.Fatalf("kept idea: %v", err)
	}
	keptPRD, _, err := st.ConvertIdea(ctx, other.ID, keptIdea.ID)
	if err != nil {
		t.Fatalf("kept prd: %v", err)
	}
	keptCommit, err := st.CreatePrdCommit(ctx, other.ID, keptPRD.ID, "kept-commit", "readers", "unchanged", nil, "note")
	if err != nil {
		t.Fatalf("kept commit: %v", err)
	}

	_, err = st.CreateIdea(ctx, erased.ID, shared.ID, "inbox", address)
	if err != nil {
		t.Fatalf("inbox idea: %v", err)
	}
	dropped, err := st.CreateIdea(ctx, erased.ID, shared.ID, "dropped", address)
	if err != nil {
		t.Fatalf("dropped idea: %v", err)
	}
	if err := st.DropIdea(ctx, erased.ID, dropped.ID); err != nil {
		t.Fatalf("drop idea: %v", err)
	}
	ownPRDIdea, err := st.CreateIdea(ctx, erased.ID, shared.ID, "own prd", address)
	if err != nil {
		t.Fatalf("own prd idea: %v", err)
	}
	ownPRD, ownConv, err := st.ConvertIdea(ctx, erased.ID, ownPRDIdea.ID)
	if err != nil {
		t.Fatalf("own prd: %v", err)
	}
	if _, err := st.InsertAgentMessage(ctx, erased.ID, ownConv.ID, "user", address, nil, nil, nil); err != nil {
		t.Fatalf("message: %v", err)
	}
	if _, err := st.CreateAIRun(ctx, shared.ID, &ownConv.ID, erased.ID, address); err != nil {
		t.Fatalf("run: %v", err)
	}
	if _, err := st.CreateSession(ctx, erased.ID, "erase-session-"+uuid.NewString(), time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("session: %v", err)
	}
	if err := st.CreateAuthToken(ctx, erased.ID, AuthTokenMagicLink, "erase-token-"+uuid.NewString(), time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("token: %v", err)
	}
	sole, err := st.CreateWorkspace(ctx, erased.ID, "Sole "+uuid.NewString()[:8], "")
	if err != nil {
		t.Fatalf("sole workspace: %v", err)
	}
	joined, err := st.CreateWorkspace(ctx, other.ID, "Joined "+uuid.NewString()[:8], "")
	if err != nil {
		t.Fatalf("joined workspace: %v", err)
	}
	if _, err := st.JoinWorkspace(ctx, erased.ID, joined.Code); err != nil {
		t.Fatalf("join other: %v", err)
	}
	hash, err := auth.HashPassword("correct-horse")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if err := st.UpdatePasswordHash(ctx, erased.ID, hash); err != nil {
		t.Fatalf("set password: %v", err)
	}
	avatar := "https://cdn.example.invalid/a.png"
	if _, err := st.UpdateProfile(ctx, erased.ID, "erased-name-"+uuid.NewString()[:8], address, []string{"kw"}, &avatar); err != nil {
		t.Fatalf("profile: %v", err)
	}

	if err := st.EraseUser(ctx, erased.ID); err != nil {
		t.Fatalf("erase: %v", err)
	}

	if hits := textsNaming(t, audit, address); len(hits) != 0 {
		t.Errorf("after erasure these columns still name the old address: %s", strings.Join(hits, ", "))
	}
	if _, err := st.GetUserByEmail(ctx, address); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetUserByEmail(old address) = %v, want ErrNotFound", err)
	}
	tomb, err := st.GetUserByID(ctx, erased.ID)
	if err != nil {
		t.Fatalf("tombstone row: %v", err)
	}
	if tomb.Email == address || tomb.Bio != "" || tomb.AvatarURL != nil || len(tomb.Keywords) != 0 {
		t.Error("account email, bio, avatar, or keywords were not overwritten")
	}
	var storedHash string
	if err := audit.Pool.QueryRow(ctx, `SELECT password_hash FROM insideout.users WHERE id = $1`, erased.ID).Scan(&storedHash); err != nil {
		t.Fatalf("read hash: %v", err)
	}
	if ok, err := auth.VerifyPassword(storedHash, "correct-horse"); err != nil || ok {
		t.Errorf("tombstone hash still verifies (ok=%v err=%v), want an unusable hash", ok, err)
	}
	if ownerCount(t, audit, `SELECT count(*) FROM insideout.ideas WHERE author_id = $1`, erased.ID) != 0 {
		t.Error("ideas of every status were not deleted")
	}
	if ownerCount(t, audit, `SELECT count(*) FROM insideout.agent_conversations WHERE user_id = $1`, erased.ID) != 0 {
		t.Error("conversations were not deleted")
	}
	if ownerCount(t, audit, `SELECT count(*) FROM insideout.ai_runs WHERE user_id = $1`, erased.ID) != 0 {
		t.Error("ai runs were not deleted")
	}
	if ownerCount(t, audit, `SELECT count(*) FROM insideout.sessions WHERE user_id = $1`, erased.ID) != 0 {
		t.Error("sessions were not deleted")
	}
	if ownerCount(t, audit, `SELECT count(*) FROM insideout.auth_tokens WHERE user_id = $1`, erased.ID) != 0 {
		t.Error("tokens were not deleted")
	}
	if ownerCount(t, audit, `SELECT count(*) FROM insideout.workspace_memberships WHERE user_id = $1`, erased.ID) != 0 {
		t.Error("memberships were not deleted")
	}
	if ownerCount(t, audit, `SELECT count(*) FROM insideout.workspaces WHERE id = $1`, sole.ID) != 0 {
		t.Error("a workspace with no other member was not deleted")
	}
	if got := ownerUUID(t, audit, `SELECT creator_id FROM insideout.workspaces WHERE id = $1`, shared.ID); got != other.ID {
		t.Errorf("shared workspace creator = %s, want the remaining member %s", got, other.ID)
	}
	if got := ownerUUID(t, audit, `SELECT creator_id FROM insideout.workspaces WHERE id = $1`, joined.ID); got != other.ID {
		t.Errorf("joined workspace creator = %s, want the original creator", got)
	}
	if ownerCount(t, audit, `SELECT count(*) FROM insideout.workspace_memberships WHERE workspace_id = $1 AND user_id = $2`, joined.ID, other.ID) != 1 {
		t.Error("the other member lost the workspace they created")
	}

	gotPRD, err := st.GetPrdForMember(ctx, keptPRD.ID, other.ID)
	if err != nil {
		t.Fatalf("kept prd: %v", err)
	}
	if gotPRD.ID != keptPRD.ID || gotPRD.Title != keptPRD.Title || gotPRD.AuthorID != other.ID || gotPRD.IdeaID == nil || *gotPRD.IdeaID != keptIdea.ID {
		t.Errorf("other member's PRD changed: %+v", gotPRD)
	}
	commits, err := st.ListPrdCommits(ctx, other.ID, keptPRD.ID)
	if err != nil {
		t.Fatalf("kept commits: %v", err)
	}
	if len(commits) != 1 || commits[0].ID != keptCommit.ID || commits[0].Name != "kept-commit" || commits[0].CommittedBy != other.ID {
		t.Errorf("other member's commit changed: %+v", commits)
	}
	ownAfter, err := st.GetPrdForMember(ctx, ownPRD.ID, other.ID)
	if err != nil {
		t.Fatalf("erased user's shared PRD: %v", err)
	}
	if ownAfter.AuthorID != erased.ID || ownAfter.Title != "own prd" {
		t.Errorf("shared PRD the erased user wrote changed: author %s title %q", ownAfter.AuthorID, ownAfter.Title)
	}
}

func mustConversation(t *testing.T, st *Store, userID, wsID uuid.UUID, title, body string) uuid.UUID {
	t.Helper()
	idea, err := st.CreateIdea(context.Background(), userID, wsID, title, body)
	if err != nil {
		t.Fatalf("idea %s: %v", title, err)
	}
	_, conv, err := st.ConvertIdea(context.Background(), userID, idea.ID)
	if err != nil {
		t.Fatalf("convert %s: %v", title, err)
	}
	return conv.ID
}

// ageConversation moves the conversation, and its messages when ageMessages
// is set, back by days. An empty conversation is dated by its own creation.
func ageConversation(t *testing.T, st *Store, userID, convID uuid.UUID, days int, ageMessages bool) {
	t.Helper()
	err := st.withUserContext(context.Background(), userID, func(tx pgx.Tx) error {
		if ageMessages {
			tag, err := tx.Exec(context.Background(), `
				UPDATE insideout.agent_messages
				SET created_at = now() - make_interval(days => $2)
				WHERE conversation_id = $1`, convID, days)
			if err != nil {
				return err
			}
			if tag.RowsAffected() == 0 {
				return errors.New("no message to age")
			}
		}
		_, err := tx.Exec(context.Background(), `
			UPDATE insideout.agent_conversations
			SET created_at = now() - make_interval(days => $2)
			WHERE id = $1`, convID, days)
		return err
	})
	if err != nil {
		t.Fatalf("age conversation: %v", err)
	}
}

func conversationVisible(t *testing.T, st *Store, userID, convID uuid.UUID) bool {
	t.Helper()
	return rowVisible(t, st, userID, `SELECT EXISTS (SELECT 1 FROM insideout.agent_conversations WHERE id = $1)`, convID)
}

func messageVisible(t *testing.T, st *Store, userID, convID uuid.UUID) bool {
	t.Helper()
	return rowVisible(t, st, userID, `SELECT EXISTS (SELECT 1 FROM insideout.agent_messages WHERE conversation_id = $1)`, convID)
}

func rowVisible(t *testing.T, st *Store, userID uuid.UUID, query string, args ...any) bool {
	t.Helper()
	var exists bool
	err := st.withUserContext(context.Background(), userID, func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(), query, args...).Scan(&exists)
	})
	if err != nil {
		t.Fatalf("visible: %v", err)
	}
	return exists
}
