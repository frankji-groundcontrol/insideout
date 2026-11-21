package store

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestResetWatchCompletesOnlyForItsToken(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	linked, err := st.CreateResetWatch(ctx, "hash-"+uuid.NewString(), time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("create linked watch: %v", err)
	}
	phantom, err := st.CreateResetWatch(ctx, "", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("create phantom watch: %v", err)
	}
	if got, err := st.ResetWatchStatus(ctx, linked); err != nil || got != "pending" {
		t.Fatalf("linked status = %q, %v", got, err)
	}
	if got, _ := st.ResetWatchStatus(ctx, phantom); got != "pending" {
		t.Fatalf("phantom status = %q", got)
	}
	if got, _ := st.ResetWatchStatus(ctx, uuid.New()); got != "expired" {
		t.Fatalf("unknown status = %q, want expired", got)
	}

	var hash string
	if err := st.Pool.QueryRow(ctx, `SELECT token_hash FROM insideout.reset_watches WHERE id = $1`, linked).Scan(&hash); err != nil {
		t.Fatalf("read hash: %v", err)
	}
	if err := st.CompleteResetWatch(ctx, hash); err != nil {
		t.Fatalf("complete: %v", err)
	}
	if got, _ := st.ResetWatchStatus(ctx, linked); got != "completed" {
		t.Fatalf("after complete = %q", got)
	}
	if got, _ := st.ResetWatchStatus(ctx, phantom); got != "pending" {
		t.Fatalf("phantom changed to %q", got)
	}
}

func TestResetWatchExpires(t *testing.T) {
	st := newTestStore(t)
	id, err := st.CreateResetWatch(context.Background(), "", time.Now().Add(-time.Minute))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if got, _ := st.ResetWatchStatus(context.Background(), id); got != "expired" {
		t.Fatalf("status = %q, want expired", got)
	}
}

// COMP-04: the reaper's retention purge clears watches that expired before
// the window and keeps live ones.
func TestReapStaleAIRuns_PurgesExpiredResetWatches(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	old, err := st.CreateResetWatch(ctx, "", time.Now().Add(-45*24*time.Hour))
	if err != nil {
		t.Fatalf("create expired watch: %v", err)
	}
	live, err := st.CreateResetWatch(ctx, "", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("create live watch: %v", err)
	}
	if _, err := st.ReapStaleAIRuns(ctx, time.Hour); err != nil {
		t.Fatalf("reap: %v", err)
	}
	const q = `SELECT EXISTS (SELECT 1 FROM insideout.reset_watches WHERE id = $1)`
	if existsRow(t, st, q, old) {
		t.Error("a watch that expired 45 days ago survived the purge")
	}
	if !existsRow(t, st, q, live) {
		t.Error("the purge deleted a live watch")
	}
}
