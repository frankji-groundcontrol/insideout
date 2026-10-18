package api

import (
	"context"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// deadlineProbe is a Coach and a RoadmapPlanner that records, for each call,
// how long the request context it was handed has left (-1: no deadline at all).
// The handlers pass r.Context() straight to these collaborators, so what they
// see is what a database call or a model call in the same handler would get.
type deadlineProbe struct {
	mu   sync.Mutex
	left map[string]time.Duration
}

func (p *deadlineProbe) note(call string, ctx context.Context) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.left == nil {
		p.left = map[string]time.Duration{}
	}
	if dl, ok := ctx.Deadline(); ok {
		p.left[call] = time.Until(dl)
	} else {
		p.left[call] = -1
	}
}

func (p *deadlineProbe) seen(call string) (time.Duration, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	left, ok := p.left[call]
	return left, ok
}

func (p *deadlineProbe) HandleMessage(w http.ResponseWriter, r *http.Request, _, _, _, _ string) {
	p.note("message", r.Context())
	w.WriteHeader(http.StatusNoContent)
}

func (p *deadlineProbe) ConfirmStage(ctx context.Context, _, _ uuid.UUID, _ string) error {
	p.note("stage", ctx)
	return store.ErrNotFound
}

func (p *deadlineProbe) PlanMVP(ctx context.Context, _, _ string, _ map[string]string) (*store.RoadmapPlanNode, error) {
	p.note("build", ctx)
	return &store.RoadmapPlanNode{Title: "root"}, nil
}

func (p *deadlineProbe) ExpandNode(ctx context.Context, _, _, _, _ string) ([]store.RoadmapPlanNode, error) {
	p.note("expand", ctx)
	return []store.RoadmapPlanNode{{Title: "child"}}, nil
}

// REL-02: no handler had a request deadline, so a database that stopped
// answering held every plain request until the client gave up. Plain routes
// must run under a deadline that ends before the web client's 30s receive
// timeout; the routes that legitimately run long (the coach's streamed model
// turn, the AI planner calls) must keep none. DB-gated: the routes
// authenticate and check ownership against real RLS before they reach the probe.
func TestRequestDeadline_BoundsPlainRoutesAndSparesTheLongOnes(t *testing.T) {
	f := newGateFixture(t)
	ctx := context.Background()
	uid, _, err := f.tokens.VerifyToken(f.token)
	if err != nil {
		t.Fatalf("read the fixture user: %v", err)
	}
	idea, err := f.st.CreateIdea(ctx, uid, f.wsID, "Deadline probe", "content")
	if err != nil {
		t.Fatalf("create idea: %v", err)
	}
	_, conv, err := f.st.ConvertIdea(ctx, uid, idea.ID)
	if err != nil {
		t.Fatalf("convert idea (creates the conversation): %v", err)
	}
	probe := &deadlineProbe{}
	ts := f.serve(t, probe, probe)

	for _, c := range []struct{ path, body string }{
		{"/api/v1/conversations/" + conv.ID.String() + "/stage", `{"stage":"draft"}`},
		{"/api/v1/conversations/" + conv.ID.String() + "/messages", `{"content":"hello"}`},
		{"/api/v1/prds/" + f.prdID.String() + "/build", ""},
		{"/api/v1/roadmap/" + f.nodeID.String() + "/expand", ""},
	} {
		resp := doJSON(t, ts, f.token, http.MethodPost, c.path, c.body)
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	left, ok := probe.seen("stage")
	switch {
	case !ok:
		t.Error("the stage confirmation never reached the coach")
	case left <= 0:
		t.Error("a plain route (confirm stage) ran with no request deadline: a stalled database holds it until the client gives up")
	case left > 30*time.Second:
		t.Errorf("a plain route's deadline is %s away, want less than the web client's 30s receive timeout", left.Round(time.Second))
	}
	for _, call := range []string{"message", "build", "expand"} {
		left, ok := probe.seen(call)
		switch {
		case !ok:
			t.Errorf("the %s call never reached its collaborator", call)
		case left != -1:
			t.Errorf("the %s call ran under a request deadline (%s left): it is a model call and must not be cut", call, left.Round(time.Second))
		}
	}
}
