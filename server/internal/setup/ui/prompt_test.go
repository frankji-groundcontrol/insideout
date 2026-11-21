package ui

import (
	"bufio"
	"strings"
	"testing"
)

const (
	keyRightSeq = "\x1b[C"
	keyLeftSeq  = "\x1b[D"
)

func newTestPrompts(in string) (*prompts, *strings.Builder) {
	var out strings.Builder
	return &prompts{in: bufio.NewReader(strings.NewReader(in)), out: &out}, &out
}

func TestStepNumberIsShownAboveTheQuestion(t *testing.T) {
	p, out := newTestPrompts("1" + keyRightSeq)
	_, _, err := p.at("1.1.2").choose("Do you already have an account?", []string{"Log in", "Register"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	text := out.String()
	if !strings.Contains(text, "1.1.2") || !strings.Contains(text, "·") || !strings.Contains(text, "Do you already have an account?") {
		t.Fatalf("output missing the step on the question line:\n%s", text)
	}
	if strings.Contains(text, "1.1.2\nDo") || strings.Contains(text, "1.1.2\r\nDo") {
		t.Fatalf("the step number is still on its own line:\n%s", text)
	}
}

func TestHideDefaultAddressDropsTheHost(t *testing.T) {
	base := "https://service.example/api/v1"
	if got := hideDefaultAddress("https://service.example", base); got != "default" {
		t.Fatalf("origin shown as %q", got)
	}
	if got := hideDefaultAddress("could not reach /healthz", base); got != "could not reach /healthz" {
		t.Fatalf("unrelated detail changed to %q", got)
	}
}

func TestChooseSelectsThenForwards(t *testing.T) {
	p, out := newTestPrompts(keyRightSeq + "2" + keyRightSeq)
	got, mv, err := p.choose("What are you setting up?", []string{"Connect", "This machine"}, 0)
	if err != nil || mv != moveOK || got != 1 {
		t.Fatalf("choose = (%d, %v, %v), want 1", got, mv, err)
	}
	text := out.String()
	for _, want := range []string{"[ ← Back ]", "[ Q Exit ]", "[ → Forward ]", "Select an option, then → Forward."} {
		if !strings.Contains(text, want) {
			t.Fatalf("output missing %q\n%s", want, text)
		}
	}
	if strings.Contains(text, "3) Back") || strings.Contains(text, "4) Exit") {
		t.Fatalf("Back and Exit are still menu rows:\n%s", text)
	}
}

func TestChooseLeftGoesBackAndQExits(t *testing.T) {
	p, _ := newTestPrompts(keyLeftSeq)
	_, mv, err := p.choose("How do you want to sign in?", []string{"Password", "Code"}, 0)
	if err != nil || mv != moveBack {
		t.Fatalf("left = %v, %v, want back", mv, err)
	}
	p, _ = newTestPrompts("q")
	_, mv, err = p.choose("How do you want to sign in?", []string{"Password", "Code"}, 0)
	if err != nil || mv != moveExit {
		t.Fatalf("q = %v, %v, want exit", mv, err)
	}
}

func TestLineForwardsTheDefaultAndTheTypedValue(t *testing.T) {
	p, _ := newTestPrompts(keyRightSeq)
	got, mv, err := p.line("API base URL", "https://hosted.example/api/v1")
	if err != nil || mv != moveOK || got != "https://hosted.example/api/v1" {
		t.Fatalf("forward default = %q, %v, %v", got, mv, err)
	}
	p, _ = newTestPrompts("ab" + keyRightSeq)
	got, mv, err = p.line("Email", "")
	if err != nil || mv != moveOK || got != "ab" {
		t.Fatalf("typed = %q, %v, %v", got, mv, err)
	}
}

func TestLineLeftAndALetterQ(t *testing.T) {
	p, _ := newTestPrompts(keyLeftSeq)
	_, mv, err := p.line("Email", "")
	if err != nil || mv != moveBack {
		t.Fatalf("left = %v, %v, want back", mv, err)
	}
	p, _ = newTestPrompts("q@x.test" + keyRightSeq)
	got, mv, err := p.line("Email", "")
	if err != nil || mv != moveOK || got != "q@x.test" {
		t.Fatalf("q in the address = %q, %v, %v", got, mv, err)
	}
}

func TestCtrlCExitsATextField(t *testing.T) {
	p, _ := newTestPrompts("\x03")
	_, mv, err := p.line("Reset code or link", "")
	if err != nil || mv != moveExit {
		t.Fatalf("ctrl-c = %v, %v, want exit", mv, err)
	}
}

func TestEmptyFieldDoesNotMoveForward(t *testing.T) {
	p, out := newTestPrompts(keyRightSeq + "a" + keyRightSeq)
	got, mv, err := p.line("Email", "")
	if err != nil || mv != moveOK || got != "a" {
		t.Fatalf("line = %q, %v, %v", got, mv, err)
	}
	if !strings.Contains(out.String(), "Enter a value, then → Forward.") {
		t.Fatalf("output missing the forward hint:\n%s", out.String())
	}
}

func TestConfirmNeedsASelection(t *testing.T) {
	p, _ := newTestPrompts("1" + keyRightSeq)
	yes, mv, err := p.confirm("Register MCP?", true)
	if err != nil || mv != moveOK || !yes {
		t.Fatalf("yes = %v, %v, %v", yes, mv, err)
	}
	p, _ = newTestPrompts("2" + keyRightSeq)
	yes, mv, err = p.confirm("Register MCP?", true)
	if err != nil || mv != moveOK || yes {
		t.Fatalf("no = %v, %v, %v", yes, mv, err)
	}
}
