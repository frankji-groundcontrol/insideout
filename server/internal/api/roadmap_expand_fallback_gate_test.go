package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/agent"
)

// An expand that the planner answered from its built-in template because the
// provider failed still serves the template over HTTP, but the gate's done
// must hear the provider error: a fallback that reports success never trips
// the circuit breaker through roadmap expands and can close a half-open
// circuit while the provider is still down.

// providerDownStreamer is a ChatStreamer whose provider call always fails
// with the given classified error.
type providerDownStreamer struct{ err error }

func (s providerDownStreamer) StreamChat(context.Context, string, []agent.Message, []agent.Tool, func(string)) (agent.Turn, error) {
	return agent.Turn{}, s.err
}

func (s providerDownStreamer) StreamChatForcingTool(context.Context, string, []agent.Message, agent.Tool, func(string)) (agent.Turn, error) {
	return agent.Turn{}, s.err
}

func (providerDownStreamer) Model() string { return "provider-down-stub" }

func TestExpandFallbackReportsProviderErrorToTheGate(t *testing.T) {
	f := newGateFixture(t)
	gate := &gateStub{allow: true}
	planner := agent.NewAnthropicRoadmapPlanner(providerDownStreamer{
		err: fmt.Errorf("upstream said no: %w", agent.ErrProviderTransient),
	})
	ts := f.serve(t, gate, planner)

	code, body := f.expand(t, ts)
	if code != http.StatusOK {
		t.Fatalf("expand during a provider outage = %d, want 200 with the template; body %s", code, body)
	}
	if !strings.Contains(body, "明确验收标准") {
		t.Fatalf("expand during a provider outage served %s, want the built-in template's subtasks", body)
	}
	if len(gate.dones) != 1 || !errors.Is(gate.dones[0], agent.ErrProviderTransient) {
		t.Fatalf("done calls = %v, want the provider failure once so the breaker counts it", gate.dones)
	}
}
