package apiclient

import (
	"net/http"
	"testing"
)

// The CRUD completion verbs: one per existing API route that had no
// projection (ideas lifecycle, PRD edits, evidence read, bootstrapping).
// Each test pins method + path + body; reads return the API's raw JSON.

func TestCrudVerbsHitRightRoutes(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		buf := make([]byte, 1024)
		n, _ := r.Body.Read(buf)
		gotBody = string(buf[:n])
		w.Write([]byte(`[{"ok":true}]`))
	})
	c.SetToken("tok")
	cases := []struct {
		name string
		call func() error
		method,
		path, body string
	}{
		{"idea_list", func() error { _, err := c.Ideas("ws1"); return err },
			"GET", "/workspaces/ws1/ideas", ""},
		{"idea_update", func() error { _, err := c.UpdateIdea("i1", "T", "C"); return err },
			"PATCH", "/ideas/i1", `{"content":"C","title":"T"}`},
		{"idea_delete", func() error { return c.DeleteIdea("i1") },
			"DELETE", "/ideas/i1", ""},
		{"prd_update", func() error {
			_, err := c.UpdatePrd("p1", nil, map[string]string{"goals": "G"})
			return err
		}, "PATCH", "/prds/p1", `{"sections":{"goals":"G"}}`},
		{"prd_status", func() error { _, err := c.SetPrdStatus("p1", "archived"); return err },
			"POST", "/prds/p1/status", `{"status":"archived"}`},
		{"roadmap_evidence", func() error { _, err := c.NodeEvidence("n1"); return err },
			"GET", "/roadmap/n1/evidence", ""},
		{"workspace_create", func() error { _, err := c.CreateWorkspace("W", "D"); return err },
			"POST", "/workspaces", `{"description":"D","title":"W"}`},
		{"project_create", func() error { _, err := c.CreateProject("ws1", "P", "D"); return err },
			"POST", "/workspaces/ws1/projects", `{"description":"D","title":"P"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.call(); err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			if gotMethod != tc.method || gotPath != tc.path {
				t.Errorf("got %s %s, want %s %s", gotMethod, gotPath, tc.method, tc.path)
			}
			if gotBody != tc.body {
				t.Errorf("body = %q, want %q", gotBody, tc.body)
			}
		})
	}
}

// UpdatePrd with a title carries it alongside sections.
func TestUpdatePrdCarriesTitle(t *testing.T) {
	var gotBody string
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 1024)
		n, _ := r.Body.Read(buf)
		gotBody = string(buf[:n])
		w.Write([]byte(`{}`))
	})
	c.SetToken("tok")
	title := "New Title"
	if _, err := c.UpdatePrd("p1", &title, map[string]string{"goals": "G"}); err != nil {
		t.Fatalf("UpdatePrd: %v", err)
	}
	if gotBody != `{"sections":{"goals":"G"},"title":"New Title"}` {
		t.Errorf("body = %q", gotBody)
	}
}
