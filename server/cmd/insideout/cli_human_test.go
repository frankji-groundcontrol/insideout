package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

// rv5-PARA-03, PARC-04: the human-only verbs MCP deliberately lacks exist on
// the CLI: account deletion (the password through the hidden prompt, never a
// flag), a validation observation and the validate confirm.
func TestHumanOnlyVerbsSendTheirRequests(t *testing.T) {
	requests := treeAPI(t)
	today := time.Now().Format("2006-01-02")
	for _, tc := range []struct {
		args  []string
		stdin string
		want  []sent
	}{
		{words("account", "delete"), "current-pw\n", []sent{{"DELETE /me", map[string]any{"password": "current-pw"}}}},
		{words("evidence", "add", "--detail", "Five users asked", "--source", "https://notes.example/1", "--observed", "2026-11-20", "n1"), "",
			[]sent{{"POST /roadmap/n1/evidence", map[string]any{"detail": "Five users asked", "sourceUrl": "https://notes.example/1", "observedAt": "2026-11-20"}}}},
		{words("evidence", "add", "--detail", "Seen today", "n1"), "",
			[]sent{{"POST /roadmap/n1/evidence", map[string]any{"detail": "Seen today", "sourceUrl": "", "observedAt": today}}}},
		{words("roadmap", "validate", "--reason", "Users kept it", "n1"), "",
			[]sent{{"POST /roadmap/n1/validate", map[string]any{"reason": "Users kept it"}}}},
	} {
		restore := swapStdin(tc.stdin)
		var handled bool
		var code int
		captureStdout(t, func() { handled, code = runClientCommand(tc.args) })
		restore()
		got := requests()
		if !handled || code != 0 {
			t.Errorf("%v = (handled=%v, code=%d), want (true, 0)", tc.args, handled, code)
			continue
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%v sent %v, want %v", tc.args, got, tc.want)
		}
	}
}

// No password, no detail, no reason, a bad date or a password flag: a usage
// error that sends nothing.
func TestHumanOnlyVerbsRefuseMissingArguments(t *testing.T) {
	requests := treeAPI(t)
	for _, tc := range []struct {
		args  []string
		stdin string
	}{
		{words("account", "delete"), "\n"},
		{words("account", "delete", "--password", "pw"), "pw\n"},
		{words("account"), ""},
		{words("evidence", "add", "n1"), ""},
		{words("evidence", "add", "--detail", "D", "--observed", "20 Nov", "n1"), ""},
		{words("roadmap", "validate", "n1"), ""},
		{words("export"), ""},
		{words("export", "--format", "pdf", "p1"), ""},
	} {
		restore := swapStdin(tc.stdin)
		var handled bool
		var code int
		captureStdout(t, func() { handled, code = runClientCommand(tc.args) })
		restore()
		if !handled || code != 2 {
			t.Errorf("%v = (handled=%v, code=%d), want a usage error (true, 2)", tc.args, handled, code)
		}
		if got := requests(); len(got) != 0 {
			t.Errorf("%v sent %v on a usage error", tc.args, got)
		}
	}
}

// rv5-PARB-06: `export` fetches the whole PRD (markdown by default, print HTML
// with --format print, one audience's markdown with --audience) and writes it
// to stdout or --out.
func TestExportFetchesTheWholePRD(t *testing.T) {
	requests := treeAPI(t)
	out := filepath.Join(t.TempDir(), "prd.html")
	for _, tc := range []struct {
		args []string
		want string
	}{
		{words("export", "p1"), "GET /prds/p1/export?format=markdown"},
		{words("export", "--audience", "delivery", "p1"), "GET /prds/p1/export?audience=delivery&format=markdown"},
		{words("export", "--format", "print", "--out", out, "p1"), "GET /prds/p1/export?format=print"},
	} {
		var handled bool
		var code int
		printed := captureStdout(t, func() { handled, code = runClientCommand(tc.args) })
		got := requests()
		if !handled || code != 0 || len(got) != 1 || got[0].target != tc.want {
			t.Errorf("%v = (handled=%v, code=%d) sent %v, want (true, 0) and %s", tc.args, handled, code, got, tc.want)
		}
		if tc.args[len(tc.args)-2] != out && printed != `{"ok":true}` {
			t.Errorf("%v printed %q, want the export as the server sent it", tc.args, printed)
		}
	}
	if b, err := os.ReadFile(out); err != nil || string(b) != `{"ok":true}` {
		t.Errorf("export --out wrote %q (%v), want the export as the server sent it", b, err)
	}
}
