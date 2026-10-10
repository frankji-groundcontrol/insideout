package mcpserver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// AIDEV-04: the commit handler parses an "unresolved" array but the tool
// schema never declared it, so schema-abiding MCP clients dropped the human's
// open questions while the CLI's --unresolved flag kept working.
func TestCommitToolSchemaDeclaresUnresolved(t *testing.T) {
	msg := Build("http://unused.test").HandleMessage(context.Background(),
		json.RawMessage(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	raw, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Result struct {
			Tools []struct {
				Name        string `json:"name"`
				InputSchema struct {
					Properties map[string]struct {
						Type  string         `json:"type"`
						Items map[string]any `json:"items"`
					} `json:"properties"`
				} `json:"inputSchema"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("tools/list answer %s: %v", raw, err)
	}
	for _, tool := range out.Result.Tools {
		if tool.Name != "commit" {
			continue
		}
		prop, ok := tool.InputSchema.Properties["unresolved"]
		if !ok {
			t.Errorf("commit schema properties = %v, want an \"unresolved\" argument declared", tool.InputSchema.Properties)
			return
		}
		if prop.Type != "array" {
			t.Errorf("commit unresolved type = %q, want array", prop.Type)
		}
		if prop.Items["type"] != "string" {
			t.Errorf("commit unresolved items = %v, want string items", prop.Items)
		}
		return
	}
	t.Fatalf("commit tool not found in tools/list")
}

// The parse side: a well-formed unresolved list must reach the commit POST.
func TestCommitPassesUnresolvedItems(t *testing.T) {
	var gotPath, gotBody string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		w.Write([]byte(`{"id":"c1"}`))
	}))
	t.Cleanup(api.Close)

	res := drive(t, api.URL, []string{
		toolCall(1, "commit", `{"prd_id":"p1","name":"v1","audience":"decision","summary":"s","unresolved":["open q1","open q2"]}`),
	})
	if res[1].Result.IsError {
		t.Fatalf("commit failed: %+v", res[1])
	}
	if gotPath != "/prds/p1/commit" {
		t.Errorf("path = %q, want /prds/p1/commit", gotPath)
	}
	if !strings.Contains(gotBody, `"unresolved":["open q1","open q2"]`) {
		t.Errorf("body = %q, want both unresolved items carried into the commit", gotBody)
	}
}
