package api

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/agent"
	"github.com/frankji-groundcontrol/insideout/server/internal/auth"
	"github.com/frankji-groundcontrol/insideout/server/internal/config"
	"github.com/frankji-groundcontrol/insideout/server/internal/httpx"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
	"github.com/google/uuid"
)

// The build and expand endpoints spend LLM budget just like a coach turn, so
// they must pass the same rate-limit, circuit and permit gate before planning.

// The gate is found by an optional type assertion, so a signature drift on
// Coach.Admit would silently ungate both routes. This keeps it a compile error.
var _ AIGate = (*agent.Coach)(nil)

type gateFixture struct {
	st     *store.Store
	tokens *auth.TokenIssuer
	token  string
	wsID   uuid.UUID
	prdID  uuid.UUID
	nodeID uuid.UUID
}

// newGateFixture is a fresh user with a workspace, a PRD to build and a roadmap
// node to expand, all in real PostgreSQL.
func newGateFixture(t *testing.T) gateFixture {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	ctx := context.Background()
	st, err := store.Open(ctx, url)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(st.Close)

	user, err := st.CreateUser(ctx, "ai-gate-"+uuid.NewString()+"@test.local", "x", "ai-gate")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	ws, err := st.CreateWorkspace(ctx, user.ID, "AI gate WS", "")
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	idea, err := st.CreateIdea(ctx, user.ID, ws.ID, "Gated planning", "content")
	if err != nil {
		t.Fatalf("create idea: %v", err)
	}
	prd, _, err := st.ConvertIdea(ctx, user.ID, idea.ID)
	if err != nil {
		t.Fatalf("convert idea: %v", err)
	}
	project, err := st.EnsureProjectForPrd(ctx, user.ID, prd.ID)
	if err != nil {
		t.Fatalf("ensure project: %v", err)
	}
	node, err := st.CreateRoadmapNode(ctx, user.ID, project.ID, nil, "Root", "")
	if err != nil {
		t.Fatalf("create node: %v", err)
	}
	tokens := auth.NewTokenIssuer("a-32-byte-or-longer-test-secret!", time.Hour)
	token, err := tokens.MintAccessToken(user.ID)
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	return gateFixture{st: st, tokens: tokens, token: token, wsID: ws.ID, prdID: prd.ID, nodeID: node.ID}
}

