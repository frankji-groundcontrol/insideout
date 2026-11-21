package ui

// Keystrokes for the setup buttons. Left is Back, right is Forward,
// and Q exits a choice. Q is a letter inside a text field.
type keyKind int

const (
	keyNone keyKind = iota
	keyLeft
	keyRight
	keyEnter
	keyQuit
	keyBackspace
	keyRune
)

func (p *prompts) readArrow() (keyKind, rune, error) {
	b, err := p.in.ReadByte()
	if err != nil || b != '[' {
		return keyNone, 0, err
	}
	b, err = p.in.ReadByte()
	if err != nil {
		return keyNone, 0, err
	}
	switch b {
	case 'D':
		return keyLeft, 0, nil
	case 'C':
		return keyRight, 0, nil
	default:
		return keyNone, 0, nil
	}
}
