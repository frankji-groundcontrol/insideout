package api

import (
	"bufio"
	"context"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
)

// REL-03 / SEC-08: a passive stream authorised its viewer once, when it opened. The
// presence heartbeat only refreshed the viewer's entry and the progress stream
// only re-read on a node change, so a member removed from the workspace kept
// receiving the project's live data, and kept appearing in its presence list,
// until their tab closed. The streams must notice on the heartbeat and end.
// DB-gated: membership is enforced by RLS in real PostgreSQL.

// shortHeartbeat makes the streams ping (and so recheck) every d.
func shortHeartbeat(t *testing.T, d time.Duration) {
	t.Helper()
	old := streamHeartbeat
	streamHeartbeat = d
	t.Cleanup(func() { streamHeartbeat = old })
}

// streamEnded reports, on the returned channel, when the server closes the
// stream. It consumes br, so the caller must not read it afterwards.
func streamEnded(br *bufio.Reader) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, err := readFrame(br); err != nil {
				return
			}
		}
	}()
	return done
}

func TestStreamRecheck_RemovedMemberLosesBothStreams(t *testing.T) {
	fx := newStreamFixture(t)
	shortHeartbeat(t, 50*time.Millisecond)
	presence, _ := openSSE(t, fx.ts, fx.tokenB, fx.presencePath("?client=tab"))
	progress, _ := openSSE(t, fx.ts, fx.tokenB, fx.progressPath())
	fx.snapshotOf(t, presence)                      // registered
	frameWithPrefix(t, progress, "event: progress") // connected

	// A, the workspace admin, removes B.
	if err := fx.srv.store.RemoveMember(context.Background(), fx.a.ID, fx.proj.WorkspaceID, fx.b.ID); err != nil {
		t.Fatalf("remove member: %v", err)
	}

	wait, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	for _, s := range []struct {
		name  string
		ended <-chan struct{}
	}{{"presence", streamEnded(presence)}, {"progress", streamEnded(progress)}} {
		select {
		case <-s.ended:
		case <-wait.Done():
			t.Errorf("the removed member's %s stream is still open 3s after the removal: a stream is authorised once, when it opens, and never again", s.name)
		}
	}
	for _, e := range fx.listPresence(t, fx.tokenA) {
		if e.UserID == fx.b.ID.String() {
			t.Errorf("the removed member is still in the project's presence list: %+v", e)
		}
	}
}

// SEC-08: the access token was checked once, when the stream opened, so a
// stream outlived the session that opened it: an expired (or revoked) token
// kept a tab connected for as long as the tab stayed open, with no lifetime at
// all. A stream may live as long as its access token and no longer; the client
// reconnects with a refreshed one. The per-user stream cap this finding also
// asked for is rv4-REL-01's (stream_caps_test.go).
func TestStreamRecheck_StreamsEndWhenTheirAccessTokenExpires(t *testing.T) {
	fx := newStreamFixture(t)
	shortHeartbeat(t, 50*time.Millisecond)
	// The fixture's secret with a 3s lifetime: valid when the streams open,
	// dead (whole-second precision) within 3s.
	short := auth.NewTokenIssuer("a-32-byte-or-longer-test-secret!", 3*time.Second)
	token, err := short.MintAccessToken(fx.a.ID)
	if err != nil {
		t.Fatalf("mint a short-lived token: %v", err)
	}
	presence, _ := openSSE(t, fx.ts, token, fx.presencePath("?client=tab"))
	progress, _ := openSSE(t, fx.ts, token, fx.progressPath())
	fx.snapshotOf(t, presence)
	frameWithPrefix(t, progress, "event: progress")

	wait, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	for _, s := range []struct {
		name  string
		ended <-chan struct{}
	}{{"presence", streamEnded(presence)}, {"progress", streamEnded(progress)}} {
		select {
		case <-s.ended:
		case <-wait.Done():
			t.Errorf("the %s stream is still open 6s after its 3s access token expired: a stream outlives the session that opened it", s.name)
		}
	}
	for _, e := range fx.listPresence(t, fx.tokenA) {
		if e.SessionID == fx.a.ID.String()+":tab" {
			t.Errorf("the expired session is still in the project's presence list: %+v", e)
		}
	}
}
