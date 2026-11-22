package roadmaptime

import (
	"reflect"
	"testing"
)

// tree builds StepNodes from (id, parent) pairs in position order among
// siblings, and links (waiting id → ids it waits on).
func tree(pairs [][2]string, links map[string][]string) []StepNode {
	pos := map[string]int{}
	out := make([]StepNode, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, StepNode{ID: p[0], ParentID: p[1], Position: pos[p[1]], DependsOn: links[p[0]]})
		pos[p[1]]++
	}
	return out
}

// sparkTree is the top of scripts/demo/spark_story.py: the project root
// "Spark MVP", its five branches, their work, and the eight DEPENDENCIES
// (after the seed replaced the Matching → Launch link that made a loop).
func sparkTree() []StepNode {
	return tree([][2]string{
		{"spark", ""},
		{"onboarding", "spark"},
		{"signup", "onboarding"}, {"profile", "onboarding"}, {"photo", "profile"}, {"prompts", "profile"},
		{"selfie", "onboarding"}, {"copy", "onboarding"},
		{"matching", "spark"},
		{"engine", "matching"}, {"embeddings", "engine"}, {"scoring", "engine"}, {"explain", "engine"},
		{"limit", "matching"}, {"drop", "matching"}, {"dashboard", "matching"},
		{"chat", "spark"},
		{"realtime", "chat"}, {"starters", "chat"}, {"unmatch", "chat"},
		{"safety", "spark"},
		{"report", "safety"}, {"reportui", "report"}, {"modqueue", "report"},
		{"block", "safety"}, {"photocheck", "safety"}, {"center", "safety"},
		{"launch", "spark"},
		{"beta", "launch"}, {"store", "launch"}, {"metrics", "launch"}, {"privacy", "launch"},
	}, map[string][]string{
		"beta":      {"engine", "realtime", "report", "profile"},
		"store":     {"selfie"},
		"drop":      {"scoring"},
		"starters":  {"embeddings"},
		"dashboard": {"drop"},
	})
}

func ids(steps []Step) []string {
	out := make([]string, len(steps))
	for i, s := range steps {
		out[i] = s.ID
	}
	return out
}

func TestStepIDsPicksTheLevel(t *testing.T) {
	spark := sparkTree()
	for _, tc := range []struct {
		name, root, wantRoot string
		nodes                []StepNode
		want                 []string
	}{
		{"one project root opens it", "", "spark", spark, []string{"onboarding", "matching", "chat", "safety", "launch"}},
		{"a branch shows its children", "profile", "profile", spark, []string{"photo", "prompts"}},
		{"a leaf is its own step", "photo", "photo", spark, []string{"photo"}},
		{"several roots are the steps", "", "", tree([][2]string{{"a", ""}, {"b", ""}, {"a1", "a"}}, nil), []string{"a", "b"}},
		{"a lone leaf root is its own step", "", "solo", tree([][2]string{{"solo", ""}}, nil), []string{"solo"}},
		{"no nodes, no steps", "", "", nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, got := StepIDs(tc.nodes, tc.root)
			if root != tc.wantRoot || !reflect.DeepEqual(got, tc.want) {
				t.Errorf("StepIDs(%q) = %q %v, want %q %v", tc.root, root, got, tc.wantRoot, tc.want)
			}
		})
	}
}

