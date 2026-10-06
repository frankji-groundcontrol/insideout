// Package envwrite ports the .env write path of scripts/env-write.sh's
// apply_changes and env_catalog.py's set_var/clear_var: replace only the
// effective line of a key, keep every other byte, quote values so the file's
// own readers parse them back, and never leave .env observable at a wider
// mode than 0600. Paths are explicit parameters — the INSIDEOUT_ENV_FILE
// override is the caller's concern.
package envwrite

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

// EnsureFromExample creates envPath as a byte copy of examplePath with mode
// 0600 when the env file is missing, mirroring setup.sh and `env.sh init`.
// An existing env file is never touched.
func EnsureFromExample(envPath, examplePath string) error {
	if _, err := os.Stat(envPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	data, err := os.ReadFile(examplePath)
	if err != nil {
		return err
	}
	if err := os.WriteFile(envPath, data, 0o600); err != nil {
		return err
	}
	return os.Chmod(envPath, 0o600)
}

// SetVar writes one KEY into the env file, replacing only the effective line —
// the last uncommented `KEY=` line, else the last commented one — and
// preserving every other byte. An absent key is appended (unfusing a missing
// final newline first, as apply_changes does). The value is quoted so
// `set -a; . .env` and every dotenv parser read it back; the write is atomic.
func SetVar(envPath, name, value string) error {
	data, err := os.ReadFile(envPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		data = nil
	}
	line := name + "=" + quote(value)
	if start, end, found := lastMatch(data, name); found {
		data = replaceLine(data, start, end, line)
	} else {
		if len(data) > 0 && data[len(data)-1] != '\n' {
			data = append(data, '\n')
		}
		data = append(data, line...)
		data = append(data, '\n')
	}
	return writeFile(envPath, data)
}

// ClearVar comments the key out (`#NAME=`), so nothing is set and the line
// keeps its place — python clear_var's exact behavior. It deliberately does
// not restore the skeleton's default text; .env.example remains where that is
// recorded. A missing file or an absent key is a no-op.
func ClearVar(envPath, name string) error {
	data, err := os.ReadFile(envPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	start, end, found := lastMatch(data, name)
	if !found {
		return nil
	}
	return writeFile(envPath, replaceLine(data, start, end, "#"+name+"="))
}

// lastMatch locates the LAST line assigning name, as python _last_match does:
// every reader of a dotenv resolves a duplicated key to the last occurrence —
// bash executes assignments in order, and so do compose and c12 — and
// appending an override at the bottom is ordinary practice, so the last
// uncommented `#?\s*NAME=` line wins; when the key is only present commented,
// the last commented line is the one to edit in place.
func lastMatch(data []byte, name string) (start, end int, found bool) {
	rx := regexp.MustCompile(`^#?\s*` + regexp.QuoteMeta(name) + `=`)
	liveStart, liveEnd := -1, -1
	anyStart, anyEnd := -1, -1
	eachLine(data, func(s, e int, line string) {
		if !rx.MatchString(line) {
			return
		}
		anyStart, anyEnd = s, e
		if !strings.HasPrefix(strings.TrimLeftFunc(line, unicode.IsSpace), "#") {
			liveStart, liveEnd = s, e
		}
	})
	if liveStart >= 0 {
		return liveStart, liveEnd, true
	}
	return anyStart, anyEnd, anyStart >= 0
}

// eachLine walks data's newline-delimited lines, passing each line's byte
// span (excluding the '\n') and its text.
func eachLine(data []byte, fn func(start, end int, line string)) {
	start := 0
	for i := 0; i <= len(data); i++ {
		if i == len(data) || data[i] == '\n' {
			if i > start || (i == len(data) && start < len(data)) {
				fn(start, i, string(data[start:i]))
			}
			start = i + 1
		}
	}
}

// replaceLine splices one line's bytes, leaving every other byte identical.
func replaceLine(data []byte, start, end int, line string) []byte {
	out := make([]byte, 0, len(data)-(end-start)+len(line))
	out = append(out, data[:start]...)
	out = append(out, line...)
	out = append(out, data[end:]...)
	return out
}

// writeFile replaces path atomically: a temp file in the same directory,
// chmod 0600 BEFORE the rename — doing it after leaves a window in which
// .env exists at the umask default — then a rename, so the file is never
// observable with the wrong mode.
func writeFile(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name) // no-op once the rename has succeeded
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(name, 0o600); err != nil {
		return err
	}
	return os.Rename(name, path)
}

const bareSafePattern = `^[A-Za-z0-9_./:@%+,=~^-]*$`

var (
	bareSafeRX   = regexp.MustCompile(bareSafePattern)
	bashActiveRX = regexp.MustCompile("([$`\"\\\\])")
)

// quote mirrors env_catalog.quote / env-lib.sh quote_val: render a value so
// `set -a; . .env` and every dotenv parser read it back. A bare `KEY=two
// words` would make bash treat the second word as a command — and a space is
// legal in a Postgres password. A bare-safe value stays bare and
// diff-friendly; a value containing no single quote is single-quoted
// (interpolation suppressed entirely); otherwise double quotes with the four
// bash-active characters escaped.
func quote(value string) string {
	if bareSafeRX.MatchString(value) {
		return value
	}
	if !strings.Contains(value, "'") {
		return "'" + value + "'"
	}
	return `"` + bashActiveRX.ReplaceAllString(value, `\$1`) + `"`
}
