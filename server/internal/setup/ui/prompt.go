package ui

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// ErrBack and ErrExit are how a step leaves the wizard. Back returns to
// the previous question. Exit stops setup.
var (
	ErrBack = errors.New("insideout setup: back")
	ErrExit = errors.New("insideout setup: exit")
)

type move int

const (
	moveOK move = iota
	moveBack
	moveExit
)

// prompts is the setup dialogue. Choices are a numbered list. Back, Exit,
// and Forward are a button row under it: ← back, Q exit, → forward.
// Forward does nothing until a number is selected. Inside a text field,
// Q is a letter; ← and → still go back and forward.
type prompts struct {
	in     *bufio.Reader
	out    io.Writer
	tty    *os.File
	raw    bool
	rows   int
	parked int // lines the cursor sits above the bottom of the painted block
	step   string
}

// at sets the step number drawn above the next question.
func (p *prompts) at(step string) *prompts {
	p.step = step
	return p
}

func (p *prompts) choose(label string, options []string, _ int) (int, move, error) {
	var picked int
	var mv move
	err := p.withRaw(func() error {
		selected := -1
		refused := false
		for {
			screen, line, col := renderChoices(p.step, label, options, selected, refused)
			p.paint(screen)
			p.place(line, col)
			kind, r, err := p.readKey(true)
			if err != nil {
				return err
			}
			refused = false
			switch kind {
			case keyLeft:
				picked, mv = 0, moveBack
				return nil
			case keyQuit:
				picked, mv = 0, moveExit
				return nil
			case keyRight, keyEnter:
				if selected < 0 {
					refused = true
					continue
				}
				picked, mv = selected, moveOK
				return nil
			case keyRune:
				n := int(r - '0')
				if n >= 1 && n <= len(options) {
					selected = n - 1
				}
			}
		}
	})
	return picked, mv, err
}

func (p *prompts) line(label, def string) (string, move, error) {
	return p.edit(label, def, false)
}

func (p *prompts) confirm(label string, defYes bool) (bool, move, error) {
	def := 1
	if defYes {
		def = 0
	}
	n, mv, err := p.choose(label, []string{"Yes", "No"}, def)
	if err != nil || mv != moveOK {
		return false, mv, err
	}
	return n == 0, moveOK, nil
}

func (p *prompts) password(label string) (string, move, error) {
	return p.edit(label, "", true)
}

func (p *prompts) edit(label, initial string, hidden bool) (string, move, error) {
	var out string
	var mv move
	err := p.withRaw(func() error {
		buf := []rune(initial)
		refused := false
		for {
			screen, line, col := renderField(p.step, label, buf, hidden, refused)
			p.paint(screen)
			p.place(line, col)
			kind, r, err := p.readKey(false)
			if err != nil {
				return err
			}
			refused = false
			switch kind {
			case keyQuit:
				mv = moveExit
				return nil
			case keyLeft:
				mv = moveBack
				return nil
			case keyRight, keyEnter:
				if len(buf) == 0 {
					refused = true
					continue
				}
				out, mv = string(buf), moveOK
				return nil
			case keyBackspace:
				if len(buf) > 0 {
					buf = buf[:len(buf)-1]
				}
			case keyRune:
				buf = append(buf, r)
			}
		}
	})
	return out, mv, err
}

func renderChoices(step, label string, options []string, selected int, refused bool) (string, int, int) {
	lines := []string{"", stepPrefix(step) + label, ""}
	for i, opt := range options {
		mark := " "
		if i == selected {
			mark = ">"
		}
		lines = append(lines, fmt.Sprintf(" %s %d) %s", mark, i+1, opt))
	}
	typed := ""
	if selected >= 0 {
		typed = strconv.Itoa(selected + 1)
	}
	lines = append(lines, "", "  > "+typed)
	inputLine := len(lines) - 1
	lines = append(lines, "", strings.TrimRight(buttonRow(selected >= 0, "Q"), "\n"))
	if refused {
		lines = append(lines, "Select an option, then → Forward.")
	}
	return strings.Join(lines, "\n") + "\n", inputLine, 5 + len(typed)
}

