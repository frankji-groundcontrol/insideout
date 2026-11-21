package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

var updateGolden = flag.Bool("update", false, "rewrite testdata/tools_*.golden.json from the live registry")

// toolsJSON is tools/list as the client sees it, one tool per entry, sorted by
// name and indented, so a moved registration that changes a name, a description
// or an argument shows up as a diff.
func toolsJSON(t *testing.T) []byte {
	t.Helper()
	raw, err := json.Marshal(Build(exampleAPI).HandleMessage(context.Background(), json.RawMessage(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)))
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Result struct {
			Tools []map[string]any `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("tools/list answer %s: %v", raw, err)
	}
	tools := out.Result.Tools
	sort.Slice(tools, func(i, j int) bool { return tools[i]["name"].(string) < tools[j]["name"].(string) })
	b, err := json.MarshalIndent(tools, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(b, '\n')
}

// The registry a process offers, with no login and with one (the agent profile
// withholds the password tools, the filter the session-swap tools), pinned
// byte for byte. Run go test -run TestToolRegistryGolden -update to accept a
// deliberate change.
func TestToolRegistryGolden(t *testing.T) {
	for _, tc := range []struct{ name, token string }{{"nologin", ""}, {"login", humanJWT()}} {
		t.Run(tc.name, func(t *testing.T) {
			pinConfigHome(t)
			t.Setenv("INSIDEOUT_TOKEN", tc.token)
			got := toolsJSON(t)
			path := filepath.Join("testdata", "tools_"+tc.name+".golden.json")
			if *updateGolden {
				if err := os.WriteFile(path, got, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("tools/list (%s) differs from %s; rerun with -update if the change is intended", tc.name, path)
			}
		})
	}
}
