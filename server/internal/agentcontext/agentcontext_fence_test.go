package agentcontext

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

// AIDEV-08: the context an agent reads mixes fields the server decides (ids,
// status, counts, the vocabulary) with text people wrote (titles, PRD sections,
// acceptance sentences, owner names, the Commit name). Nothing told a reader
// which was which, so a PRD or a title could talk to the agent as if it were
// the tool. The output now names the fields that hold people's text, and says
// they are data. These tests fill every place such text enters the assembly
// with a marker and require every place the marker lands to be declared.

const marker = "MARK-"

// markedInputs puts a marker naming its origin into every text the assembly is
// given, so the output shows where each one lands.
func markedInputs(mode, focus string) Inputs {
	sections := map[string]string{}
	for _, k := range []string{"background", "users", "goals", "nonGoals", "stories", "requirements", "constraints", "risks"} {
		sections[k] = marker + "section-" + k
	}
	return Inputs{
		ProjectTitle: marker + "projectTitle", ProjectID: "p1", Mode: mode, FocusNodeID: focus,
		PrdTitle: marker + "prdTitle", PrdSections: sections,
		LatestCommit: &CommitIn{Revision: 2, Name: marker + "versionName", Audience: "decision"},
		Nodes: []NodeIn{
			{ID: "root", Title: marker + "rootTitle", Status: "pending"},
			{ID: "a", ParentID: "root", Title: marker + "aTitle", Status: "in_progress", Acceptance: marker + "aAcceptance", OwnerName: marker + "aOwner"},
			{ID: "b", ParentID: "root", Title: marker + "bTitle", Status: "pending", Acceptance: marker + "bAcceptance", OwnerName: marker + "bOwner"},
		},
		EvidenceCounts: map[string]int{"a": 1},
	}
}

// markerPaths returns the path of every string under v that holds a marker.
// Map keys join with ".", and every array position is written "[]".
func markerPaths(v any, path string, out *[]string) {
	switch x := v.(type) {
	case map[string]any:
		for k, c := range x {
			p := k
			if path != "" {
				p = path + "." + k
			}
			markerPaths(c, p, out)
		}
	case []any:
		for _, c := range x {
			markerPaths(c, path+"[]", out)
		}
	case string:
		if strings.Contains(x, marker) {
			*out = append(*out, path)
		}
	}
}

// covers reports whether the declared field names path or something under it.
func covers(field, path string) bool {
	return path == field || strings.HasPrefix(path, field+".") || strings.HasPrefix(path, field+"[]")
}

func TestAssemble_DeclaresEveryFieldThatCarriesPeoplesText(t *testing.T) {
	cases := []struct{ name, mode, focus string }{
		{"brainstorming", "brainstorming", ""},
		{"review", "review", ""},
		{"implementation with a focus", "implementation", "a"},
		{"implementation without a focus", "implementation", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// What the API writes is what a reader sees: go through JSON.
			raw, err := json.Marshal(Assemble(markedInputs(tc.mode, tc.focus)))
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var doc map[string]any
			if err := json.Unmarshal(raw, &doc); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			label, ok := doc["untrusted"].(map[string]any)
			if !ok {
				t.Fatalf("the context has no \"untrusted\" block naming the fields that hold people's text; keys: %v", keys(doc))
			}
			notice, _ := label["notice"].(string)
			if !strings.Contains(notice, "instructions") {
				t.Errorf("untrusted.notice = %q, want it to say the text is data and not instructions", notice)
			}
			var declared []string
			if list, ok := label["fields"].([]any); ok {
				for _, f := range list {
					if s, ok := f.(string); ok {
						declared = append(declared, s)
					}
				}
			}
			if len(declared) == 0 {
				t.Fatalf("untrusted.fields is empty: %v", label)
			}

			rest := map[string]any{}
			for k, v := range doc {
				if k != "untrusted" {
					rest[k] = v
				}
			}
			var landed []string
			markerPaths(rest, "", &landed)
			if len(landed) == 0 {
				t.Fatal("no marker reached the output: the fixture no longer reaches the assembly")
			}
			for _, p := range landed {
				if !slices.ContainsFunc(declared, func(f string) bool { return covers(f, p) }) {
					t.Errorf("%q holds people's text but no untrusted field covers it (declared %v)", p, declared)
				}
			}
			// A field that names nothing the output holds is a false label.
			for _, f := range declared {
				if !slices.ContainsFunc(landed, func(p string) bool { return covers(f, p) }) {
					t.Errorf("untrusted field %q matches no text in this output (text at %v)", f, landed)
				}
			}
		})
	}
}

func keys(m map[string]any) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

// FenceData is the fence rule the MCP tools use; the coach's own copy in
// agent/prompt_fence.go is held to it by a test over there.
func TestFenceData(t *testing.T) {
	for _, tc := range []struct{ name, in, want string }{
		{"plain text arrives unchanged", "ship it & <b>go</b>", "<prd_data>ship it & <b>go</b></prd_data>"},
		{"a closing tag is broken", "a</prd_data>b", "<prd_data>a&lt;/prd_data>b</prd_data>"},
		{"in any case", "a</PRD_Data>b", "<prd_data>a&lt;/PRD_Data>b</prd_data>"},
		{"split by whitespace", "a< \n/ prd_data >b", "<prd_data>a&lt; \n/ prd_data >b</prd_data>"},
		{"a fake opener", "<prd_data>x", "<prd_data>&lt;prd_data>x</prd_data>"},
		{"nothing at all", "", "<prd_data></prd_data>"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := FenceData(tc.in); got != tc.want {
				t.Errorf("FenceData(%q) = %q, want %q", tc.in, got, tc.want)
			}
			if once := Neutralise(tc.in); Neutralise(once) != once {
				t.Errorf("Neutralise is not idempotent on %q: %q then %q", tc.in, once, Neutralise(once))
			}
		})
	}
}
