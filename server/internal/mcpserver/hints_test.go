package mcpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
)

// rv5-PARA-04: while a login is held, tools/list hides auth_login_code_verify,
// auth_magic_link_consume and auth_logout, so no error may send the model to
// them. A refused in-process login and a refused identity swap both say to
// restart the process instead.
func TestHintsNameNoToolTheListHides(t *testing.T) {
	pinConfigHome(t)
	t.Cleanup(func() { setSessionToken("") })
	expired := fakeJWT("", "expired", time.Now().Add(-time.Hour), 15*time.Minute)
	api, _ := routedAPI(t, func(w http.ResponseWriter, r *http.Request, _ int) bool {
		if r.URL.Path == "/agent/token" || r.URL.Path == "/me" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized"}`))
			return true
		}
		return false
	})
	setSessionToken(expired)
	s := Build(api.URL)
	offered := offeredTools(t, s)

	_, err := client(api.URL).Whoami()
	if err == nil {
		t.Fatal("whoami on a refused login succeeded")
	}
	swap := resultText(t, callTool(t, s, 1, "auth_login_code_verify", `{"email":"a@example.com","code":"123456"}`))
	for _, text := range []string{err.Error(), swap} {
		for _, hidden := range sessionSwapTools {
			if !slices.Contains(offered, hidden) && strings.Contains(text, hidden) {
				t.Errorf("%q names %s, which tools/list does not offer while a login is held", text, hidden)
			}
		}
		if !strings.Contains(text, "restart insideout-mcp") {
			t.Errorf("%q does not say to restart insideout-mcp", text)
		}
	}
}

// People-in-charge mode (2026-11-22): the instructions send an agent to its
// human's work first, roadmap_mine before roadmap_steps and agent_context on
// that root, and every tool they name there is one the list offers.
func TestTheInstructionsStartFromTheHumansWork(t *testing.T) {
	pinConfigHome(t)
	t.Setenv("INSIDEOUT_TOKEN", humanJWT())
	s := Build(exampleAPI)
	raw, err := json.Marshal(s.HandleMessage(context.Background(), json.RawMessage(
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"t","version":"0"}}}`)))
	if err != nil {
		t.Fatal(err)
	}
	var res struct {
		Result struct{ Instructions string } `json:"result"`
	}
	if err := json.Unmarshal(raw, &res); err != nil || res.Result.Instructions == "" {
		t.Fatalf("initialize answered %s", raw)
	}
	text, offered := res.Result.Instructions, offeredTools(t, s)
	at := map[string]int{}
	for _, tool := range []string{"roadmap_mine", "roadmap_steps", "agent_context"} {
		at[tool] = strings.Index(text, tool)
		if at[tool] < 0 || !slices.Contains(offered, tool) {
			t.Errorf("the instructions must name %s, and tools/list must offer it: %q", tool, text)
		}
	}
	if at["roadmap_mine"] > at["roadmap_steps"] || at["roadmap_mine"] > at["agent_context"] {
		t.Errorf("the instructions must start from roadmap_mine: %q", text)
	}
}
