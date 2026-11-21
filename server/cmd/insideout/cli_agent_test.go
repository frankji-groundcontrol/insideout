package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// `propose --item Title[@Parent]` keeps its string syntax (AIDEV-17 made the
// MCP and API-client side structured; the CLI still splits at the first "@").
func TestProposeItemFlagSplitsTitleAtParent(t *testing.T) {
	var gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		w.Write([]byte(`{"recorded":true}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("INSIDEOUT_API", srv.URL)
	t.Setenv("INSIDEOUT_TOKEN", "tok")

	var handled bool
	var code int
	captureStdout(t, func() {
		handled, code = runClientCommand([]string{"propose", "--item", "Ship v1@Launch", "--item", "Standalone", "p1", "the summary"})
	})
	if !handled || code != 0 {
		t.Fatalf("propose = (handled=%v, code=%d), want (true, 0)", handled, code)
	}
	if gotPath != "/agent/propose" {
		t.Errorf("path = %q, want /agent/propose", gotPath)
	}
	var body struct {
		Items []struct {
			Action     string `json:"action"`
			Title      string `json:"title"`
			ParentHint string `json:"parentHint"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(gotBody), &body); err != nil {
		t.Fatalf("body %q: %v", gotBody, err)
	}
	if len(body.Items) != 2 ||
		body.Items[0].Title != "Ship v1" || body.Items[0].ParentHint != "Launch" ||
		body.Items[1].Title != "Standalone" || body.Items[1].ParentHint != "" {
		t.Errorf("items = %+v, want {Ship v1 under Launch} and {Standalone}; body %q", body.Items, gotBody)
	}
}