func renderField(step, label string, buf []rune, hidden, refused bool) (string, int, int) {
	shown := string(buf)
	if hidden {
		shown = strings.Repeat("•", len(buf))
	}
	lines := []string{"", stepPrefix(step) + label, "", "  > " + shown}
	inputLine := len(lines) - 1
	lines = append(lines, "", strings.TrimRight(buttonRow(len(buf) > 0, "Ctrl-C"), "\n"))
	if refused {
		lines = append(lines, "Enter a value, then → Forward.")
	}
	return strings.Join(lines, "\n") + "\n", inputLine, 5 + len(shown)
}

// stepPrefix puts the step on the question line: "1.1.2 · Email".
// The number is bold; the separator is dim. An empty step adds nothing.
func stepPrefix(step string) string {
	if step == "" {
		return ""
	}
	return "\033[1m" + step + "\033[2m · \033[0m"
}

func buttonRow(canForward bool, quit string) string {
	fwd := "[ → Forward ]"
	if !canForward {
		fwd = "\033[2m[ → Forward ]\033[0m"
	}
	return "  [ ← Back ]    [ " + quit + " Exit ]    " + fwd + "\n"
}

func (p *prompts) paint(screen string) {
	if p.parked > 0 {
		fmt.Fprintf(p.out, "\033[%dB", p.parked)
		p.parked = 0
	}
	if p.rows > 0 {
		fmt.Fprintf(p.out, "\033[%dA", p.rows)
	}
	n := 0
	for _, line := range strings.Split(strings.TrimRight(screen, "\n"), "\n") {
		p.write("\033[2K" + line + "\n")
		n++
	}
	p.rows = n
}

// place leaves the cursor on the input line, after the text already there.
// inputLine is 0-based inside the painted block. col is 1-based.
func (p *prompts) place(inputLine, col int) {
	if inputLine < 0 || inputLine >= p.rows {
		return
	}
	up := p.rows - inputLine
	if up > 0 {
		fmt.Fprintf(p.out, "\033[%dA", up)
	}
	if col < 1 {
		col = 1
	}
	fmt.Fprintf(p.out, "\033[%dG", col)
	p.parked = up
}

func (p *prompts) write(s string) {
	if p.raw {
		s = strings.ReplaceAll(s, "\n", "\r\n")
	}
	fmt.Fprint(p.out, s)
}

func (p *prompts) withRaw(fn func() error) error {
	p.rows = 0
	p.parked = 0
	if p.tty == nil || !term.IsTerminal(int(p.tty.Fd())) {
		return fn()
	}
	old, err := term.MakeRaw(int(p.tty.Fd()))
	if err != nil {
		return fn()
	}
	p.raw = true
	defer func() {
		if p.parked > 0 {
			fmt.Fprintf(p.out, "\033[%dB\r\n", p.parked)
			p.parked = 0
		}
		p.raw = false
		_ = term.Restore(int(p.tty.Fd()), old)
	}()
	return fn()
}

func (p *prompts) readKey(quit bool) (keyKind, rune, error) {
	b, err := p.in.ReadByte()
	if err != nil {
		return keyNone, 0, err
	}
	switch b {
	case 3, 0x11: // Ctrl-C. Ctrl-Q is also accepted, but iTerm over SSH swallows it.
		return keyQuit, 0, nil
	case '\r', '\n':
		return keyEnter, 0, nil
	case 127, 8:
		return keyBackspace, 0, nil
	case 0x1b:
		return p.readArrow()
	}
	if quit && (b == 'q' || b == 'Q') {
		return keyQuit, rune(b), nil
	}
	if b >= 32 && b < 127 {
		return keyRune, rune(b), nil
	}
	return keyNone, 0, nil
}

func moveErr(mv move) error {
	switch mv {
	case moveBack:
		return ErrBack
	case moveExit:
		return ErrExit
	default:
		return nil
	}
}
