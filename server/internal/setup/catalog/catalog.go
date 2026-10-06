// Package catalog ports scripts/env_catalog.py: every environment variable
// InsideOut knows about, and each one's current state. The form rules come
// from .env.example's header — an uncommented assignment is REQUIRED, a
// commented one is OPTIONAL with the shown value as the default in effect.
package catalog

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Var is one variable declared by a .env.example skeleton. Example is the
// value shown on the skeleton line (the key a "default" status compares
// against); Default is the value in effect while the variable is unset — the
// skeleton value for optional variables, empty for required ones, whose
// shipped value is a placeholder a person must replace.
type Var struct {
	Name     string
	Comment  string // help text accumulated from the comment block above the key
	Default  string
	Example  string
	Required bool
	Secret   bool
}

// Status mirrors env_catalog.py's five states exactly.
type Status string

const (
	StatusSet         Status = "set"
	StatusDefault     Status = "default"
	StatusPlaceholder Status = "placeholder"
	StatusMissing     Status = "missing"
	StatusUnset       Status = "unset"
)

// State pairs a catalog variable with the live value read from the root .env.
// Value carries raw bytes; anything shown to a user must go through Display.
type State struct {
	Var    Var
	Value  string // "" when not set
	Status Status
}

// A value is masked when its NAME says it is a credential. Name-based because
// the decision must be made BEFORE the value reaches a widget — inspecting it
// to decide would mean it had already been read onto the screen.
var secretRX = regexp.MustCompile(`(?i)(KEY|SECRET|PASSWORD|PASSWD|TOKEN|CREDENTIAL)`)

// Not caught by the regex above: DATABASE_URL embeds a password, and
// INSIDEOUT_LLM_BASE_URL points at a provider (gateway URLs routinely carry a
// key in the path), which repo policy treats as sensitive alongside secrets.
var secretNames = map[string]bool{
	"DATABASE_URL":           true,
	"DATABASE_OWNER_URL":     true,
	"INSIDEOUT_LLM_BASE_URL": true,
}

// Secret mirrors the python name-based rule; display paths consult this
// directly so a stale Var.Secret field can never unmask a credential.
func Secret(name string) bool {
	return secretNames[name] || secretRX.MatchString(name)
}

// Markers meaning "copied from the skeleton and never filled in", matched
// lowercased. Deliberately NOT a generic '<' test — a bare-angle heuristic is
// how Piper's checker started rejecting legitimate values.
var placeholders = []string{"change_me", "your-remote-host"}

func isPlaceholder(value string) bool {
	low := strings.ToLower(value)
	for _, m := range placeholders {
		if strings.Contains(low, m) {
			return true
		}
	}
	return false
}

var (
	keyRX           = regexp.MustCompile(`^(#?)\s*([A-Za-z_][A-Za-z0-9_]*)=(.*)$`)
	nameRX          = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	inlineCommentRX = regexp.MustCompile(`\s+#`)
)

// ParseSkeleton reads a .env.example's bytes, mirroring python
// parse_skeleton's comment-attachment exactly: free-text comment lines
// accumulate and attach to the next key as its help text, a blank line clears
// the buffer, a commented `#KEY=` line is an OPTIONAL variable, and any other
// line is ignored with the buffer preserved. It returns one Var per key line
// in file order, mirroring the python function; callers merging several
// skeletons own the dedup.
func ParseSkeleton(example []byte) ([]Var, error) {
	var (
		out []Var
		buf []string
	)
	for _, line := range strings.Split(string(example), "\n") {
		stripped := strings.TrimSpace(line)
		switch {
		case stripped == "":
			buf = nil
		case keyRX.MatchString(stripped):
			m := keyRX.FindStringSubmatch(stripped)
			hashed, name, example := m[1], m[2], m[3]
			v := Var{
				Name:     name,
				Required: hashed == "",
				Example:  stripInlineComment(strings.TrimSpace(example)),
				Comment:  strings.Join(buf, " "),
			}
			if !v.Required {
				v.Default = v.Example
			}
			v.Secret = Secret(v.Name)
			out = append(out, v)
			buf = nil
		case strings.HasPrefix(stripped, "#"):
			buf = append(buf, strings.TrimRightFunc(strings.TrimLeft(stripped, "# "), unicode.IsSpace))
		}
	}
	return out, nil
}

