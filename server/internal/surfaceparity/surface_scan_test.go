package surfaceparity

import (
	"go/ast"
	"go/importer"
	"go/types"
	"testing"
)

// The CLI and MCP half: both reach the API only through apiclient, so a
// surface reaches a key when it calls a client method that sends it and fills
// the argument the key hangs on (a non-zero argument, or the struct field set).

type surface struct {
	name  string
	calls map[string][][]ast.Expr    // "Type.Method" of an apiclient method to each call's arguments
	sets  map[string]map[string]bool // apiclient struct type to the fields this surface sets
}

func loadSurface(t *testing.T, name, rel string) surface {
	s := scanSurface(t, name, parseDir(t, rel))
	if len(s.calls) < 20 {
		t.Fatalf("%s: found only %d client methods called in %s; the scan is broken", name, len(s.calls), rel)
	}
	return s
}

// srcImporter type-checks imports from source (no golang.org/x/tools in go.mod),
// shared so each package is checked once.
var srcImporter = importer.ForCompiler(fset, "source", nil)

// clientCall names the apiclient method a call runs ("Client.RoadmapMove"),
// resolved through the receiver's type, or "" for any other call: a
// same-named method on another type is not a client call.
func clientCall(info *types.Info, call *ast.CallExpr) string {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || info.Selections[sel] == nil {
		return ""
	}
	fn, ok := info.Selections[sel].Obj().(*types.Func)
	if !ok || fn.Pkg() == nil || fn.Pkg().Path() != clientType.Elem().PkgPath() {
		return ""
	}
	recv := fn.Signature().Recv().Type()
	if p, ok := recv.(*types.Pointer); ok {
		recv = p.Elem()
	}
	if n, ok := recv.(*types.Named); ok {
		return n.Obj().Name() + "." + fn.Name()
	}
	return ""
}

// scanSurface collects the client calls and struct field sets in one package's files.
func scanSurface(t *testing.T, name string, files []*ast.File) surface {
	t.Helper()
	info := &types.Info{Selections: map[*ast.SelectorExpr]*types.Selection{}}
	if _, err := (&types.Config{Importer: srcImporter}).Check(name, fset, files, info); err != nil {
		t.Fatalf("type-check %s: %v", name, err)
	}
	s := surface{name: name, calls: map[string][][]ast.Expr{}, sets: map[string]map[string]bool{}}
	for _, f := range files {
		mentioned := map[string]bool{} // apiclient types this file names
		ast.Inspect(f, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok {
				if id, ok := sel.X.(*ast.Ident); ok && id.Name == "apiclient" {
					mentioned[sel.Sel.Name] = true
				}
			}
			return true
		})
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.CallExpr:
				if m := clientCall(info, n); m != "" {
					s.calls[m] = append(s.calls[m], n.Args)
				}
			case *ast.CompositeLit:
				if sel, ok := n.Type.(*ast.SelectorExpr); ok {
					if id, ok := sel.X.(*ast.Ident); ok && id.Name == "apiclient" {
						for _, el := range n.Elts {
							if kv, ok := el.(*ast.KeyValueExpr); ok {
								s.set(sel.Sel.Name, kv.Key.(*ast.Ident).Name)
							}
						}
					}
				}
			case *ast.AssignStmt:
				// x.Field = v, in a file that names the type: a field set after the literal.
				for _, l := range n.Lhs {
					if sel, ok := l.(*ast.SelectorExpr); ok {
						for typ := range mentioned {
							s.set(typ, sel.Sel.Name)
						}
					}
				}
			}
			return true
		})
	}
	return s
}

func (s surface) set(typ, field string) {
	if s.sets[typ] == nil {
		s.sets[typ] = map[string]bool{}
	}
	s.sets[typ][field] = true
}

// reaches reports whether the surface fills d: it calls d.method, and the key
// needs nothing more, or the surface sets the struct field, or some call passes
// a value other than a zero literal in the argument.
func (s surface) reaches(d dep) bool {
	calls := s.calls[d.method]
	switch {
	case len(calls) == 0:
		return false
	case d.field != "":
		return s.sets[d.typ][d.field]
	case d.arg < 0:
		return true
	}
	for _, args := range calls {
		end := min(d.arg+1, len(args))
		if d.tail {
			end = len(args) // a variadic tail is one argument
		}
		for _, a := range args[min(d.arg, len(args)):end] {
			if !zeroLiteral(a) {
				return true
			}
		}
	}
	return false
}

func zeroLiteral(e ast.Expr) bool {
	switch e := e.(type) {
	case *ast.BasicLit:
		return e.Value == `""` || e.Value == "``" || e.Value == "0"
	case *ast.Ident:
		return e.Name == "nil" || e.Name == "false"
	}
	return false
}
