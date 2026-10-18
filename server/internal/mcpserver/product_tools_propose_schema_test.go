package mcpserver

import (
	"context"
	"encoding/json"
	"maps"
	"slices"
	"testing"
)

// rv4-AIDEV-10: the propose handler reads an `items` array (objects with a
// title and a parentHint) that a human can apply on acceptance, but the
// registered schema never declared it. A schema-abiding MCP client, or an
// agent that only reads tools/list, could not know structured proposals
// exist while the CLI's --item flag kept working. The schema the server
// advertises must declare the argument the handler consumes.
func TestProposeToolSchemaDeclaresItems(t *testing.T) {
	msg := Build("http://unused.example.org").HandleMessage(context.Background(),
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
						Type        string         `json:"type"`
						Description string         `json:"description"`
						Items       map[string]any `json:"items"`
					} `json:"properties"`
					Required []string `json:"required"`
				} `json:"inputSchema"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("tools/list answer %s: %v", raw, err)
	}

	for _, tool := range out.Result.Tools {
		if tool.Name != "propose" {
			continue
		}
		items, ok := tool.InputSchema.Properties["items"]
		if !ok {
			t.Fatalf("propose declares %v, want an \"items\" argument declared (its handler reads it)",
				slices.Sorted(maps.Keys(tool.InputSchema.Properties)))
		}
		if items.Type != "array" {
			t.Errorf("propose items type = %q, want array", items.Type)
		}
		if items.Description == "" {
			t.Errorf("propose items has no description; an agent reads it to learn the shape")
		}
		if items.Items["type"] != "object" {
			t.Errorf("propose items elements = %v, want objects", items.Items)
		}
		props, _ := items.Items["properties"].(map[string]any)
		for _, name := range []string{"title", "parentHint"} {
			p, _ := props[name].(map[string]any)
			if p["type"] != "string" {
				t.Errorf("propose item property %q = %v, want a string property", name, props[name])
			}
		}
		// The handler drops an item with no title, so the schema says so; the
		// hint stays optional (a top-level node has no parent).
		req, _ := items.Items["required"].([]any)
		if !slices.Contains(req, any("title")) || slices.Contains(req, any("parentHint")) {
			t.Errorf("propose item required = %v, want title required and parentHint optional", req)
		}
		// A proposal without structure is still valid, so the array is optional.
		if slices.Contains(tool.InputSchema.Required, "items") {
			t.Errorf("propose required = %v, want items optional", tool.InputSchema.Required)
		}
		return
	}
	t.Fatalf("propose tool not found in tools/list")
}