// stripInlineComment drops a trailing ` # …` comment from an unquoted dotenv
// value, exactly as bash `set -a; . .env`, compose and c12 all do — a catalog
// that kept it would report a spurious mismatch against the skeleton.
// Whitespace before the '#' is required, which leaves `p@ss#word` intact.
func stripInlineComment(value string) string {
	if strings.HasPrefix(value, "'") || strings.HasPrefix(value, `"`) {
		return value
	}
	return strings.TrimRightFunc(inlineCommentRX.Split(value, 2)[0], unicode.IsSpace)
}

// parseEnv mirrors python parse_env_file: KEY=value pairs from dotenv text.
// Comments and non-assignments are skipped, one `export ` prefix is removed,
// paired quotes are stripped from the value, and a bare value loses its
// trailing ` # …` comment. Later assignments win, matching every dotenv
// reader.
func parseEnv(text string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" ||
			strings.HasPrefix(strings.TrimLeftFunc(line, unicode.IsSpace), "#") {
			continue
		}
		name, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		name = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(name), "export "))
		if !nameRX.MatchString(name) {
			continue
		}
		value = strings.TrimSpace(value)
		if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') && value[len(value)-1] == value[0] {
			value = value[1 : len(value)-1]
		} else {
			value = stripInlineComment(value)
		}
		out[name] = value
	}
	return out
}

// Evaluate reads dotenv bytes against catalog variables and reports each one's
// state in input order, with python's exact five statuses:
//
//	placeholder — the live value contains a skeleton marker (case-insensitive)
//	default     — the live value is byte-identical to a non-empty example value
//	set         — any other non-empty value
//	missing     — required and without a value
//	unset       — optional and without a value
//
// Value validity is deliberately NOT decided here; that stays with env.sh
// check, the one authority for "usable" as opposed to "supplied".
func Evaluate(env []byte, vars []Var) []State {
	values := parseEnv(string(env))
	out := make([]State, 0, len(vars))
	for _, v := range vars {
		value := values[v.Name]
		out = append(out, State{Var: v, Value: value, Status: statusOf(v, value)})
	}
	return out
}

func statusOf(v Var, value string) Status {
	switch {
	case isPlaceholder(value):
		return StatusPlaceholder
	case sameAsExample(v, value):
		return StatusDefault
	case value != "":
		return StatusSet
	case v.Required:
		return StatusMissing
	default:
		return StatusUnset
	}
}

// sameAsExample: the value came from a skeleton, not from a person. An empty
// example does not count — an empty value is already unset.
func sameAsExample(v Var, value string) bool {
	if value == "" {
		return false
	}
	return (v.Example != "" && v.Example == value) || (v.Default != "" && v.Default == value)
}

// Mask renders a secret's presence without its bytes, mirroring env-lib.sh's
// mask_secret: "set (N chars)", or "" when there is no value.
func Mask(value string) string {
	if value == "" {
		return ""
	}
	return fmt.Sprintf("set (%d chars)", utf8.RuneCountInString(value))
}

// RedactDSN mirrors env-lib.sh's redact_url: enough of a postgres DSN to
// orient (which host?) and never a credential. The blank anchors on the LAST
// '@' so an '@' inside a password is fully masked. Any shape we cannot
// confidently redact — a libpq keyword/value DSN, or a URL carrying a
// credential query parameter — is masked whole rather than guessed at.
func RedactDSN(value string) string {
	if value == "" {
		return ""
	}
	low := strings.ToLower(value)
	if strings.Contains(low, "password=") || strings.Contains(low, "passfile=") ||
		strings.Contains(low, "sslpassword=") {
		return Mask(value)
	}
	if !strings.HasPrefix(value, "postgres://") && !strings.HasPrefix(value, "postgresql://") {
		return Mask(value)
	}
	at := strings.LastIndex(value, "@")
	if at < 0 {
		return value
	}
	scheme := "postgres://"
	if strings.HasPrefix(value, "postgresql://") {
		scheme = "postgresql://"
	}
	return scheme + "***@" + value[at+1:]
}

// Display renders a state for a list row. It NEVER returns raw secret bytes:
// the name test is absolute, and DSN-shaped values get their userinfo blanked
// (redact_url) whichever name carries them. A content test can never override
// this — a real credential merely containing `change_me` stays masked.
func (s State) Display() string {
	if s.Value == "" {
		return ""
	}
	if strings.HasPrefix(s.Value, "postgres://") || strings.HasPrefix(s.Value, "postgresql://") {
		return RedactDSN(s.Value)
	}
	if Secret(s.Var.Name) {
		return Mask(s.Value)
	}
	return s.Value
}
