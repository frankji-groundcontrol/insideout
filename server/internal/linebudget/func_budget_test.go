package linebudget

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"
	"testing"
)

// longFunctions lists the functions allowed over functionLines, keyed
// "path:Func" (methods as "path:Type.Method") with their current length
// in lines, signature to closing brace. Every entry is tracked in
// docs/issues/2026-11-21-long-functions.md. A listed function may not
// grow, and an entry for a function that is gone or back under the limit
// fails too, so the list only shrinks.
var longFunctions = map[string]int{
	"internal/api/agent_routes.go:Server.handleAgentContext":      89,
	"internal/api/presence_routes.go:Server.handlePresenceStream": 85,
	"internal/api/roadmap.go:Server.handleUpdateRoadmapNode":      97,
}

// funcName is "Func" for a function and "Type.Method" for a method.
func funcName(fd *ast.FuncDecl) string {
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return fd.Name.Name
	}
	typ := fd.Recv.List[0].Type
	for {
		switch e := typ.(type) {
		case *ast.StarExpr:
			typ = e.X
			continue
		case *ast.IndexExpr:
			typ = e.X
			continue
		case *ast.IndexListExpr:
			typ = e.X
			continue
		case *ast.Ident:
			return e.Name + "." + fd.Name.Name
		}
		return fd.Name.Name
	}
}

// A function past the shared threshold is doing more than one step of
// work; it fails here until it is split or listed above.
func TestServerFunctionsStayWithinTheFunctionBudget(t *testing.T) {
	limit := loadBudget(t).FunctionLines
	fset := token.NewFileSet()
	seen := map[string]int{}
	var paste []string
	walkSource(t, func(rel, abs string) {
		f, err := parser.ParseFile(fset, abs, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", rel, err)
		}
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			key := rel + ":" + funcName(fd)
			lines := fset.Position(fd.End()).Line - fset.Position(fd.Pos()).Line + 1
			seen[key] = lines
			allowed, listed := longFunctions[key]
			switch {
			case lines > limit && !listed:
				t.Errorf("%s: %d lines, over the %d-line function budget — split it or list it against docs/issues/2026-11-21-long-functions.md", key, lines, limit)
				paste = append(paste, fmt.Sprintf("%q: %d,", key, lines))
			case listed && lines > allowed:
				t.Errorf("%s: %d lines, grew past its listed %d — split it instead", key, lines, allowed)
				paste = append(paste, fmt.Sprintf("%q: %d,", key, lines))
			}
		}
	})
	for key := range longFunctions {
		if lines, ok := seen[key]; !ok || lines <= limit {
			t.Errorf("%s: listed in longFunctions but is gone or back within %d lines — delete the entry", key, limit)
		}
	}
	if len(paste) > 0 {
		sort.Strings(paste)
		t.Logf("longFunctions entries:\n%s", strings.Join(paste, "\n"))
	}
}
