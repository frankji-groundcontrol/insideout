package main

import (
	"reflect"
	"strings"
	"testing"
)

// rv5-PARB-05, PARB-12: accepting a proposal applies its items unless
// --no-apply says otherwise, the same default the web carries; `proposal
// decide` is the verb, and `idea proposal-decide` stays as its old spelling.
func TestProposalDecideAppliesByDefault(t *testing.T) {
	requests := treeAPI(t)
	decision := func(d string, apply bool, reason string) []sent {
		return []sent{{"POST /agent/proposals/u1/decision", map[string]any{"decision": d, "apply": apply, "reason": reason}}}
	}
	for _, tc := range []struct {
		args []string
		want []sent
	}{
		{words("proposal", "decide", "--accept", "u1"), decision("accepted", true, "")},
		{words("proposal", "decide", "--accept", "--no-apply", "u1"), decision("accepted", false, "")},
		{words("proposal", "decide", "--reject", "--reason", "not now", "u1"), decision("rejected", false, "not now")},
		{words("idea", "proposal-decide", "--accept", "u1"), decision("accepted", true, "")},
		{words("idea", "proposal-decide", "--accept", "--apply", "u1"), decision("accepted", true, "")},
	} {
		var handled bool
		var code int
		captureStdout(t, func() { handled, code = runClientCommand(tc.args) })
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

// The idea usage line names every sub-verb the switch runs (PARA-11, PARB-12).
func TestIdeaUsageNamesEverySubVerb(t *testing.T) {
	var code int
	stderr := captureStderr(t, func() { _, code = runClientCommand(words("idea")) })
	for _, sub := range []string{"create", "update", "delete", "convert", "proposal-decide"} {
		if !strings.Contains(stderr, sub) {
			t.Errorf("idea usage %q does not name %s", stderr, sub)
		}
	}
	if code != 2 {
		t.Errorf("idea with no sub-verb exit %d, want 2", code)
	}
}
