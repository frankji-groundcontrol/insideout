package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// ARCH-04: the response layer reads time through the server's injected clock
// (Server.now), never from the wall clock, so a test can pin the date. L1 was
// the failure mode: projectResponse called time.Now() while its test pinned a
// date, and the test went red the day the real calendar passed the 14-day
// threshold.
//
// The scan parses every non-test file of this package and flags time.Now,
// time.Since and time.Until, whether called or handed over as a value. A new
// view builder is therefore checked by default. A file that must read the wall
// clock goes in wallClockReaders with the reason; an entry that no longer
// reads the clock is harmless, delete it when you next touch the file.
var wallClockReaders = map[string]string{
	"admin.go":            "stamps an invitation expiry into the database",
	"auth.go":             "stamps session and refresh-token expiry into the database",
	"auth_email_flows.go": "stamps one-time token expiry into the database",
	"auth_cooldown.go":    "the cooldown's default clock; its tests inject their own",
	"auth_throttle.go":    "the throttle's default clock; its tests inject their own",
	"middleware.go":       "times a request for the access log",
	"server.go":           "the production defaults: Server.now falls back to the wall clock and the throttle is built on it",
}

var wallClockFuncs = map[string]bool{"Now": true, "Since": true, "Until": true}

func TestViewBuildersDoNotReadTheWallClock(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") || wallClockReaders[f] != "" {
			continue
		}
		src, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", f, err)
		}
		pkg := timeImportName(src)
		if pkg == "" {
			continue
		}
		ast.Inspect(src, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok || !wallClockFuncs[sel.Sel.Name] {
				return true
			}
			if id, ok := sel.X.(*ast.Ident); ok && id.Name == pkg {
				t.Errorf("%s: %s.%s reads the wall clock; use s.now() or take now as a parameter (or list the file in wallClockReaders with a reason)",
					fset.Position(sel.Pos()), pkg, sel.Sel.Name)
			}
			return true
		})
	}
}

// timeImportName is the name the file gives the "time" package, "" when it
// does not import it.
func timeImportName(f *ast.File) string {
	for _, imp := range f.Imports {
		if imp.Path.Value != `"time"` {
			continue
		}
		if imp.Name != nil {
			return imp.Name.Name
		}
		return "time"
	}
	return ""
}
