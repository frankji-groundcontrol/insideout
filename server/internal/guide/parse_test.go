package guide

import (
	"testing"
)

const sample = `version: 1
nodes:
  11111111-1111-1111-1111-111111111111:
    title: "A"
    branches: [main, "feature/*"]
    labels: [roadmap/a]
    paths: ["server/"]
  22222222-2222-2222-2222-222222222222:
    title: "B"
    branches: []
    labels: [roadmap/b]
    paths: []
`

func TestParseAndMatch(t *testing.T) {
	nodes, err := Parse([]byte(sample))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("got %d nodes", len(nodes))
	}
	yes := func(string) bool { return true }

	for _, tc := range []struct {
		name          string
		branch        string
		labels, paths []string
		want          []string
	}{
		{"exact branch", "main", nil, nil, []string{"11111111-1111-1111-1111-111111111111"}},
		{"prefix branch", "feature/x", nil, nil, []string{"11111111-1111-1111-1111-111111111111"}},
		{"no prefix spill", "features/x", nil, nil, nil},
		{"label", "other", []string{"roadmap/b"}, nil, []string{"22222222-2222-2222-2222-222222222222"}},
		{"path prefix", "dev", nil, []string{"server/api/x.go"}, []string{"11111111-1111-1111-1111-111111111111"}},
		{"nothing", "dev", nil, nil, nil},
	} {
		got := Match(nodes, tc.branch, tc.labels, tc.paths, yes)
		if len(got) != len(tc.want) {
			t.Errorf("%s: got %v want %v", tc.name, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("%s: got %v want %v", tc.name, got, tc.want)
			}
		}
	}

	leaf := func(id string) bool { return id == "22222222-2222-2222-2222-222222222222" }
	if got := Match(nodes, "main", nil, nil, leaf); len(got) != 0 {
		t.Errorf("branch node matched despite leaf filter: %v", got)
	}
}

func TestParseRejects(t *testing.T) {
	if _, err := Parse([]byte("version: 2\nnodes: {}\n")); err == nil {
		t.Error("version 2 accepted")
	}
	if _, err := Parse([]byte("nodes: {}\n")); err == nil {
		t.Error("missing version accepted")
	}
	if _, err := Parse([]byte("version: 1\n")); err == nil {
		t.Error("missing nodes accepted")
	}
	if _, err := Parse([]byte(":::not yaml")); err == nil {
		t.Error("garbage accepted")
	}
}