func (f gateFixture) serve(t *testing.T, coach Coach, planner RoadmapPlanner) *httptest.Server {
	t.Helper()
	srv := NewServer(f.st, f.tokens, &config.Config{}, slog.New(slog.NewTextHandler(io.Discard, nil)), coach, planner)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

func (f gateFixture) build(t *testing.T, ts *httptest.Server) (int, string) {
	return f.post(t, ts, "/api/v1/prds/"+f.prdID.String()+"/build")
}

func (f gateFixture) expand(t *testing.T, ts *httptest.Server) (int, string) {
	return f.post(t, ts, "/api/v1/roadmap/"+f.nodeID.String()+"/expand")
}

func (f gateFixture) post(t *testing.T, ts *httptest.Server, path string) (int, string) {
	t.Helper()
	resp := doJSON(t, ts, f.token, http.MethodPost, path, "")
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
}

// countingPlanner records how many times the LLM would have been called.
type countingPlanner struct {
	mu             sync.Mutex
	plans, expands int
	err            error
}

func (p *countingPlanner) PlanMVP(context.Context, string, map[string]string) (*store.RoadmapPlanNode, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.plans++
	return &store.RoadmapPlanNode{Title: "root"}, p.err
}

func (p *countingPlanner) ExpandNode(context.Context, string, string, string) ([]store.RoadmapPlanNode, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.expands++
	return []store.RoadmapPlanNode{{Title: "child"}}, p.err
}

func (p *countingPlanner) counts() (plans, expands int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.plans, p.expands
}

// gateStub is a Coach that also gates: it answers every Admit as scripted and
// records what it was asked and what each done was told.
type gateStub struct {
	allow bool

	mu         sync.Mutex
	labels     []string
	workspaces []uuid.UUID
	dones      []error
}

func (*gateStub) HandleMessage(http.ResponseWriter, *http.Request, string, string, string, string) {}

func (g *gateStub) Admit(w http.ResponseWriter, _ *http.Request, workspaceID, _ uuid.UUID, label string) (func(error), bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.labels = append(g.labels, label)
	g.workspaces = append(g.workspaces, workspaceID)
	if !g.allow {
		httpx.WriteError(w, http.StatusTooManyRequests, "Rate limit exceeded", "APP_THROTTLE", nil)
		return nil, false
	}
	return func(err error) {
		g.mu.Lock()
		defer g.mu.Unlock()
		g.dones = append(g.dones, err)
	}, true
}

func TestBuildAndExpandAskTheAIGateBeforePlanning(t *testing.T) {
	f := newGateFixture(t)

	t.Run("a refused call never reaches the planner", func(t *testing.T) {
		gate, planner := &gateStub{allow: false}, &countingPlanner{}
		ts := f.serve(t, gate, planner)

		if code, body := f.build(t, ts); code != http.StatusTooManyRequests {
			t.Fatalf("build past a refusing gate = %d, want 429; body %s", code, body)
		}
		if code, body := f.expand(t, ts); code != http.StatusTooManyRequests {
			t.Fatalf("expand past a refusing gate = %d, want 429; body %s", code, body)
		}
		if plans, expands := planner.counts(); plans != 0 || expands != 0 {
			t.Fatalf("the planner ran %d plan(s) and %d expand(s) after the gate refused", plans, expands)
		}
	})

	t.Run("an admitted call plans once and reports back once", func(t *testing.T) {
		gate, planner := &gateStub{allow: true}, &countingPlanner{}
		ts := f.serve(t, gate, planner)

		if code, body := f.build(t, ts); code != http.StatusOK {
			t.Fatalf("admitted build = %d, want 200; body %s", code, body)
		}
		if code, body := f.expand(t, ts); code != http.StatusOK {
			t.Fatalf("admitted expand = %d, want 200; body %s", code, body)
		}
		if plans, expands := planner.counts(); plans != 1 || expands != 1 {
			t.Fatalf("planner ran %d plan(s) and %d expand(s), want 1 and 1", plans, expands)
		}
		if want := []string{"roadmap:plan-mvp", "roadmap:expand"}; !slices.Equal(gate.labels, want) {
			t.Fatalf("admit labels = %v, want %v", gate.labels, want)
		}
		if len(gate.dones) != 2 || gate.dones[0] != nil || gate.dones[1] != nil {
			t.Fatalf("done calls = %v, want one nil per admitted call", gate.dones)
		}
		for _, ws := range gate.workspaces {
			if ws != f.wsID {
				t.Fatalf("admitted against workspace %s, want the PRD's workspace %s", ws, f.wsID)
			}
		}
	})

	t.Run("a planner failure is handed to done", func(t *testing.T) {
		boom := errors.New("model said no")
		gate, planner := &gateStub{allow: true}, &countingPlanner{err: boom}
		ts := f.serve(t, gate, planner)

		if code, _ := f.build(t, ts); code != http.StatusInternalServerError {
			t.Fatalf("build with a failing planner = %d, want 500", code)
		}
		if code, _ := f.expand(t, ts); code != http.StatusInternalServerError {
			t.Fatalf("expand with a failing planner = %d, want 500", code)
		}
		if len(gate.dones) != 2 || !errors.Is(gate.dones[0], boom) || !errors.Is(gate.dones[1], boom) {
			t.Fatalf("done calls = %v, want the planner's error twice", gate.dones)
		}
	})
}

// With the real coach in front, build and expand draw on the same per-user
// budget as coach turns (10 a minute): the eleventh call is throttled with the
// coach's own 429 and never reaches the planner.
func TestBuildAndExpandCountTowardTheCoachRateLimit(t *testing.T) {
	f := newGateFixture(t)
	planner := &countingPlanner{}
	coach := agent.New(f.st, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := f.serve(t, coach, planner)

	for i := 0; i < 5; i++ {
		if code, body := f.build(t, ts); code != http.StatusOK {
			t.Fatalf("build %d = %d, want 200; body %s", i+1, code, body)
		}
		if code, body := f.expand(t, ts); code != http.StatusOK {
			t.Fatalf("expand %d = %d, want 200; body %s", i+1, code, body)
		}
	}
	for name, call := range map[string]func(*testing.T, *httptest.Server) (int, string){"build": f.build, "expand": f.expand} {
		code, body := call(t, ts)
		if code != http.StatusTooManyRequests || !strings.Contains(body, "APP_THROTTLE") {
			t.Fatalf("%s after 10 gated calls = %d %s, want 429 APP_THROTTLE", name, code, body)
		}
	}
	if plans, expands := planner.counts(); plans != 5 || expands != 5 {
		t.Fatalf("planner ran %d plan(s) and %d expand(s), want 5 and 5: throttled calls must not reach it", plans, expands)
	}
}
