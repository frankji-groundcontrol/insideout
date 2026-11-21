package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// AIDEV-10: `idea delete` told the user a dropped idea "keeps" showing in the
// list with status "dropped"; the list has hidden dropped ideas since USER-09.
func TestIdeaDeleteSaysDroppedIdeasLeaveTheList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("INSIDEOUT_API", srv.URL)
	t.Setenv("INSIDEOUT_TOKEN", "tok")

	var handled bool
	var code int
	out := captureStdout(t, func() { handled, code = runClientCommand([]string{"idea", "delete", "i1"}) })
	if !handled || code != 0 {
		t.Fatalf("idea delete = (handled=%v, code=%d), want (true, 0)", handled, code)
	}
	if !strings.HasPrefix(out, "dropped") || strings.Contains(out, "keeps it") {
		t.Errorf("idea delete printed %q, want a dropped notice that does not claim the idea stays listed", out)
	}
}

// PARA-02: `idea update --title X` must not send a content key, or the
// server would overwrite the idea body with "".
func TestIdeaUpdateWithoutContentSendsNoContentKey(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("INSIDEOUT_API", srv.URL)
	t.Setenv("INSIDEOUT_TOKEN", "tok")

	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"idea", "update", "--title", "X", "i1"}, `{"title":"X"}`},
		// One rule (rv5-STRS-06): an empty value leaves the body, the word clear clears it.
		{[]string{"idea", "update", "--title", "X", "--content", "", "i1"}, `{"title":"X"}`},
		{[]string{"idea", "update", "--title", "X", "--content", "clear", "i1"}, `{"content":"","title":"X"}`},
	} {
		var handled bool
		var code int
		captureStdout(t, func() { handled, code = runClientCommand(tc.args) })
		if !handled || code != 0 {
			t.Fatalf("%v = (handled=%v, code=%d), want (true, 0)", tc.args, handled, code)
		}
		if gotBody != tc.want {
			t.Errorf("%v sent %q, want %q", tc.args, gotBody, tc.want)
		}
	}
}
