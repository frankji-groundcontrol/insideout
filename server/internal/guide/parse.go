package guide

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Matchers is one node's insideout.yaml match block.
type Matchers struct {
	Branches []string `yaml:"branches"`
	Labels   []string `yaml:"labels"`
	Paths    []string `yaml:"paths"`
}

type guideFile struct {
	Version int                 `yaml:"version"`
	Nodes   map[string]Matchers `yaml:"nodes"`
}

// Parse validates an insideout.yaml and returns its node matchers.
// Node ids are trusted downstream: writes go through ownership-checked
// store paths, and unknown ids simply never match a real node.
func Parse(data []byte) (map[string]Matchers, error) {
	var g guideFile
	if err := yaml.Unmarshal(data, &g); err != nil {
		return nil, fmt.Errorf("guide: parse: %w", err)
	}
	if g.Version != 1 {
		return nil, fmt.Errorf("guide: unsupported version %d (want 1)", g.Version)
	}
	if g.Nodes == nil {
		return nil, fmt.Errorf("guide: no nodes map")
	}
	return g.Nodes, nil
}

// Match returns the ids of nodes whose matchers hit the event. leaf
// decides whether a node id is a leaf (evidence attaches to leaves
// only). A node matches when ANY of its matchers hits:
//   - branches: exact branch, or prefix when the entry ends in /*
//   - labels:   exact label
//   - paths:    entry is a prefix of any touched path
func Match(nodes map[string]Matchers, branch string, labels, paths []string, leaf func(id string) bool) []string {
	var out []string
	for id, m := range nodes {
		if leaf != nil && !leaf(id) {
			continue
		}
		if branchMatches(m.Branches, branch) ||
			labelMatches(m.Labels, labels) ||
			pathMatches(m.Paths, paths) {
			out = append(out, id)
		}
	}
	return out
}

func branchMatches(entries []string, branch string) bool {
	for _, e := range entries {
		if e == branch {
			return true
		}
		if len(e) > 2 && e[len(e)-2:] == "/*" && len(branch) > len(e)-1 && branch[:len(e)-1] == e[:len(e)-1] {
			return true
		}
	}
	return false
}

func labelMatches(entries, labels []string) bool {
	for _, e := range entries {
		for _, l := range labels {
			if e == l {
				return true
			}
		}
	}
	return false
}

func pathMatches(entries, paths []string) bool {
	for _, e := range entries {
		for _, p := range paths {
			if len(p) >= len(e) && p[:len(e)] == e {
				return true
			}
		}
	}
	return false
}
