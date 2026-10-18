package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// rv4-USER-02 / FULL-03 / PM-04: `insideout --help`, `-h`, `help`, `version`,
// `--version` and a mistyped verb all fell through to config.Load() and died
// with "config: DATABASE_URL is required" — the first thing a new user typed
// told them nothing. These run the real main() in a child process with an
// empty environment (main_exec_test.go), i.e. `env -i insideout ...`.

const configError = "DATABASE_URL is required"

func TestHelpAndVersionNeedNoServerConfig(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"-h"}, {"help"}} {
		res := runCLI(t, nil, "", args...)
		if res.code != 0 {
			t.Errorf("insideout %v: exit %d, want 0\n%s", args, res.code, res.output())
		}
		if !strings.Contains(res.stdout, "Usage") {
			t.Errorf("insideout %v: stdout has no usage text:\n%s", args, res.stdout)
		}
		if strings.Contains(res.output(), configError) {
			t.Errorf("insideout %v demanded server config:\n%s", args, res.output())
		}
	}
	for _, args := range [][]string{{"version"}, {"--version"}} {
		res := runCLI(t, nil, "", args...)
		if res.code != 0 {
			t.Errorf("insideout %v: exit %d, want 0\n%s", args, res.code, res.output())
		}
		// An unstamped build reports "dev"; release.yml stamps the tag.
		if got := strings.TrimSpace(res.stdout); got != "insideout dev" {
			t.Errorf("insideout %v: stdout %q, want %q", args, got, "insideout dev")
		}
		if strings.Contains(res.output(), configError) {
			t.Errorf("insideout %v demanded server config:\n%s", args, res.output())
		}
	}
}

func TestUnknownVerbExitsTwoBeforeServerConfig(t *testing.T) {
	for _, verb := range []string{"bogus", "migrat"} {
		res := runCLI(t, nil, "", verb)
		out := res.output()
		if res.code != 2 {
			t.Errorf("insideout %s: exit %d, want 2\n%s", verb, res.code, out)
		}
		if !strings.Contains(out, verb) || !strings.Contains(out, "insideout help") {
			t.Errorf("insideout %s: the error must name the verb and point at `insideout help`:\n%s", verb, out)
		}
		if strings.Contains(out, configError) {
			t.Errorf("insideout %s demanded server config instead of saying the verb is unknown:\n%s", verb, out)
		}
	}
}

// The container runs `/insideout` with no arguments (server/Dockerfile
// ENTRYPOINT, no CMD): no verb must keep meaning "run the server".
func TestBareInvocationStillStartsTheServer(t *testing.T) {
	res := runCLI(t, nil, "")
	if res.code != 1 || !strings.Contains(res.output(), configError) {
		t.Fatalf("bare insideout must go on to server startup (and fail on missing config here): exit %d\n%s", res.code, res.output())
	}
}

// Front-door routing must not swallow the verbs that do need the database.
func TestServerVerbsStillLoadServerConfig(t *testing.T) {
	for _, verb := range []string{"migrate", "seed", "promote-admin"} {
		res := runCLI(t, nil, "", verb)
		if res.code != 1 || !strings.Contains(res.output(), configError) {
			t.Errorf("insideout %s: exit %d, want the server-config error\n%s", verb, res.code, res.output())
		}
	}
}

// Every verb the dispatch switches accept must be in the help listing —
// review 3 ARCH-04 found hand-written verb lists going stale.
func TestHelpListsEveryDispatchedVerb(t *testing.T) {
	res := runCLI(t, nil, "", "help")
	if res.code != 0 {
		t.Fatalf("insideout help: exit %d\n%s", res.code, res.output())
	}
	for _, verb := range dispatchedVerbs(t, "cli.go:runClientCommand", "main.go:main") {
		if !strings.Contains(res.stdout, "\n  "+verb+" ") {
			t.Errorf("insideout help does not list the verb %q", verb)
		}
	}
}

// dispatchedVerbs returns the first word of every string-literal case in the
// named functions ("file.go:funcName"): the verbs those switches route.
func dispatchedVerbs(t *testing.T, sources ...string) []string {
	t.Helper()
	seen := map[string]bool{}
	for _, src := range sources {
		file, fn, _ := strings.Cut(src, ":")
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		found := false
		for _, decl := range parsed.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Name.Name != fn {
				continue
			}
			found = true
			ast.Inspect(fd, func(n ast.Node) bool {
				if cc, ok := n.(*ast.CaseClause); ok {
					for _, e := range cc.List {
						lit, ok := e.(*ast.BasicLit)
						if !ok || lit.Kind != token.STRING {
							continue
						}
						v, _ := strconv.Unquote(lit.Value)
						if words := strings.Fields(v); len(words) > 0 && !strings.HasPrefix(words[0], "-") {
							seen[words[0]] = true
						}
					}
				}
				return true
			})
		}
		if !found {
			t.Fatalf("%s: no func %s — update the drift test with the code", file, fn)
		}
	}
	verbs := make([]string, 0, len(seen))
	for v := range seen {
		verbs = append(verbs, v)
	}
	sort.Strings(verbs)
	return verbs
}
