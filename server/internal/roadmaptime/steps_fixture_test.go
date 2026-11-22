package roadmaptime

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

// testdata/steps_cases.json holds step cases with their expected order. The
// web's roadmap_step_flow.dart reads the same file
// (client/test/features/project/roadmap_steps_fixture_test.dart), so the
// Canvas and GET /roadmap/steps cannot drift apart. Regenerate the expected
// steps with STEPS_FIXTURE_UPDATE=1 after a deliberate rule change.

type stepCase struct {
	Name   string         `json:"name"`
	Root   string         `json:"root"`
	Nodes  []fixtureNode  `json:"nodes"`
	Expect []expectedStep `json:"expect"`
}

type fixtureNode struct {
	ID        string   `json:"id"`
	ParentID  string   `json:"parentId"`
	Position  int      `json:"position"`
	DependsOn []string `json:"dependsOn"`
}

// expectedStep: Loop is the ids of the step's loop group (empty: none).
type expectedStep struct {
	ID    string   `json:"id"`
	Stage int      `json:"stage"`
	After []string `json:"after"`
	Loop  []string `json:"loop"`
}

func TestStepsMatchTheSharedFixture(t *testing.T) {
	const path = "testdata/steps_cases.json"
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var file struct {
		Cases []stepCase `json:"cases"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatal(err)
	}
	for i, c := range file.Cases {
		got := expectedOf(c)
		if os.Getenv("STEPS_FIXTURE_UPDATE") == "1" {
			file.Cases[i].Expect = got
			continue
		}
		if !reflect.DeepEqual(got, c.Expect) {
			t.Errorf("%s:\n got  %+v\n want %+v", c.Name, got, c.Expect)
		}
	}
	if os.Getenv("STEPS_FIXTURE_UPDATE") == "1" {
		out, _ := json.MarshalIndent(file, "", " ")
		if err := os.WriteFile(path, append(out, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func expectedOf(c stepCase) []expectedStep {
	nodes := make([]StepNode, len(c.Nodes))
	for i, n := range c.Nodes {
		nodes[i] = StepNode{ID: n.ID, ParentID: n.ParentID, Position: n.Position, DependsOn: n.DependsOn}
	}
	_, ids := StepIDs(nodes, c.Root)
	steps := Steps(nodes, ids)
	groups := map[int][]string{}
	for _, s := range steps {
		if s.Loop != nil {
			groups[*s.Loop] = append(groups[*s.Loop], s.ID)
		}
	}
	out := make([]expectedStep, len(steps))
	for i, s := range steps {
		e := expectedStep{ID: s.ID, Stage: s.Stage, After: append([]string{}, s.After...), Loop: []string{}}
		if s.Loop != nil {
			e.Loop = groups[*s.Loop]
		}
		out[i] = e
	}
	return out
}