// The Spark top level: three teams start together, Chat waits on Matching's
// embeddings, Launch waits on all four; no loop.
func TestStepsOrdersTheSparkTopLevel(t *testing.T) {
	nodes := sparkTree()
	_, stepIDs := StepIDs(nodes, "")
	got := Steps(nodes, stepIDs)
	want := []Step{
		{ID: "onboarding", Stage: 0, After: []string{}, Links: map[string]int{}, ItemsInside: 6, WaitsOutside: []string{}},
		{ID: "matching", Stage: 0, After: []string{}, Links: map[string]int{}, ItemsInside: 7, WaitsOutside: []string{}},
		{ID: "safety", Stage: 0, After: []string{}, Links: map[string]int{}, ItemsInside: 6, WaitsOutside: []string{}},
		{ID: "chat", Stage: 1, After: []string{"matching"}, Links: map[string]int{"matching": 1}, ItemsInside: 3, WaitsOutside: []string{}},
		{ID: "launch", Stage: 2, After: []string{"onboarding", "matching", "safety", "chat"},
			Links: map[string]int{"onboarding": 2, "matching": 1, "safety": 1, "chat": 1}, ItemsInside: 4, WaitsOutside: []string{}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Spark steps:\n got %+v\nwant %+v", got, want)
	}
}

func loopOf(i int) *int { return &i }

func TestStepsGroupsLoopsAndDropsWhatTheTreeSays(t *testing.T) {
	for _, tc := range []struct {
		name  string
		nodes []StepNode
		want  []Step
	}{
		{"two steps wait on each other through their work",
			tree([][2]string{{"r", ""}, {"a", "r"}, {"b", "r"}, {"c", "r"}, {"a1", "a"}, {"a2", "a"}, {"b1", "b"}},
				map[string][]string{"a1": {"b1"}, "b1": {"a2"}, "c": {"b"}}),
			[]Step{
				{ID: "a", Stage: 0, Loop: loopOf(0), After: []string{}, Links: map[string]int{}, ItemsInside: 2, WaitsOutside: []string{}},
				{ID: "b", Stage: 0, Loop: loopOf(0), After: []string{}, Links: map[string]int{}, ItemsInside: 1, WaitsOutside: []string{}},
				{ID: "c", Stage: 1, After: []string{"b"}, Links: map[string]int{"b": 1}, WaitsOutside: []string{}},
			}},
		{"three steps in a ring share a stage after a step before them",
			tree([][2]string{{"r", ""}, {"x", "r"}, {"a", "r"}, {"b", "r"}, {"c", "r"}, {"a1", "a"}, {"b1", "b"}, {"c1", "c"}},
				map[string][]string{"a1": {"c1", "x"}, "b1": {"a1"}, "c1": {"b1"}}),
			[]Step{
				{ID: "x", Stage: 0, After: []string{}, Links: map[string]int{}, WaitsOutside: []string{}},
				{ID: "a", Stage: 1, Loop: loopOf(0), After: []string{"x"}, Links: map[string]int{"x": 1}, ItemsInside: 1, WaitsOutside: []string{}},
				{ID: "b", Stage: 1, Loop: loopOf(0), After: []string{}, Links: map[string]int{}, ItemsInside: 1, WaitsOutside: []string{}},
				{ID: "c", Stage: 1, Loop: loopOf(0), After: []string{}, Links: map[string]int{}, ItemsInside: 1, WaitsOutside: []string{}},
			}},
		{"a link inside one step drops",
			tree([][2]string{{"r", ""}, {"a", "r"}, {"b", "r"}, {"a1", "a"}, {"a2", "a"}}, map[string][]string{"a1": {"a2"}}),
			[]Step{
				{ID: "a", Stage: 0, After: []string{}, Links: map[string]int{}, ItemsInside: 2, WaitsOutside: []string{}},
				{ID: "b", Stage: 0, After: []string{}, Links: map[string]int{}, WaitsOutside: []string{}},
			}},
		{"a link along one branch drops",
			tree([][2]string{{"r", ""}, {"a", "r"}, {"b", "r"}, {"a1", "a"}, {"a11", "a1"}}, map[string][]string{"a11": {"a"}, "b": {"r"}}),
			[]Step{
				{ID: "a", Stage: 0, After: []string{}, Links: map[string]int{}, ItemsInside: 2, WaitsOutside: []string{}},
				{ID: "b", Stage: 0, After: []string{}, Links: map[string]int{}, WaitsOutside: []string{}},
			}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, stepIDs := StepIDs(tc.nodes, "r")
			if got := Steps(tc.nodes, stepIDs); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("\n got %+v\nwant %+v", got, tc.want)
			}
		})
	}
}

// A link from a shown step's work to a node under no shown step waits
// outside; a link from outside into a step is not this level's.
func TestStepsListWhatWaitsOutside(t *testing.T) {
	nodes := tree([][2]string{{"top", ""}, {"r", "top"}, {"other", "top"}, {"a", "r"}, {"b", "r"}, {"a1", "a"}},
		map[string][]string{"a1": {"other", "other2"}, "a": {"other"}, "other": {"b"}})
	nodes = append(nodes, StepNode{ID: "other2", ParentID: "top", Position: 2})
	_, stepIDs := StepIDs(nodes, "r")
	got := Steps(nodes, stepIDs)
	want := []Step{
		{ID: "a", Stage: 0, After: []string{}, Links: map[string]int{}, ItemsInside: 1, WaitsOutside: []string{"other", "other2"}},
		{ID: "b", Stage: 0, After: []string{}, Links: map[string]int{}, WaitsOutside: []string{}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("\n got %+v\nwant %+v", got, want)
	}
}
