package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// TEST-13: every route this package registers must turn away a caller with
// no credentials. The routes are read from the registration calls themselves,
// so a route added without requireAuth fails here the day it lands, with no
// list to remember to update.
//
// publicRoutes answer without a session on purpose: the liveness probe and
// GitHub's webhook (authenticated by its HMAC signature, not a session). All
// of /api/v1/auth/ is public too: sign-in, sign-up and the password and email
// flows. A new public route is a decision; add it here and say why.
var publicRoutes = []string{"GET /healthz", "POST /api/v1/hooks/github"}

const publicPrefix = "/api/v1/auth/"

var pathPlaceholder = regexp.MustCompile(`\{[^}]*\}`)

// registeredRoutes lists the patterns ("METHOD /path") this package hands to a
// ServeMux. A pattern that is not a string literal cannot be read, so it fails
// the test rather than slipping past it.
func registeredRoutes(t *testing.T) []string {
	t.Helper()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	var routes []string
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", f, err)
		}
		ast.Inspect(src, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || (sel.Sel.Name != "HandleFunc" && sel.Sel.Name != "Handle") {
				return true
			}
			lit, ok := call.Args[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				t.Errorf("%s: %s takes a pattern that is not a string literal, so this test cannot check the route", fset.Position(call.Pos()), sel.Sel.Name)
				return true
			}
			pattern, _ := strconv.Unquote(lit.Value)
			routes = append(routes, pattern)
			return true
		})
	}
	return routes
}

func TestEveryRouteRequiresAuthUnlessPublic(t *testing.T) {
	h := newValidationServer().Handler()
	known := map[string]bool{}
	checked := 0
	for _, pattern := range registeredRoutes(t) {
		known[pattern] = true
		method, path, found := strings.Cut(pattern, " ")
		if !found {
			method, path = http.MethodGet, pattern // no method in the pattern: any method matches
		}
		if strings.HasPrefix(path, publicPrefix) || slices.Contains(publicRoutes, pattern) {
			continue
		}
		checked++
		t.Run(pattern, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(method, pathPlaceholder.ReplaceAllString(path, uuid.NewString()), nil))
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("%s with no credentials: status %d, want 401 (was requireAuth dropped from its registration?); body %s", pattern, rec.Code, rec.Body)
			}
		})
	}
	for _, p := range publicRoutes {
		if !known[p] {
			t.Errorf("publicRoutes lists %q but no such route is registered; drop the stale entry", p)
		}
	}
	// A scan that finds almost nothing would pass vacuously.
	if checked < 40 {
		t.Fatalf("scanned only %d routes that need a session; the registration scan is broken or the routes moved out of this package", checked)
	}
}
