package api

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/agent"
	"github.com/frankji-groundcontrol/insideout/server/internal/store"
)

// A planner panic must not strand the gate: the recover middleware answers
// 500, and the handler's deferred done still hands the permit back and fails
// the run, so the very next planner call is not starved into the busy 503.

type panickingPlanner struct{ calls int }

func (p *panickingPlanner) PlanMVP(context.Context, string, map[string]string) (*store.RoadmapPlanNode, error) {
	p.calls++
	panic("planner bug")
}

func (p *panickingPlanner) ExpandNode(context.Context, string, string, string) ([]store.RoadmapPlanNode, error) {
	p.calls++
	panic("planner bug")
}

func TestBuildAndExpandReleaseThePermitWhenThePlannerPanics(t *testing.T) {
	// One permit for the whole coach: a leaked one starves every later call.
	t.Setenv("INSIDEOUT_AI_MAX_CONCURRENT", "1")
	t.Setenv("INSIDEOUT_AI_QUEUE_WAIT", "150ms")
	f := newGateFixture(t)
	coach := agent.New(f.st, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	boomTS := f.serve(t, coach, &panickingPlanner{})
	planner := &countingPlanner{}
	okTS := f.serve(t, coach, planner)

	lastRunStatus := func(t *testing.T) string {
		t.Helper()
		var status string
		if err := f.st.Pool.QueryRow(context.Background(),
			`SELECT status FROM insideout.ai_runs WHERE workspace_id = $1 ORDER BY created_at DESC, ctid DESC LIMIT 1`,
			f.wsID).Scan(&status); err != nil {
			t.Fatalf("read the last run: %v", err)
		}
		return status
	}

	t.Run("build", func(t *testing.T) {
		if code, body := f.build(t, boomTS); code != http.StatusInternalServerError {
			t.Fatalf("build with a panicking planner = %d, want the recover middleware's 500; body %s", code, body)
		}
		if status := lastRunStatus(t); status != "failed" {
			t.Fatalf("the panicked build's run = %q, want failed: done must still run", status)
		}
		if code, body := f.build(t, okTS); code != http.StatusOK {
			t.Fatalf("the build after a panicked planner = %d, want 200: the permit must come back, not the busy 503; body %s", code, body)
		}
		if plans, _ := planner.counts(); plans != 1 {
			t.Fatalf("the surviving planner ran %d plan(s), want 1", plans)
		}
	})

	t.Run("expand", func(t *testing.T) {
		if code, body := f.expand(t, boomTS); code != http.StatusInternalServerError {
			t.Fatalf("expand with a panicking planner = %d, want the recover middleware's 500; body %s", code, body)
		}
		if status := lastRunStatus(t); status != "failed" {
			t.Fatalf("the panicked expand's run = %q, want failed: done must still run", status)
		}
		if code, body := f.expand(t, okTS); code != http.StatusOK {
			t.Fatalf("the expand after a panicked planner = %d, want 200: the permit must come back, not the busy 503; body %s", code, body)
		}
		if _, expands := planner.counts(); expands != 1 {
			t.Fatalf("the surviving planner ran %d expand(s), want 1", expands)
		}
	})
}
